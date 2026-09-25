// Package pipeline обрабатывает собранные посты: склейка в сюжеты, пересказ моделью, публикация.
package pipeline

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"shtil/backend/internal/cluster"
	"shtil/backend/internal/llm"
)

// DigestCandidate — сюжет, которому пора делать пересказ.
type DigestCandidate struct {
	ID        int64
	PostCount int
}

type Store interface {
	// Склейка
	UnclusteredPosts(ctx context.Context, since time.Time) ([]cluster.Doc, error)
	OpenStories(ctx context.Context, since time.Time) ([]cluster.Story, error)
	CreateStory(ctx context.Context, firstSeen time.Time) (int64, error)
	AttachPost(ctx context.Context, storyID, postID int64) error
	Recount(ctx context.Context, storyID int64) error
	// Пересказ
	StoriesForDigest(ctx context.Context, quietBefore time.Time, maxAttempts, batch int) ([]DigestCandidate, error)
	StoryPosts(ctx context.Context, storyID int64, limit int) ([]llm.Post, error)
	SaveDigest(ctx context.Context, storyID int64, d llm.Digest, at time.Time) error
	RecordDigestFailure(ctx context.Context, storyID int64, msg string) error
	// Публикация: сюжет с ≥ 2 источниками или из официального источника (раздел 9 плана).
	Publish(ctx context.Context) (published, unpublished int64, err error)
}

type Worker struct {
	Store   Store
	LLM     llm.Client
	Cluster cluster.Config
	Log     *slog.Logger
	Now     func() time.Time

	// Quiet — дебаунс: пересказ делается, когда новых постов в сюжете не было столько времени.
	Quiet time.Duration
	// MaxAttempts — сколько раз пробовать пересказать сюжет, прежде чем оставить как есть.
	MaxAttempts int
	// Batch — сколько сюжетов обрабатывать за проход (ограничивает расход токенов).
	Batch int
	// PostsPerStory — сколько постов сюжета передавать модели.
	PostsPerStory int
}

func New(store Store, client llm.Client, log *slog.Logger) *Worker {
	return &Worker{
		Store: store, LLM: client, Cluster: cluster.DefaultConfig(), Log: log, Now: time.Now,
		Quiet: 10 * time.Minute, MaxAttempts: 3, Batch: 20, PostsPerStory: 6,
	}
}

type Summary struct {
	NewPosts, NewStories, Attached int
	Digested, Failed               int
	Published, Unpublished         int64
	PromptTokens, CompletionTokens int
	Stopped                        string // причина досрочной остановки пересказа (например, исчерпан лимит токенов)
}

func (w *Worker) RunOnce(ctx context.Context) (Summary, error) {
	var s Summary
	if err := w.clusterPosts(ctx, &s); err != nil {
		return s, fmt.Errorf("склейка: %w", err)
	}
	if err := w.digestStories(ctx, &s); err != nil {
		return s, fmt.Errorf("пересказ: %w", err)
	}
	pub, unpub, err := w.Store.Publish(ctx)
	if err != nil {
		return s, fmt.Errorf("публикация: %w", err)
	}
	s.Published, s.Unpublished = pub, unpub
	return s, nil
}

func (w *Worker) clusterPosts(ctx context.Context, s *Summary) error {
	now := w.Now()
	since := now.Add(-w.Cluster.Window)
	fresh, err := w.Store.UnclusteredPosts(ctx, since)
	if err != nil {
		return err
	}
	s.NewPosts = len(fresh)
	if len(fresh) == 0 {
		return nil
	}
	existing, err := w.Store.OpenStories(ctx, since.Add(-w.Cluster.Window))
	if err != nil {
		return err
	}
	assignments, _ := cluster.Assign(w.Cluster, existing, fresh)

	realID := map[int64]int64{} // временный (отрицательный) ID -> настоящий
	byDoc := map[int64]cluster.Doc{}
	for _, d := range fresh {
		byDoc[d.ID] = d
	}
	touched := map[int64]bool{}
	for _, a := range assignments {
		storyID := a.StoryID
		if a.NewStory {
			d := byDoc[a.DocID]
			id, err := w.Store.CreateStory(ctx, d.At)
			if err != nil {
				return err
			}
			realID[a.StoryID] = id
			storyID = id
			s.NewStories++
		} else {
			if a.StoryID < 0 { // присоединился к сюжету, созданному в этом же проходе
				storyID = realID[a.StoryID]
			}
			s.Attached++
		}
		if err := w.Store.AttachPost(ctx, storyID, a.DocID); err != nil {
			return err
		}
		touched[storyID] = true
	}
	for id := range touched {
		if err := w.Store.Recount(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func (w *Worker) digestStories(ctx context.Context, s *Summary) error {
	if w.LLM == nil {
		return nil
	}
	quietBefore := w.Now().Add(-w.Quiet)
	candidates, err := w.Store.StoriesForDigest(ctx, quietBefore, w.MaxAttempts, w.Batch)
	if err != nil {
		return err
	}
	for _, c := range candidates {
		if err := ctx.Err(); err != nil {
			return err
		}
		posts, err := w.Store.StoryPosts(ctx, c.ID, w.PostsPerStory)
		if err != nil {
			return err
		}
		if len(posts) == 0 {
			// Пустой сюжет (например, посты удалены вручную): засчитываем попытку и идём дальше, проход не останавливаем.
			if err := w.Store.RecordDigestFailure(ctx, c.ID, "в сюжете нет постов"); err != nil {
				return err
			}
			continue
		}
		d, usage, err := w.LLM.Digest(ctx, llm.StoryInput{Posts: posts})
		s.PromptTokens += usage.PromptTokens
		s.CompletionTokens += usage.CompletionTokens
		switch {
		case err == nil:
			if err := w.Store.SaveDigest(ctx, c.ID, d, w.Now()); err != nil {
				return err
			}
			s.Digested++
			w.Log.Info("сюжет обработан", "story", c.ID, "posts", c.PostCount, "topic", d.Topic, "type", d.InfoType, "heaviness", d.Heaviness, "title", d.Title)
		case errors.Is(err, llm.ErrInvalid):
			// Модель отвечает, но неверно: засчитываем попытку сюжету и идём дальше.
			s.Failed++
			w.Log.Warn("пересказ не удался", "story", c.ID, "err", err)
			if err := w.Store.RecordDigestFailure(ctx, c.ID, err.Error()); err != nil {
				return err
			}
		default:
			// Сеть, авторизация, лимит токенов: попытки не тратим и останавливаем проход, чтобы не жечь запросы зря.
			s.Stopped = err.Error()
			w.Log.Error("пересказ остановлен", "err", err)
			return nil
		}
	}
	return nil
}
