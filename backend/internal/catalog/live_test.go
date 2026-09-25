package catalog

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"life/backend/internal/sources"
)

// Живая проверка лент каталога: LIFE_LIVE_TESTS=1 go test ./internal/catalog -run Live -v
// Только читает публичные RSS (по одному запросу на источник), ничего не сохраняет.
// Ловит смену формата или адреса ленты на стороне сайта.
func TestLiveFeedsParse(t *testing.T) {
	if os.Getenv("LIFE_LIVE_TESTS") == "" {
		t.Skip("LIFE_LIVE_TESTS не задан")
	}
	entries, err := Load("../../seeds/sources.yaml")
	if err != nil {
		t.Fatal(err)
	}
	fetcher := sources.NewHTTPFetcher("LifeBot/0.1 (catalog check)")
	for _, e := range entries {
		if e.Kind != "rss" && e.Kind != "gov" {
			continue
		}
		t.Run(e.Handle, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			body, err := fetcher.Fetch(ctx, e.URL)
			if err != nil {
				// Сайт может блокировать или тормозить: это не ошибка парсера, поэтому только пропуск.
				t.Skipf("загрузка %s не удалась: %v", e.URL, err)
			}
			posts, err := sources.ParseFeed(bytes.NewReader(body), time.Now())
			if err != nil {
				t.Fatalf("разбор: %v", err)
			}
			if len(posts) == 0 {
				t.Fatal("лента пуста")
			}
			for _, p := range posts {
				if p.ExternalID == "" || p.Text == "" || p.PublishedAt.IsZero() {
					t.Fatalf("неполный пост: %+v", p)
				}
			}
			t.Logf("%s: %d постов, первый: %.70q", e.Handle, len(posts), posts[0].Text)
		})
		time.Sleep(time.Second) // вежливость к сайтам
	}
}
