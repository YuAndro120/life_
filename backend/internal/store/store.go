// Package store читает данные API из PostgreSQL через код, сгенерированный sqlc.
package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"life/backend/internal/api"
	"life/backend/internal/db/sqlcgen"
)

type Postgres struct {
	pool *pgxpool.Pool
	q    *sqlcgen.Queries
}

func New(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool: pool, q: sqlcgen.New(pool)}
}

func (p *Postgres) Ping(ctx context.Context) error { return p.pool.Ping(ctx) }

func (p *Postgres) Feed(ctx context.Context, since time.Time) (api.Feed, error) {
	rows, err := p.q.ListPublishedStories(ctx, pgtype.Timestamptz{Time: since, Valid: true})
	if err != nil {
		return api.Feed{}, fmt.Errorf("list stories: %w", err)
	}
	stats, err := p.q.FeedStats(ctx, pgtype.Timestamptz{Time: since, Valid: true})
	if err != nil {
		return api.Feed{}, fmt.Errorf("feed stats: %w", err)
	}

	feed := api.Feed{
		GeneratedAt: since,
		Stories:     make([]api.Story, 0, len(rows)),
		Stats:       api.FeedStats{PostsTotal: int(stats.PostsTotal), AdsHidden: int(stats.AdsHidden)},
	}
	for _, r := range rows {
		var sources []api.Source
		if err := json.Unmarshal(r.Sources, &sources); err != nil {
			return api.Feed{}, fmt.Errorf("story %d sources: %w", r.ID, err)
		}
		if sources == nil {
			sources = []api.Source{}
		}
		updated := r.UpdatedAt.Time.UTC()
		if updated.After(feed.GeneratedAt) {
			feed.GeneratedAt = updated
		}
		feed.Stories = append(feed.Stories, api.Story{
			ID:          "st_" + strconv.FormatInt(r.ID, 10),
			Topic:       r.Topic,
			InfoType:    r.InfoType,
			Heaviness:   r.Heaviness,
			Title:       r.TitleNeutral,
			Meaning:     textPtr(r.Meaning),
			Summary:     r.Summary,
			PostCount:   int(r.PostCount),
			SourceCount: int(r.SourceCount),
			Sources:     sources,
			RegionCode:  textPtr(r.RegionCode),
			UpdatedAt:   updated,
		})
	}
	feed.GeneratedAt = feed.GeneratedAt.UTC()
	return feed, nil
}

func (p *Postgres) Laws(ctx context.Context, from, to time.Time) (api.Laws, error) {
	rows, err := p.q.ListVerifiedLaws(ctx, sqlcgen.ListVerifiedLawsParams{
		FromDate: pgtype.Date{Time: from, Valid: true},
		ToDate:   pgtype.Date{Time: to, Valid: true},
	})
	if err != nil {
		return api.Laws{}, fmt.Errorf("list laws: %w", err)
	}
	out := api.Laws{Laws: make([]api.Law, 0, len(rows))}
	for _, r := range rows {
		actions := []string{}
		if err := json.Unmarshal(r.Actions, &actions); err != nil {
			return api.Laws{}, fmt.Errorf("law %d actions: %w", r.ID, err)
		}
		tags := r.AudienceTags
		if tags == nil {
			tags = []string{}
		}
		var verified *time.Time
		if r.VerifiedAt.Valid {
			t := r.VerifiedAt.Time.UTC()
			verified = &t
		}
		out.Laws = append(out.Laws, api.Law{
			ID:           "lw_" + strconv.FormatInt(r.ID, 10),
			Title:        r.Title,
			WhatChanged:  r.WhatChanged,
			WhoAffected:  r.WhoAffected,
			Actions:      actions,
			AudienceTags: tags,
			RegionCode:   textPtr(r.RegionCode),
			Status:       r.Status,
			Dates: api.LawDates{
				Introduced: datePtr(r.IntroducedAt),
				Passed:     datePtr(r.PassedAt),
				Signed:     datePtr(r.SignedAt),
				Effective:  datePtr(r.EffectiveAt),
			},
			OfficialURL: textPtr(r.OfficialUrl),
			BillURL:     textPtr(r.BillUrl),
			ActNumber:   textPtr(r.ActNumber),
			VerifiedAt:  verified,
		})
	}
	return out, nil
}

func textPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	s := t.String
	return &s
}

func datePtr(d pgtype.Date) *string {
	if !d.Valid {
		return nil
	}
	s := d.Time.Format("2006-01-02")
	return &s
}
