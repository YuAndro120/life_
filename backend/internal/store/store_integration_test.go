package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Требует мигрированную БД, заполненную `go run ./cmd/seed -yes`; запуск: `make test-integration`.
func openTestStore(t *testing.T) *Postgres {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL не задан")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return New(pool)
}

func TestFeedFromSeededDB(t *testing.T) {
	s := openTestStore(t)
	feed, err := s.Feed(context.Background(), time.Now().Add(-36*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Stories) != 9 {
		t.Fatalf("сюжетов %d, ожидалось 9", len(feed.Stories))
	}
	if feed.Stats.AdsHidden != 7 {
		t.Errorf("ads_hidden = %d", feed.Stats.AdsHidden)
	}
	if feed.Stats.PostsTotal < feed.Stats.AdsHidden {
		t.Errorf("posts_total %d меньше ads_hidden %d", feed.Stats.PostsTotal, feed.Stats.AdsHidden)
	}
	for i, st := range feed.Stories {
		if len(st.Sources) == 0 || len(st.Sources) != st.SourceCount {
			t.Errorf("сюжет %s: источников %d, source_count %d", st.ID, len(st.Sources), st.SourceCount)
		}
		if i > 0 && st.UpdatedAt.After(feed.Stories[i-1].UpdatedAt) {
			t.Errorf("сюжеты не по убыванию updated_at на позиции %d", i)
		}
		if feed.GeneratedAt.Before(st.UpdatedAt) {
			t.Errorf("generated_at раньше updated_at сюжета %s", st.ID)
		}
	}
}

func TestFeedWindowExcludesOldStories(t *testing.T) {
	s := openTestStore(t)
	feed, err := s.Feed(context.Background(), time.Now().Add(-4*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Stories) == 0 || len(feed.Stories) >= 9 {
		t.Fatalf("окно 4 часа дало %d сюжетов", len(feed.Stories))
	}
}

func TestLawsOnlyVerifiedAndInRange(t *testing.T) {
	s := openTestStore(t)
	today := time.Now().UTC().Truncate(24 * time.Hour)
	laws, err := s.Laws(context.Background(), today, today.AddDate(1, 0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if len(laws.Laws) != 6 {
		t.Fatalf("законов %d, ожидалось 6 (непроверенный не должен попасть)", len(laws.Laws))
	}
	for _, l := range laws.Laws {
		if l.VerifiedAt == nil {
			t.Errorf("%s: нет verified_at", l.ID)
		}
		if l.Actions == nil || l.AudienceTags == nil {
			t.Errorf("%s: actions и audience_tags должны быть [] а не null", l.ID)
		}
	}
	narrow, err := s.Laws(context.Background(), today, today.AddDate(0, 0, 10))
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range narrow.Laws {
		if l.Dates.Effective != nil && *l.Dates.Effective > today.AddDate(0, 0, 10).Format("2006-01-02") {
			t.Errorf("%s вне диапазона: %s", l.ID, *l.Dates.Effective)
		}
	}
}

func TestPing(t *testing.T) {
	if err := openTestStore(t).Ping(context.Background()); err != nil {
		t.Fatal(err)
	}
}
