package collector

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"life/backend/internal/db/sqlcgen"
)

type PGStore struct{ q *sqlcgen.Queries }

func NewPGStore(pool *pgxpool.Pool) *PGStore { return &PGStore{q: sqlcgen.New(pool)} }

func (s *PGStore) ActiveSources(ctx context.Context) ([]Source, error) {
	rows, err := s.q.ListActiveSources(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Source, 0, len(rows))
	for _, r := range rows {
		src := Source{ID: r.ID, Kind: r.Kind, Handle: r.Handle, URL: r.Url, Title: r.Title, Failures: int(r.ConsecutiveFailures)}
		if r.LastFetchedAt.Valid {
			t := r.LastFetchedAt.Time
			src.LastFetchedAt = &t
		}
		out = append(out, src)
	}
	return out, nil
}

func (s *PGStore) InsertPost(ctx context.Context, sourceID int64, p Post) (bool, error) {
	n, err := s.q.InsertPost(ctx, sqlcgen.InsertPostParams{
		SourceID:    sourceID,
		ExternalID:  p.ExternalID,
		Url:         p.URL,
		PublishedAt: pgtype.Timestamptz{Time: p.PublishedAt, Valid: true},
		Text:        p.Text,
		IsAd:        p.IsAd,
		AdSuspected: p.AdSuspected,
	})
	return n > 0, err
}

func (s *PGStore) RecordSuccess(ctx context.Context, sourceID int64, at time.Time) error {
	return s.q.RecordFetchSuccess(ctx, sqlcgen.RecordFetchSuccessParams{
		FetchedAt: pgtype.Timestamptz{Time: at, Valid: true}, ID: sourceID,
	})
}

func (s *PGStore) RecordFailure(ctx context.Context, sourceID int64, at time.Time, msg string) error {
	return s.q.RecordFetchFailure(ctx, sqlcgen.RecordFetchFailureParams{
		FetchedAt: pgtype.Timestamptz{Time: at, Valid: true}, Error: msg, ID: sourceID,
	})
}
