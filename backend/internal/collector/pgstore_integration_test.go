package collector

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"shtil/backend/internal/db/sqlcgen"
)

// Требует мигрированную БД: `make test-integration`. Использует свою запись источника и удаляет её.
func TestPGStoreEndToEnd(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL не задан")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	q := sqlcgen.New(pool)

	const handle = "itest-collector"
	cleanup := func() {
		_, _ = pool.Exec(ctx, `DELETE FROM posts WHERE source_id IN (SELECT id FROM sources WHERE handle = $1)`, handle)
		_, _ = pool.Exec(ctx, `DELETE FROM sources WHERE handle = $1`, handle)
	}
	cleanup()
	defer cleanup()

	checked := pgtype.Date{Time: time.Now(), Valid: true}
	if err := q.UpsertSource(ctx, sqlcgen.UpsertSourceParams{
		Kind: "rss", Handle: handle, Url: "https://itest.test/feed", Title: "itest", LegalStatus: "ok", LegalCheckedAt: checked, Active: true,
	}); err != nil {
		t.Fatal(err)
	}

	// Правило проекта в БД: без даты проверки активным быть нельзя.
	err = q.UpsertSource(ctx, sqlcgen.UpsertSourceParams{
		Kind: "rss", Handle: "itest-unchecked", Url: "https://itest.test/x", Title: "x", LegalStatus: "ok", Active: true,
	})
	if err == nil {
		_, _ = pool.Exec(ctx, `DELETE FROM sources WHERE handle = 'itest-unchecked'`)
		t.Fatal("БД приняла активный источник без legal_checked_at")
	}

	f := &fakeFetcher{bodies: map[string]string{"https://itest.test/feed": rssBody}}
	r := NewRunner(NewPGStore(pool), f, slog.New(slog.NewTextHandler(io.Discard, nil)))
	r.HostDelay = 0
	r.Now = func() time.Time { return t0 }

	if _, err := r.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	var newPosts, adPosts, suspected int
	if err := pool.QueryRow(ctx, `SELECT count(*), count(*) FILTER (WHERE is_ad), count(*) FILTER (WHERE ad_suspected)
		FROM posts p JOIN sources s ON s.id = p.source_id WHERE s.handle = $1`, handle).Scan(&newPosts, &adPosts, &suspected); err != nil {
		t.Fatal(err)
	}
	if newPosts != 3 || adPosts != 1 || suspected != 1 {
		t.Fatalf("после первого прохода: постов %d, реклама %d, подозрения %d", newPosts, adPosts, suspected)
	}

	// Повтор после интервала: дубликаты по (source_id, external_id) не создаются.
	r.Now = func() time.Time { return t0.Add(time.Hour) }
	_, _ = r.RunOnce(ctx)
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM posts p JOIN sources s ON s.id = p.source_id WHERE s.handle = $1`, handle).Scan(&n); err != nil || n != 3 {
		t.Fatalf("постов в БД %d (%v)", n, err)
	}

	// Ошибки копятся, успех обнуляет счётчик.
	f.errs = map[string]error{"https://itest.test/feed": errors.New("HTTP 503")}
	r.Now = func() time.Time { return t0.Add(3 * time.Hour) }
	_, _ = r.RunOnce(ctx) // в общей БД есть и другие источники, поэтому смотрим на состояние нашего
	var failures int
	var lastErr *string
	_ = pool.QueryRow(ctx, `SELECT consecutive_failures, last_error FROM sources WHERE handle = $1`, handle).Scan(&failures, &lastErr)
	if failures != 1 || lastErr == nil || *lastErr != "HTTP 503" {
		t.Fatalf("failures=%d err=%v", failures, lastErr)
	}
	f.errs = nil
	r.Now = func() time.Time { return t0.Add(10 * time.Hour) }
	_, _ = r.RunOnce(ctx)
	_ = pool.QueryRow(ctx, `SELECT consecutive_failures, last_error FROM sources WHERE handle = $1`, handle).Scan(&failures, &lastErr)
	if failures != 0 || lastErr != nil {
		t.Fatalf("после успеха failures=%d err=%v", failures, lastErr)
	}
}
