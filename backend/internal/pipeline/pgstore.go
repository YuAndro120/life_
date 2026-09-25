package pipeline

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"shtil/backend/internal/cluster"
	"shtil/backend/internal/db/sqlcgen"
	"shtil/backend/internal/llm"
)

type PGStore struct{ q *sqlcgen.Queries }

func NewPGStore(pool *pgxpool.Pool) *PGStore { return &PGStore{q: sqlcgen.New(pool)} }

func ts(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }

func text(s string) pgtype.Text { return pgtype.Text{String: s, Valid: s != ""} }

func (s *PGStore) UnclusteredPosts(ctx context.Context, since time.Time) ([]cluster.Doc, error) {
	rows, err := s.q.ListUnclusteredPosts(ctx, ts(since))
	if err != nil {
		return nil, err
	}
	out := make([]cluster.Doc, 0, len(rows))
	for _, r := range rows {
		out = append(out, cluster.Doc{ID: r.ID, SourceID: r.SourceID, Text: r.Text, Lang: r.Lang, At: r.PublishedAt.Time})
	}
	return out, nil
}

func (s *PGStore) OpenStories(ctx context.Context, since time.Time) ([]cluster.Story, error) {
	rows, err := s.q.ListOpenStoryPosts(ctx, ts(since))
	if err != nil {
		return nil, err
	}
	var out []cluster.Story
	idx := map[int64]int{}
	for _, r := range rows {
		if !r.StoryID.Valid {
			continue
		}
		i, ok := idx[r.StoryID.Int64]
		if !ok {
			i = len(out)
			idx[r.StoryID.Int64] = i
			out = append(out, cluster.Story{ID: r.StoryID.Int64})
		}
		out[i].Docs = append(out[i].Docs, cluster.Doc{ID: r.ID, SourceID: r.SourceID, Text: r.Text, Lang: r.Lang, At: r.PublishedAt.Time})
	}
	return out, nil
}

func (s *PGStore) CreateStory(ctx context.Context, firstSeen time.Time) (int64, error) {
	return s.q.CreateStory(ctx, sqlcgen.CreateStoryParams{FirstSeenAt: ts(firstSeen), UpdatedAt: ts(firstSeen)})
}

func (s *PGStore) AttachPost(ctx context.Context, storyID, postID int64) error {
	if err := s.q.AttachPost(ctx, sqlcgen.AttachPostParams{StoryID: pgtype.Int8{Int64: storyID, Valid: true}, PostID: postID}); err != nil {
		return err
	}
	return s.q.AttachStoryPost(ctx, sqlcgen.AttachStoryPostParams{StoryID: storyID, PostID: postID})
}

func (s *PGStore) Recount(ctx context.Context, storyID int64) error {
	return s.q.RecountStory(ctx, storyID)
}

func (s *PGStore) StoriesForDigest(ctx context.Context, quietBefore time.Time, maxAttempts, batch int) ([]DigestCandidate, error) {
	rows, err := s.q.ListStoriesForDigest(ctx, sqlcgen.ListStoriesForDigestParams{
		QuietBefore: ts(quietBefore), MaxAttempts: int32(maxAttempts), Batch: int32(batch),
	})
	if err != nil {
		return nil, err
	}
	out := make([]DigestCandidate, 0, len(rows))
	for _, r := range rows {
		out = append(out, DigestCandidate{ID: r.ID, PostCount: int(r.PostCount)})
	}
	return out, nil
}

func (s *PGStore) StoryPosts(ctx context.Context, storyID int64, limit int) ([]llm.Post, error) {
	rows, err := s.q.ListStoryPostsForDigest(ctx, sqlcgen.ListStoryPostsForDigestParams{StoryID: pgtype.Int8{Int64: storyID, Valid: true}, Lim: int32(limit)})
	if err != nil {
		return nil, err
	}
	out := make([]llm.Post, 0, len(rows))
	for _, r := range rows {
		out = append(out, llm.Post{SourceTitle: r.SourceTitle, SourceKind: r.SourceKind, Lang: r.Lang, URL: r.Url, PublishedAt: r.PublishedAt.Time, Text: r.Text})
	}
	return out, nil
}

func (s *PGStore) SaveDigest(ctx context.Context, storyID int64, d llm.Digest, at time.Time) error {
	return s.q.SaveStoryDigest(ctx, sqlcgen.SaveStoryDigestParams{
		ID: storyID, Topic: text(d.Topic), InfoType: text(d.InfoType), Heaviness: text(d.Heaviness),
		Title: text(d.Title), Summary: text(d.Summary), Meaning: text(d.Meaning), RegionCode: text(d.RegionCode), At: ts(at), Newsworthy: d.Newsworthy,
	})
}

func (s *PGStore) RecordDigestFailure(ctx context.Context, storyID int64, msg string) error {
	if len(msg) > 300 {
		msg = msg[:300]
	}
	return s.q.RecordDigestFailure(ctx, sqlcgen.RecordDigestFailureParams{ID: storyID, Error: msg})
}

func dayOf(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t.UTC().Truncate(24 * time.Hour), Valid: true}
}

func (s *PGStore) TokensUsed(ctx context.Context, day time.Time) (int64, error) {
	return s.q.TokensUsedToday(ctx, dayOf(day))
}

func (s *PGStore) AddTokens(ctx context.Context, day time.Time, prompt, completion int) error {
	return s.q.AddTokenUsage(ctx, sqlcgen.AddTokenUsageParams{Day: dayOf(day), Prompt: int64(prompt), Completion: int64(completion)})
}

func (s *PGStore) Publish(ctx context.Context) (int64, int64, error) {
	p, err := s.q.PublishReadyStories(ctx)
	if err != nil {
		return 0, 0, err
	}
	u, err := s.q.UnpublishWeakStories(ctx)
	return p, u, err
}

// RecentPosts — для `worker debug-clusters`.
type RecentPost struct {
	cluster.Doc
	SourceTitle string
	SourceKind  string
	StoryID     int64
}

func (s *PGStore) RecentPosts(ctx context.Context, since time.Time) ([]RecentPost, error) {
	rows, err := s.q.ListRecentPosts(ctx, ts(since))
	if err != nil {
		return nil, err
	}
	out := make([]RecentPost, 0, len(rows))
	for _, r := range rows {
		out = append(out, RecentPost{
			Doc:         cluster.Doc{ID: r.ID, SourceID: r.SourceID, Text: r.Text, Lang: r.Lang, At: r.PublishedAt.Time},
			SourceTitle: r.SourceTitle, SourceKind: r.SourceKind, StoryID: r.StoryID.Int64,
		})
	}
	return out, nil
}
