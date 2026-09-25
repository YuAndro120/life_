package pipeline

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"shtil/backend/internal/llm"
	"shtil/backend/internal/store"
)

// Требует мигрированную БД: `make test-integration`. Создаёт свои источники и посты и удаляет их.
func TestPipelineOnPostgres(t *testing.T) {
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

	cleanup := func() {
		_, _ = pool.Exec(ctx, `DELETE FROM stories WHERE id IN (SELECT story_id FROM posts p JOIN sources s ON s.id = p.source_id WHERE s.handle LIKE 'itest-pipe-%')`)
		_, _ = pool.Exec(ctx, `DELETE FROM posts WHERE source_id IN (SELECT id FROM sources WHERE handle LIKE 'itest-pipe-%')`)
		_, _ = pool.Exec(ctx, `DELETE FROM sources WHERE handle LIKE 'itest-pipe-%'`)
	}
	cleanup()
	defer cleanup()

	now := time.Now().UTC().Truncate(time.Second)
	var gov, media, media2 int64
	mk := func(handle, kind, title string, dst *int64) {
		if err := pool.QueryRow(ctx, `INSERT INTO sources (kind, handle, url, title, legal_status, legal_checked_at, active) VALUES ($1, $2, 'https://itest.test/', $3, 'ok', CURRENT_DATE, true) RETURNING id`, kind, handle, title).Scan(dst); err != nil {
			t.Fatal(err)
		}
	}
	mk("itest-pipe-gov", "gov", "Минфин тест", &gov)
	mk("itest-pipe-a", "rss", "Издание А", &media)
	mk("itest-pipe-b", "rss", "Издание Б", &media2)
	post := func(src int64, ext string, ago time.Duration, txt string) {
		if _, err := pool.Exec(ctx, `INSERT INTO posts (source_id, external_id, url, published_at, text) VALUES ($1, $2, 'https://itest.test/'||$2, $3, $4)`, src, ext, now.Add(-ago), txt); err != nil {
			t.Fatal(err)
		}
	}
	post(gov, "1", 3*time.Hour, "Минфин предложил изменить порядок уплаты авансовых платежей для ИП на УСН")
	post(media, "1", 2*time.Hour, "Сборная России сыграла вничью с Сербией в товарищеском матче")
	post(media2, "1", 100*time.Minute, "Футболисты сборной России не смогли обыграть сербов: 1:1 в товарищеском матче")
	post(media, "2", 90*time.Minute, "В Москве отключат горячую воду в октябре: опубликован график по районам")

	l := &fakeLLM{fn: func(in llm.StoryInput, _ int) (llm.Digest, error) {
		return llm.Digest{Topic: "economy", InfoType: "official", Heaviness: "neutral", Newsworthy: true,
			Title: "Проверочный заголовок для интеграционного теста конвейера", Summary: "Проверочный пересказ для интеграционного теста конвейера обработки"}, nil
	}}
	w := New(NewPGStore(pool), l, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if _, err := w.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}

	type row struct {
		status         string
		posts, sources int
		official       bool
		hasTitle       bool
	}
	rows := map[string]row{}
	q, err := pool.Query(ctx, `SELECT DISTINCT ON (st.id) s.handle, st.status, st.post_count, st.source_count, st.has_official_source, st.title_neutral IS NOT NULL
		FROM stories st JOIN posts p ON p.story_id = st.id JOIN sources s ON s.id = p.source_id WHERE s.handle LIKE 'itest-pipe-%' ORDER BY st.id, s.handle`)
	if err != nil {
		t.Fatal(err)
	}
	defer q.Close()
	for q.Next() {
		var h string
		var r row
		if err := q.Scan(&h, &r.status, &r.posts, &r.sources, &r.official, &r.hasTitle); err != nil {
			t.Fatal(err)
		}
		rows[h] = r
	}
	if r := rows["itest-pipe-gov"]; r.status != "published" || !r.official || r.posts != 1 || !r.hasTitle {
		t.Errorf("сюжет Минфина (официальный источник): %+v", r)
	}
	// Матч: два поста из двух изданий -> один сюжет, опубликован по правилу «≥ 2 источника».
	var matchStories, matchSources int
	_ = pool.QueryRow(ctx, `SELECT count(DISTINCT p.story_id), max(st.source_count) FROM posts p JOIN stories st ON st.id = p.story_id JOIN sources s ON s.id = p.source_id
		WHERE s.handle LIKE 'itest-pipe-%' AND p.text LIKE '%сборной России%'`).Scan(&matchStories, &matchSources)
	if matchStories != 1 || matchSources != 2 {
		t.Errorf("матч: сюжетов %d, источников %d", matchStories, matchSources)
	}
	// Вода: один пост одного издания -> черновик, в ленту не попадает.
	var waterStatus string
	_ = pool.QueryRow(ctx, `SELECT st.status FROM posts p JOIN stories st ON st.id = p.story_id WHERE p.text LIKE 'В Москве отключат%' AND p.source_id = $1`, media).Scan(&waterStatus)
	if waterStatus != "draft" {
		t.Errorf("сюжет из одного неофициального источника должен остаться черновиком, статус %q", waterStatus)
	}
	// В ленте API: два опубликованных, черновика нет.
	feed, err := store.New(pool).Feed(ctx, now.Add(-24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, st := range feed.Stories {
		if st.Title == "Проверочный заголовок для интеграционного теста конвейера" {
			found++
		}
	}
	if found < 2 {
		t.Errorf("в ленте должно быть >= 2 проверочных сюжета, найдено %d", found)
	}

	// Повторный проход не создаёт дублей и не вызывает модель заново.
	calls := l.calls
	if _, err := w.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if l.calls != calls {
		t.Errorf("повторный проход вызвал модель ещё %d раз", l.calls-calls)
	}
}
