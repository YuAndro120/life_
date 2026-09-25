package httpapi

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"shtil/backend/internal/api"
)

type fakeStore struct {
	feed     api.Feed
	laws     api.Laws
	err      error
	pingErr  error
	gotSince time.Time
	gotFrom  time.Time
	gotTo    time.Time
}

func (f *fakeStore) Feed(_ context.Context, since time.Time) (api.Feed, error) {
	f.gotSince = since
	return f.feed, f.err
}

func (f *fakeStore) Laws(_ context.Context, from, to time.Time) (api.Laws, error) {
	f.gotFrom, f.gotTo = from, to
	return f.laws, f.err
}

func (f *fakeStore) Ping(context.Context) error { return f.pingErr }

var fixedNow = time.Date(2026, 9, 25, 8, 0, 0, 0, time.UTC)

func newServer(store *fakeStore) http.Handler {
	s := NewServer(store)
	s.now = func() time.Time { return fixedNow }
	return s.Handler()
}

func get(h http.Handler, target string, headers ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func sampleFeed() api.Feed {
	meaning := "Условия по вкладам не изменятся."
	return api.Feed{
		GeneratedAt: fixedNow,
		Stories: []api.Story{{
			ID: "st_1", Topic: "economy", InfoType: "official", Heaviness: "neutral",
			Title: "Банк России объявил решение", Meaning: &meaning, Summary: "…",
			PostCount: 9, SourceCount: 5, Sources: []api.Source{{Title: "Банк России", URL: "https://www.cbr.ru/"}},
			UpdatedAt: fixedNow,
		}},
		Stats: api.FeedStats{PostsTotal: 38, AdsHidden: 7},
	}
}

func TestHealth(t *testing.T) {
	rec := get(newServer(&fakeStore{}), "/v1/health")
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != `{"status":"ok"}` {
		t.Fatalf("got %d %q", rec.Code, rec.Body.String())
	}
}

func TestHealthDegradedWhenDBDown(t *testing.T) {
	rec := get(newServer(&fakeStore{pingErr: errors.New("down")}), "/v1/health")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestFeedShapeAndDefaultWindow(t *testing.T) {
	store := &fakeStore{feed: sampleFeed()}
	rec := get(newServer(store), "/v1/feed")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}
	if want := fixedNow.Add(-36 * time.Hour); !store.gotSince.Equal(want) {
		t.Fatalf("since = %v, want %v", store.gotSince, want)
	}
	var raw map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	stories := raw["stories"].([]any)
	first := stories[0].(map[string]any)
	for _, key := range []string{"id", "topic", "info_type", "heaviness", "title", "meaning", "summary", "post_count", "source_count", "sources", "region_code", "updated_at"} {
		if _, ok := first[key]; !ok {
			t.Errorf("в сюжете нет поля %q", key)
		}
	}
	if first["region_code"] != nil {
		t.Errorf("region_code должен быть null, получено %v", first["region_code"])
	}
	stats := raw["stats"].(map[string]any)
	if stats["posts_total"].(float64) != 38 || stats["ads_hidden"].(float64) != 7 {
		t.Errorf("stats = %v", stats)
	}
	if got := rec.Header().Get("Cache-Control"); got != "public, max-age=300" {
		t.Errorf("Cache-Control = %q", got)
	}
}

func TestFeedExplicitSince(t *testing.T) {
	store := &fakeStore{feed: sampleFeed()}
	get(newServer(store), "/v1/feed?since=2026-09-24T19:00:00Z")
	if want := time.Date(2026, 9, 24, 19, 0, 0, 0, time.UTC); !store.gotSince.Equal(want) {
		t.Fatalf("since = %v", store.gotSince)
	}
}

func TestFeedBadSince(t *testing.T) {
	rec := get(newServer(&fakeStore{}), "/v1/feed?since=вчера")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestStoreErrorIsHiddenFromClient(t *testing.T) {
	rec := get(newServer(&fakeStore{err: errors.New("secret sql failure")}), "/v1/feed")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "secret") {
		t.Fatalf("внутренняя ошибка утекла клиенту: %s", rec.Body.String())
	}
}

func TestETagAndNotModified(t *testing.T) {
	h := newServer(&fakeStore{feed: sampleFeed()})
	first := get(h, "/v1/feed")
	etag := first.Header().Get("ETag")
	if etag == "" {
		t.Fatal("нет ETag")
	}
	if again := get(h, "/v1/feed"); again.Header().Get("ETag") != etag {
		t.Fatal("ETag нестабилен для одинакового ответа")
	}
	cached := get(h, "/v1/feed", "If-None-Match", etag)
	if cached.Code != http.StatusNotModified || cached.Body.Len() != 0 {
		t.Fatalf("status = %d, body = %d байт", cached.Code, cached.Body.Len())
	}
	if get(h, "/v1/feed", "If-None-Match", `W/"other"`).Code != http.StatusOK {
		t.Fatal("чужой ETag не должен давать 304")
	}
}

func TestETagChangesWithContent(t *testing.T) {
	store := &fakeStore{feed: sampleFeed()}
	h := newServer(store)
	before := get(h, "/v1/feed").Header().Get("ETag")
	store.feed.Stats.AdsHidden = 8
	if get(h, "/v1/feed").Header().Get("ETag") == before {
		t.Fatal("ETag не изменился при новом содержимом")
	}
}

func TestGzipForLargeBodies(t *testing.T) {
	feed := sampleFeed()
	for i := 0; i < 30; i++ {
		feed.Stories = append(feed.Stories, feed.Stories[0])
	}
	h := newServer(&fakeStore{feed: feed})

	rec := get(h, "/v1/feed", "Accept-Encoding", "gzip")
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatal("ответ не сжат")
	}
	zr, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	plain, _ := io.ReadAll(zr)
	var decoded api.Feed
	if err := json.Unmarshal(plain, &decoded); err != nil || len(decoded.Stories) != 31 {
		t.Fatalf("после распаковки: %v, %d сюжетов", err, len(decoded.Stories))
	}
	if get(h, "/v1/feed").Header().Get("Content-Encoding") != "" {
		t.Fatal("без Accept-Encoding ответ не должен сжиматься")
	}
	if get(h, "/v1/feed", "Accept-Encoding", "gzip;q=0").Header().Get("Content-Encoding") != "" {
		t.Fatal("gzip;q=0 запрещает сжатие")
	}
}

