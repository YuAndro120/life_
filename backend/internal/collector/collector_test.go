package collector

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"
)

var t0 = time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)

type fakeFetcher struct {
	mu       sync.Mutex
	bodies   map[string]string
	errs     map[string]error
	calls    []string
	callTime map[string][]time.Time
}

func (f *fakeFetcher) Fetch(_ context.Context, url string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, url)
	if f.callTime == nil {
		f.callTime = map[string][]time.Time{}
	}
	f.callTime[url] = append(f.callTime[url], time.Now())
	if err := f.errs[url]; err != nil {
		return nil, err
	}
	return []byte(f.bodies[url]), nil
}

type fakeStore struct {
	mu       sync.Mutex
	sources  []Source
	posts    map[string]Post // ключ: sourceID/externalID
	success  map[int64]time.Time
	failures map[int64][]string
}

func newStore(src ...Source) *fakeStore {
	return &fakeStore{sources: src, posts: map[string]Post{}, success: map[int64]time.Time{}, failures: map[int64][]string{}}
}

func (s *fakeStore) ActiveSources(context.Context) ([]Source, error) { return s.sources, nil }

func key(id int64, ext string) string { return string(rune('0'+id)) + "/" + ext }

func (s *fakeStore) InsertPost(_ context.Context, id int64, p Post) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := key(id, p.ExternalID)
	if _, dup := s.posts[k]; dup {
		return false, nil
	}
	s.posts[k] = p
	return true, nil
}

func (s *fakeStore) RecordSuccess(_ context.Context, id int64, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.success[id] = at
	return nil
}

func (s *fakeStore) RecordFailure(_ context.Context, id int64, _ time.Time, msg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failures[id] = append(s.failures[id], msg)
	return nil
}

func runner(store Store, f *fakeFetcher) *Runner {
	r := NewRunner(store, f, slog.New(slog.NewTextHandler(io.Discard, nil)))
	r.Now = func() time.Time { return t0 }
	r.HostDelay = 0
	return r
}

const rssBody = `<rss version="2.0"><channel>
<item><title>Обычная новость</title><link>https://a.test/1</link><guid>g1</guid><pubDate>Fri, 25 Sep 2026 11:00:00 +0300</pubDate></item>
<item><title>Реклама. ООО «Р», ИНН 7701234567. erid: 2VtzqwP8abc</title><link>https://a.test/2</link><guid>g2</guid><pubDate>Fri, 25 Sep 2026 11:05:00 +0300</pubDate></item>
<item><title>Промокод LIFE со скидкой</title><link>https://a.test/3</link><guid>g3</guid><pubDate>Fri, 25 Sep 2026 11:10:00 +0300</pubDate></item>
<item><title>Древняя новость</title><link>https://a.test/4</link><guid>g4</guid><pubDate>Mon, 01 Jan 2024 10:00:00 +0300</pubDate></item>
</channel></rss>`

func TestRunOnceStoresPostsAndMarksAds(t *testing.T) {
	store := newStore(Source{ID: 1, Kind: "rss", Handle: "a", URL: "https://a.test/feed"})
	f := &fakeFetcher{bodies: map[string]string{"https://a.test/feed": rssBody}}
	s, err := runner(store, f).RunOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if s.Sources != 1 || s.Fetched != 3 || s.New != 3 || s.Ads != 1 || s.Suspected != 1 || s.Failed != 0 {
		t.Fatalf("%+v (древний пост должен отфильтроваться по возрасту)", s)
	}
	if !store.posts["1/g2"].IsAd || store.posts["1/g1"].IsAd {
		t.Error("реклама определена неверно")
	}
	if !store.posts["1/g3"].AdSuspected || store.posts["1/g3"].IsAd {
		t.Error("промокод должен быть подозрением, а не рекламой")
	}
	if _, ok := store.posts["1/g4"]; ok {
		t.Error("пост старше MaxPostAge попал в БД")
	}
	if store.success[1] != t0 {
		t.Error("не записан успех источника")
	}
}

func TestSecondRunDoesNotDuplicate(t *testing.T) {
	store := newStore(Source{ID: 1, Kind: "rss", Handle: "a", URL: "https://a.test/feed"})
	f := &fakeFetcher{bodies: map[string]string{"https://a.test/feed": rssBody}}
	r := runner(store, f)
	if _, err := r.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	store.sources[0].LastFetchedAt = nil // принудительно «пора»
	s, _ := r.RunOnce(context.Background())
	if s.Fetched != 3 || s.New != 0 {
		t.Fatalf("%+v", s)
	}
	if len(store.posts) != 3 {
		t.Fatalf("в БД %d постов", len(store.posts))
	}
}

func TestFailureIsRecordedAndDoesNotAffectOthers(t *testing.T) {
	store := newStore(
		Source{ID: 1, Kind: "rss", Handle: "bad", URL: "https://bad.test/feed"},
		Source{ID: 2, Kind: "rss", Handle: "ok", URL: "https://ok.test/feed"},
	)
	f := &fakeFetcher{
		bodies: map[string]string{"https://ok.test/feed": rssBody},
		errs:   map[string]error{"https://bad.test/feed": errors.New("HTTP 503")},
	}
	s, _ := runner(store, f).RunOnce(context.Background())
	if s.Failed != 1 || s.New != 3 {
		t.Fatalf("%+v", s)
	}
	if len(store.failures[1]) != 1 || store.failures[1][0] != "HTTP 503" {
		t.Errorf("%v", store.failures)
	}
	if _, ok := store.success[2]; !ok {
		t.Error("исправный источник не отмечен")
	}
}

