package legal

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"shtil/backend/internal/db/sqlcgen"
)

type PGStore struct {
	pool *pgxpool.Pool
	q    *sqlcgen.Queries
}

func NewPGStore(pool *pgxpool.Pool) *PGStore { return &PGStore{pool: pool, q: sqlcgen.New(pool)} }

func (s *PGStore) Queries() *sqlcgen.Queries { return s.q }

func date(t *time.Time) pgtype.Date {
	if t == nil {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: *t, Valid: true}
}

func text(v string) pgtype.Text { return pgtype.Text{String: v, Valid: v != ""} }

func (s *PGStore) Seen(ctx context.Context, eo string) (bool, error) {
	return s.q.LawSeen(ctx, pgtype.Text{String: eo, Valid: true})
}

func (s *PGStore) SaveDraft(ctx context.Context, d Draft) error {
	actions, err := json.Marshal(nonNil(d.Extract.Actions))
	if err != nil {
		return err
	}
	quotes, err := json.Marshal(d.Extract.Quotes)
	if err != nil {
		return err
	}
	signed := d.Doc.Signed
	_, err = s.q.InsertLawDraft(ctx, sqlcgen.InsertLawDraftParams{
		EoNumber:     text(d.Doc.EONumber),
		Title:        d.Extract.Title,
		WhatChanged:  d.Extract.WhatChanged,
		WhoAffected:  d.Extract.WhoAffected,
		Actions:      actions,
		AudienceTags: d.Extract.AudienceTags,
		Status:       d.Status,
		PassedAt:     date(d.PassedAt),
		SignedAt:     date(&signed),
		EffectiveAt:  date(d.EffectiveAt),
		OfficialUrl:  text(d.Doc.OfficialURL()),
		SourceUrl:    text(d.SourceURL),
		ActNumber:    text("№ " + d.Doc.Number),
		Quotes:       quotes,
	})
	return err
}

func (s *PGStore) SaveRejected(ctx context.Context, doc Doc, reason string) error {
	signed := doc.Signed
	return s.q.InsertRejectedLaw(ctx, sqlcgen.InsertRejectedLawParams{
		EoNumber:     text(doc.EONumber),
		Title:        doc.Title,
		SignedAt:     date(&signed),
		ActNumber:    text("№ " + doc.Number),
		RejectReason: text(reason),
	})
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