func TestErrorsAreNotCached(t *testing.T) {
	rec := get(newServer(&fakeStore{err: errors.New("x")}), "/v1/feed")
	if rec.Header().Get("ETag") != "" || rec.Header().Get("Cache-Control") != "" {
		t.Fatal("ошибки не должны кэшироваться")
	}
}

func TestLawsDefaultsAndParams(t *testing.T) {
	store := &fakeStore{laws: api.Laws{Laws: []api.Law{}}}
	h := newServer(store)
	if rec := get(h, "/v1/laws"); rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if want := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC); !store.gotFrom.Equal(want) {
		t.Fatalf("from = %v", store.gotFrom)
	}
	if want := time.Date(2027, 9, 25, 0, 0, 0, 0, time.UTC); !store.gotTo.Equal(want) {
		t.Fatalf("to = %v", store.gotTo)
	}
	get(h, "/v1/laws?from=2026-09-01&to=2027-06-30")
	if !store.gotFrom.Equal(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)) || !store.gotTo.Equal(time.Date(2027, 6, 30, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("from/to = %v %v", store.gotFrom, store.gotTo)
	}
}

func TestLawsBadParams(t *testing.T) {
	h := newServer(&fakeStore{})
	for _, target := range []string{"/v1/laws?from=01.09.2026", "/v1/laws?to=завтра", "/v1/laws?from=2027-01-01&to=2026-01-01"} {
		if rec := get(h, target); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d", target, rec.Code)
		}
	}
}

func TestLawsJSONShape(t *testing.T) {
	eff := "2026-10-01"
	store := &fakeStore{laws: api.Laws{Laws: []api.Law{{
		ID: "lw_1", Title: "Закон", WhatChanged: "Что", WhoAffected: "Кого",
		Actions: []string{"Сделать"}, AudienceTags: []string{"work:ip_usn"}, Status: "signed",
		Dates: api.LawDates{Effective: &eff},
	}}}}
	rec := get(newServer(store), "/v1/laws")
	var raw map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	law := raw["laws"].([]any)[0].(map[string]any)
	dates := law["dates"].(map[string]any)
	if dates["effective"] != "2026-10-01" || dates["signed"] != nil {
		t.Errorf("dates = %v", dates)
	}
	for _, key := range []string{"official_url", "bill_url", "act_number", "verified_at", "region_code"} {
		if v, ok := law[key]; !ok || v != nil {
			t.Errorf("%s: ожидался null, получено %v (есть=%v)", key, v, ok)
		}
	}
}

func TestUnknownRouteAndMethod(t *testing.T) {
	h := newServer(&fakeStore{})
	if get(h, "/v1/unknown").Code != http.StatusNotFound {
		t.Error("неизвестный путь должен давать 404")
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/feed", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /v1/feed: status = %d", rec.Code)
	}
}