func TestGarbageBodyCountsAsFailure(t *testing.T) {
	store := newStore(Source{ID: 1, Kind: "rss", Handle: "a", URL: "https://a.test/feed"})
	f := &fakeFetcher{bodies: map[string]string{"https://a.test/feed": "<html>Cloudflare: checking your browser</html>"}}
	s, _ := runner(store, f).RunOnce(context.Background())
	if s.Failed != 1 || len(store.posts) != 0 {
		t.Fatalf("%+v", s)
	}
}

func TestUnsupportedKindFails(t *testing.T) {
	store := newStore(Source{ID: 1, Kind: "site", Handle: "s", URL: "https://s.test/"})
	s, _ := runner(store, &fakeFetcher{bodies: map[string]string{"https://s.test/": "x"}}).RunOnce(context.Background())
	if s.Failed != 1 {
		t.Fatalf("%+v", s)
	}
}

func TestTelegramSource(t *testing.T) {
	page := `<div class="tgme_widget_message" data-post="chan/7"><div class="tgme_widget_message_text">Текст поста</div><time datetime="2026-09-25T08:00:00+00:00"></time></div>`
	store := newStore(Source{ID: 1, Kind: "tg", Handle: "chan", URL: "https://t.me/s/chan"})
	f := &fakeFetcher{bodies: map[string]string{"https://t.me/s/chan": page}}
	s, _ := runner(store, f).RunOnce(context.Background())
	if s.New != 1 || store.posts["1/7"].URL != "https://t.me/chan/7" {
		t.Fatalf("%+v %+v", s, store.posts)
	}
}

func TestDueAndBackoff(t *testing.T) {
	r := runner(newStore(), &fakeFetcher{})
	at := func(ago time.Duration) *time.Time { v := t0.Add(-ago); return &v }
	cases := []struct {
		name string
		src  Source
		want bool
	}{
		{"никогда не забирался", Source{}, true},
		{"свежий", Source{LastFetchedAt: at(5 * time.Minute)}, false},
		{"интервал прошёл", Source{LastFetchedAt: at(10 * time.Minute)}, true},
		{"1 ошибка: ждём 20 минут", Source{LastFetchedAt: at(15 * time.Minute), Failures: 1}, false},
		{"1 ошибка: 20 минут прошло", Source{LastFetchedAt: at(20 * time.Minute), Failures: 1}, true},
		{"3 ошибки: ждём 80 минут", Source{LastFetchedAt: at(79 * time.Minute), Failures: 3}, false},
		{"много ошибок: потолок 6 часов", Source{LastFetchedAt: at(6 * time.Hour), Failures: 50}, true},
		{"много ошибок: до потолка ещё рано", Source{LastFetchedAt: at(5*time.Hour + 30*time.Minute), Failures: 50}, false},
	}
	for _, c := range cases {
		if got := r.Due(c.src, t0); got != c.want {
			t.Errorf("%s: %v, ожидалось %v", c.name, got, c.want)
		}
	}
}

func TestNotDueSourcesAreSkipped(t *testing.T) {
	recent := t0.Add(-time.Minute)
	store := newStore(Source{ID: 1, Kind: "rss", Handle: "a", URL: "https://a.test/feed", LastFetchedAt: &recent})
	f := &fakeFetcher{bodies: map[string]string{"https://a.test/feed": rssBody}}
	s, _ := runner(store, f).RunOnce(context.Background())
	if s.Sources != 0 || len(f.calls) != 0 {
		t.Fatalf("%+v %v", s, f.calls)
	}
}

func TestSameHostIsPolite(t *testing.T) {
	// Два канала одного хоста: между запросами должна быть пауза.
	store := newStore(
		Source{ID: 1, Kind: "tg", Handle: "one", URL: "https://t.me/s/one"},
		Source{ID: 2, Kind: "tg", Handle: "two", URL: "https://t.me/s/two"},
	)
	f := &fakeFetcher{bodies: map[string]string{"https://t.me/s/one": "<html></html>", "https://t.me/s/two": "<html></html>"}}
	r := runner(store, f)
	r.HostDelay = 120 * time.Millisecond
	if _, err := r.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	a, b := f.callTime["https://t.me/s/one"][0], f.callTime["https://t.me/s/two"][0]
	if gap := b.Sub(a); gap < 100*time.Millisecond {
		t.Fatalf("пауза между запросами к одному хосту %v, ожидалось >= 120мс", gap)
	}
}

func TestCancelledContextStops(t *testing.T) {
	store := newStore(Source{ID: 1, Kind: "rss", Handle: "a", URL: "https://a.test/feed"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f := &fakeFetcher{bodies: map[string]string{"https://a.test/feed": rssBody}}
	if _, err := runner(store, f).RunOnce(ctx); err == nil {
		t.Fatal("ожидалась ошибка отменённого контекста")
	}
	if len(f.calls) != 0 {
		t.Fatalf("запросы после отмены: %v", f.calls)
	}
}
