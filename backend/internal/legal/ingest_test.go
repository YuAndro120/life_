package legal

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"shtil/backend/internal/llm"
)

type fakeStore struct {
	seen     map[string]bool
	drafts   []Draft
	rejected map[string]string
}

func (s *fakeStore) Seen(_ context.Context, eo string) (bool, error) { return s.seen[eo], nil }
func (s *fakeStore) SaveDraft(_ context.Context, d Draft) error {
	s.drafts = append(s.drafts, d)
	return nil
}
func (s *fakeStore) SaveRejected(_ context.Context, d Doc, reason string) error {
	s.rejected[d.EONumber] = reason
	return nil
}

type fakeLLM struct {
	byNumber map[string]llm.LawDraft
	calls    int
}

func (f *fakeLLM) ExtractLaw(_ context.Context, in llm.LawInput) (llm.LawDraft, llm.Usage, error) {
	f.calls++
	d, ok := f.byNumber[in.Number]
	if !ok {
		return llm.LawDraft{}, llm.Usage{}, llm.ErrInvalid
	}
	return d, llm.Usage{PromptTokens: 100, CompletionTokens: 20}, nil
}

func TestIngestSavesDraftsRejectsAndSkipsSeen(t *testing.T) {
	body := strings.Repeat("Текст закона. ", 30)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/Documents", func(w http.ResponseWriter, r *http.Request) {
		items := []string{
			item("eo1", "1-ФЗ", "О налоге для ИП"),
			item("eo2", "2-ФЗ", "О ратификации соглашения между Россией и Ниже"),
			item("eo3", "3-ФЗ", "О техническом регламенте"),
			item("eo4", "4-ФЗ", "Уже виденный"),
			item("eo5", "5-ФЗ", "С ошибкой модели"),
		}
		fmt.Fprintf(w, `{"items":[%s],"pagesTotalCount":1}`, strings.Join(items, ","))
	})
	mux.HandleFunc("/acts/bank/search", func(w http.ResponseWriter, r *http.Request) {
		var links []string
		for i := 1; i <= 5; i++ {
			links = append(links, fmt.Sprintf(`<a href="/acts/bank/%d">Федеральный закон от 04.08.2026 г. № %d-ФЗ <span>x</span></a>`, 100+i, i))
		}
		io.WriteString(w, strings.Join(links, "\n"))
	})
	mux.HandleFunc("/acts/bank/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `<div itemprop="articleBody"><div><p>Принят Государственной Думой 21 июля 2026 года. %s</p></div></div></div></div>`, body)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	store := &fakeStore{seen: map[string]bool{"eo4": true}, rejected: map[string]string{}}
	model := &fakeLLM{byNumber: map[string]llm.LawDraft{
		"1-ФЗ": {Relevant: true, Title: "Меняется налог для ИП", AudienceTags: []string{"work:ip"}, EffectiveAt: "2027-01-01"},
		"3-ФЗ": {Relevant: false},
	}}
	now := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	g := &Ingester{
		Pravo: &Pravo{HTTP: srv.Client(), Base: srv.URL}, Kremlin: &Kremlin{HTTP: srv.Client(), Base: srv.URL},
		Extractor: model, Store: store, Log: slog.New(slog.NewTextHandler(io.Discard, nil)), Now: func() time.Time { return now },
	}
	rep, err := g.Run(context.Background(), now.AddDate(0, -3, 0), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(store.drafts) != 1 || store.drafts[0].Doc.Number != "1-ФЗ" || store.drafts[0].Status != "signed" {
		t.Fatalf("черновики: %+v", store.drafts)
	}
	if p := store.drafts[0].PassedAt; p == nil || p.Format("2006-01-02") != "2026-07-21" {
		t.Fatalf("дата принятия: %v", p)
	}
	if store.rejected["eo2"] == "" || store.rejected["eo3"] == "" {
		t.Fatalf("ратификация и «не касается» должны быть отклонены: %v", store.rejected)
	}
	if _, ok := store.rejected["eo5"]; ok {
		t.Fatal("ошибка модели не должна записываться как отклонение: повторим позже")
	}
	if model.calls != 3 { // eo1, eo3, eo5; ратификация отсеяна до модели, виденный пропущен
		t.Fatalf("вызовов модели: %d", model.calls)
	}
	if rep.Drafts != 1 || rep.Seen != 1 || rep.Failed != 1 || rep.Rejected != 2 || rep.PromptTokens != 200 {
		t.Fatalf("отчёт: %+v", rep)
	}
}

func item(eo, num, name string) string {
	return fmt.Sprintf(`{"eoNumber":%q,"number":%q,"documentDate":"2026-08-04T00:00:00","name":%q}`, eo, num, name)
}
