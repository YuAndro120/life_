// Package collector периодически забирает посты активных источников и сохраняет их в БД.
package collector

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"sort"
	"sync"
	"time"

	"shtil/backend/internal/ads"
	"shtil/backend/internal/sources"
)

type Source struct {
	ID            int64
	Kind          string
	Handle        string
	URL           string
	Title         string
	Country       string
	Lang          string
	LastFetchedAt *time.Time
	Failures      int
}

type Post struct {
	sources.RawPost
	Lang        string
	IsAd        bool
	AdSuspected bool
}

type Store interface {
	ActiveSources(ctx context.Context) ([]Source, error)
	// InsertPost возвращает true, если пост новый (не дубликат).
	InsertPost(ctx context.Context, sourceID int64, p Post) (bool, error)
	RecordSuccess(ctx context.Context, sourceID int64, at time.Time) error
	RecordFailure(ctx context.Context, sourceID int64, at time.Time, msg string) error
}

type Runner struct {
	Store   Store
	Fetcher sources.Fetcher
	Now     func() time.Time
	Log     *slog.Logger

	// Interval — как часто опрашивать источник (план: раз в 10 минут).
	Interval time.Duration
	// HostDelay — пауза между запросами к одному хосту.
	HostDelay time.Duration
	// Concurrency — сколько хостов обрабатывать одновременно.
	Concurrency int
	// MaxPostAge — более старые посты игнорируются.
	MaxPostAge time.Duration
}

func NewRunner(store Store, fetcher sources.Fetcher, log *slog.Logger) *Runner {
	return &Runner{
		Store: store, Fetcher: fetcher, Log: log, Now: time.Now,
		Interval: 10 * time.Minute, HostDelay: time.Second, Concurrency: 4, MaxPostAge: 14 * 24 * time.Hour,
	}
}

type Summary struct {
	Sources, Failed, Fetched, New, Ads, Suspected int
}

const maxBackoff = 6 * time.Hour

// Due говорит, пора ли опрашивать источник. После ошибок интервал удваивается (до 6 часов).
func (r *Runner) Due(s Source, now time.Time) bool {
	if s.LastFetchedAt == nil {
		return true
	}
	wait := r.Interval
	for i := 0; i < s.Failures && wait < maxBackoff; i++ {
		wait *= 2
	}
	if wait > maxBackoff {
		wait = maxBackoff
	}
	return now.Sub(*s.LastFetchedAt) >= wait
}

// RunOnce опрашивает все источники, которым пора. Источники одного хоста идут по очереди с паузой,
// разные хосты — параллельно.
func (r *Runner) RunOnce(ctx context.Context) (Summary, error) {
	all, err := r.Store.ActiveSources(ctx)
	if err != nil {
		return Summary{}, fmt.Errorf("список источников: %w", err)
	}
	now := r.Now()
	byHost := map[string][]Source{}
	for _, s := range all {
		if !r.Due(s, now) {
			continue
		}
		byHost[hostOf(s.URL)] = append(byHost[hostOf(s.URL)], s)
	}

	var (
		mu    sync.Mutex
		total Summary
		wg    sync.WaitGroup
		sem   = make(chan struct{}, max(r.Concurrency, 1))
	)
	hosts := make([]string, 0, len(byHost))
	for h := range byHost {
		hosts = append(hosts, h)
	}
	sort.Strings(hosts)

	for _, host := range hosts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			for i, s := range byHost[host] {
				if ctx.Err() != nil {
					return
				}
				if i > 0 {
					select {
					case <-time.After(r.HostDelay):
					case <-ctx.Done():
						return
					}
				}
				res := r.collect(ctx, s)
				mu.Lock()
				total.Sources++
				total.Fetched += res.Fetched
				total.New += res.New
				total.Ads += res.Ads
				total.Suspected += res.Suspected
				if res.Failed {
					total.Failed++
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return total, ctx.Err()
}

type result struct {
	Fetched, New, Ads, Suspected int
	Failed                       bool
}

func (r *Runner) collect(ctx context.Context, s Source) result {
	log := r.Log.With("source", s.Handle, "kind", s.Kind)
	at := r.Now()

	posts, err := r.fetchPosts(ctx, s, at)
	if err != nil {
		log.Warn("источник не забран", "err", err, "failures", s.Failures+1)
		if rerr := r.Store.RecordFailure(ctx, s.ID, at, truncateErr(err)); rerr != nil {
			log.Error("не удалось записать ошибку источника", "err", rerr)
		}
		return result{Failed: true}
	}

	res := result{}
	cutoff := at.Add(-r.MaxPostAge)
	for _, raw := range posts {
		if raw.PublishedAt.Before(cutoff) || raw.PublishedAt.After(at.Add(24*time.Hour)) {
			continue
		}
		res.Fetched++
		verdict := ads.Detect(raw.Text)
		post := Post{RawPost: raw, Lang: langOf(s), IsAd: verdict.Verdict == ads.Definite, AdSuspected: verdict.Verdict == ads.Suspected}
		inserted, err := r.Store.InsertPost(ctx, s.ID, post)
		if err != nil {
			log.Error("пост не сохранён", "err", err, "external_id", raw.ExternalID)
			continue
		}
		if inserted {
			res.New++
			if post.IsAd {
				res.Ads++
			}
			if post.AdSuspected {
				res.Suspected++
			}
		}
	}
	if err := r.Store.RecordSuccess(ctx, s.ID, at); err != nil {
		log.Error("не удалось записать успех источника", "err", err)
	}
	log.Info("источник забран", "posts", res.Fetched, "new", res.New, "ads", res.Ads)
	return res
}

func (r *Runner) fetchPosts(ctx context.Context, s Source, now time.Time) ([]sources.RawPost, error) {
	fetchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	body, err := r.Fetcher.Fetch(fetchCtx, s.URL)
	if err != nil {
		return nil, err
	}
	switch s.Kind {
	case "rss", "gov":
		return sources.ParseFeed(bytes.NewReader(body), now)
	case "tg":
		return sources.ParseTelegramPreview(bytes.NewReader(body), s.Handle, now)
	default:
		return nil, errors.New("тип источника «" + s.Kind + "» пока не поддерживается")
	}
}

func langOf(s Source) string {
	if s.Lang == "" {
		return "ru"
	}
	return s.Lang
}

func hostOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return raw
	}
	return u.Host
}

func truncateErr(err error) string {
	msg := err.Error()
	if len(msg) > 300 {
		return msg[:300]
	}
	return msg
}
