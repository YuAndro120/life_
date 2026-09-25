// Команда seed заполняет БД тестовыми данными для разработки (фаза 2). Даты в seeds/dev.json заданы
// относительно «сейчас», поэтому «Д–6» в приложении не устаревает. ВНИМАНИЕ: очищает таблицы данных.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"life/backend/internal/config"
)

type seedFile struct {
	Sources []struct {
		Kind      string `json:"kind"`
		Handle    string `json:"handle"`
		URL       string `json:"url"`
		Title     string `json:"title"`
		TopicHint string `json:"topic_hint"`
	} `json:"sources"`
	Stories []struct {
		Topic      string   `json:"topic"`
		InfoType   string   `json:"info_type"`
		Heaviness  string   `json:"heaviness"`
		Title      string   `json:"title"`
		Meaning    *string  `json:"meaning"`
		Summary    string   `json:"summary"`
		PostCount  int      `json:"post_count"`
		Sources    []string `json:"sources"`
		HoursAgo   float64  `json:"hours_ago"`
		RegionCode *string  `json:"region_code"`
	} `json:"stories"`
	Ads struct {
		Count  int    `json:"count"`
		Source string `json:"source"`
	} `json:"ads"`
	Laws []struct {
		Title        string   `json:"title"`
		WhatChanged  string   `json:"what_changed"`
		WhoAffected  string   `json:"who_affected"`
		Actions      []string `json:"actions"`
		AudienceTags []string `json:"audience_tags"`
		RegionCode   *string  `json:"region_code"`
		Status       string   `json:"status"`
		IntroducedIn *int     `json:"introduced_days"`
		PassedIn     *int     `json:"passed_days"`
		SignedIn     *int     `json:"signed_days"`
		EffectiveIn  *int     `json:"effective_days"`
		OfficialURL  *string  `json:"official_url"`
		BillURL      *string  `json:"bill_url"`
		ActNumber    *string  `json:"act_number"`
		Verified     bool     `json:"verified"`
	} `json:"laws"`
}

func main() {
	file := flag.String("file", "seeds/dev.json", "файл с тестовыми данными")
	yes := flag.Bool("yes", false, "подтвердить очистку таблиц данных")
	flag.Parse()
	if !*yes {
		fmt.Fprintln(os.Stderr, "seed очищает sources, posts, stories, law_changes; добавьте -yes")
		os.Exit(2)
	}
	if err := run(*file); err != nil {
		slog.Error("seed", "err", err)
		os.Exit(1)
	}
}

func run(path string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var seed seedFile
	if err := json.Unmarshal(raw, &seed); err != nil {
		return fmt.Errorf("разбор %s: %w", path, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `TRUNCATE story_posts, posts, stories, sources, law_changes RESTART IDENTITY CASCADE`); err != nil {
		return err
	}

	now := time.Now().UTC()
	sourceIDs := map[string]int64{}
	sourceURLs := map[string]string{}
	for _, s := range seed.Sources {
		var id int64
		err := tx.QueryRow(ctx,
			`INSERT INTO sources (kind, handle, url, title, topic_hint, legal_checked_at) VALUES ($1, $2, $3, $4, NULLIF($5, ''), CURRENT_DATE) RETURNING id`,
			s.Kind, s.Handle, s.URL, s.Title, s.TopicHint).Scan(&id)
		if err != nil {
			return fmt.Errorf("source %s: %w", s.Handle, err)
		}
		sourceIDs[s.Handle], sourceURLs[s.Handle] = id, s.URL
	}

	seq := 0
	insertPost := func(handle string, storyID *int64, at time.Time, isAd bool) error {
		id, ok := sourceIDs[handle]
		if !ok {
			return fmt.Errorf("неизвестный источник %q", handle)
		}
		seq++
		var postID int64
		err := tx.QueryRow(ctx,
			`INSERT INTO posts (source_id, external_id, url, published_at, text, is_ad, story_id)
			 VALUES ($1, $2, $3, $4, '', $5, $6) RETURNING id`,
			id, fmt.Sprintf("seed-%d", seq), fmt.Sprintf("%s#post-%d", sourceURLs[handle], seq), at, isAd, storyID).Scan(&postID)
		if err != nil {
			return err
		}
		if storyID != nil {
			_, err = tx.Exec(ctx, `INSERT INTO story_posts (story_id, post_id) VALUES ($1, $2)`, *storyID, postID)
		}
		return err
	}

	for _, st := range seed.Stories {
		updated := now.Add(-time.Duration(st.HoursAgo * float64(time.Hour)))
		var id int64
		err := tx.QueryRow(ctx,
			`INSERT INTO stories (first_seen_at, updated_at, topic, info_type, heaviness, title_neutral, summary, meaning,
			                      post_count, source_count, region_code, status)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 'published') RETURNING id`,
			updated.Add(-2*time.Hour), updated, st.Topic, st.InfoType, st.Heaviness, st.Title, st.Summary, st.Meaning,
			st.PostCount, len(st.Sources), st.RegionCode).Scan(&id)
		if err != nil {
			return fmt.Errorf("story %q: %w", st.Title, err)
		}
		for i := 0; i < st.PostCount; i++ {
			// Посты равномерно по источникам, самый свежий совпадает с updated_at сюжета.
			handle := st.Sources[i%len(st.Sources)]
			at := updated.Add(-time.Duration(st.PostCount-1-i) * 7 * time.Minute)
			if err := insertPost(handle, &id, at, false); err != nil {
				return fmt.Errorf("story %q: %w", st.Title, err)
			}
		}
	}

	for i := 0; i < seed.Ads.Count; i++ {
		if err := insertPost(seed.Ads.Source, nil, now.Add(-time.Duration(i+1)*20*time.Minute), true); err != nil {
			return fmt.Errorf("ads: %w", err)
		}
	}

	today := now.Truncate(24 * time.Hour)
	day := func(offset *int) *time.Time {
		if offset == nil {
			return nil
		}
		d := today.AddDate(0, 0, *offset)
		return &d
	}
	for _, l := range seed.Laws {
		actions, _ := json.Marshal(l.Actions)
		if l.Actions == nil {
			actions = []byte("[]")
		}
		var verifiedAt *time.Time
		if l.Verified {
			verifiedAt = &now
		}
		tags := l.AudienceTags
		if tags == nil {
			tags = []string{}
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO law_changes (title, what_changed, who_affected, actions, audience_tags, region_code, status,
			                          introduced_at, passed_at, signed_at, effective_at,
			                          official_url, bill_url, act_number, verified, verified_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`,
			l.Title, l.WhatChanged, l.WhoAffected, actions, tags, l.RegionCode, l.Status,
			day(l.IntroducedIn), day(l.PassedIn), day(l.SignedIn), day(l.EffectiveIn),
			l.OfficialURL, l.BillURL, l.ActNumber, l.Verified, verifiedAt); err != nil {
			return fmt.Errorf("law %q: %w", l.Title, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	slog.Info("seed готов", "sources", len(seed.Sources), "stories", len(seed.Stories), "laws", len(seed.Laws))
	return nil
}
