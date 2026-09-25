// Package api описывает контракт API v1 (раздел 8 plan.md). Ответы одинаковы для всех пользователей.
package api

import (
	"context"
	"time"
)

type Source struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

type Story struct {
	ID          string    `json:"id"`
	Topic       string    `json:"topic"`
	InfoType    string    `json:"info_type"`
	Heaviness   string    `json:"heaviness"`
	Title       string    `json:"title"`
	Meaning     *string   `json:"meaning"`
	Summary     string    `json:"summary"`
	PostCount   int       `json:"post_count"`
	SourceCount int       `json:"source_count"`
	Sources     []Source  `json:"sources"`
	RegionCode  *string   `json:"region_code"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type FeedStats struct {
	PostsTotal int `json:"posts_total"`
	AdsHidden  int `json:"ads_hidden"`
}

type Feed struct {
	// GeneratedAt — время последнего изменения в ленте, а не «сейчас»: иначе ETag менялся бы на каждый запрос.
	GeneratedAt time.Time `json:"generated_at"`
	Stories     []Story   `json:"stories"`
	Stats       FeedStats `json:"stats"`
}

// LawDates хранит даты как «2006-01-02» (без времени и пояса).
type LawDates struct {
	Introduced *string `json:"introduced"`
	Passed     *string `json:"passed"`
	Signed     *string `json:"signed"`
	Effective  *string `json:"effective"`
}

type Law struct {
	ID           string     `json:"id"`
	Title        string     `json:"title"`
	WhatChanged  string     `json:"what_changed"`
	WhoAffected  string     `json:"who_affected"`
	Actions      []string   `json:"actions"`
	AudienceTags []string   `json:"audience_tags"`
	RegionCode   *string    `json:"region_code"`
	Status       string     `json:"status"`
	Dates        LawDates   `json:"dates"`
	OfficialURL  *string    `json:"official_url"`
	BillURL      *string    `json:"bill_url"`
	ActNumber    *string    `json:"act_number"`
	VerifiedAt   *time.Time `json:"verified_at"`
}

type Laws struct {
	Laws []Law `json:"laws"`
}

// Store — источник данных для обработчиков (PostgreSQL в проде, подделка в тестах).
type Store interface {
	Feed(ctx context.Context, since time.Time) (Feed, error)
	Laws(ctx context.Context, from, to time.Time) (Laws, error)
	Ping(ctx context.Context) error
}
