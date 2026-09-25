package legal

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPassedDate(t *testing.T) {
	got := PassedDate("Принят Государственной Думой 23 июля 2026 года Одобрен Советом Федерации 29 июля 2026 года")
	if got == nil || got.Format("2006-01-02") != "2026-07-23" {
		t.Fatalf("получили %v", got)
	}
	if PassedDate("Принят Государственной Думой 31 февраля 2026 года") != nil {
		t.Fatal("несуществующая дата должна отклоняться")
	}
	if PassedDate("нет такой строки") != nil {
		t.Fatal("нет даты — nil")
	}
}

func TestStatusFor(t *testing.T) {
	now := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	past, future := now.AddDate(0, -1, 0), now.AddDate(0, 1, 0)
	if StatusFor(&past, now) != "in_force" || StatusFor(&future, now) != "signed" || StatusFor(nil, now) != "signed" {
		t.Fatal("неверный статус")
	}
}

const searchPage = `<h3><a href="/acts/bank/111" data-weight="1">Федеральный закон от 24.06.2025 г. № 333-ФЗ <span>Другой</span></a></h3>
<h3><a href="/acts/bank/222" data-weight="2">Федеральный закон от 04.08.2026 г. № 333-ФЗ <span>Нужный</span></a></h3>
<h3><a href="/acts/bank/333" data-weight="3">Федеральный закон от 04.08.2026 г. № 334-ФЗ <span>Соседний</span></a></h3>`

func TestFindResultMatchesNumberAndDate(t *testing.T) {
	d := Doc{Number: "333-ФЗ", Signed: time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)}
	if got := findResult(searchPage, d); got != "/acts/bank/222" {
		t.Fatalf("получили %q", got)
	}
	d.Number = "999-ФЗ"
	if findResult(searchPage, d) != "" {
		t.Fatal("несуществующий закон не должен находиться")
	}
}

func TestExtractBodyStripsMarkupAndNormalizes(t *testing.T) {
	page := `<html><script>var x=1;</script><div itemprop="articleBody"><div class="w"><p>Статья&nbsp;1.  Внести «изменения» — в кодекс.</p><p>Статья 2.</p></div></div></div></div><footer>лишнее</footer>`
	got := ExtractBody(page)
	want := `Статья 1. Внести "изменения" - в кодекс. Статья 2.`
	if got != want {
		t.Fatalf("получили %q, ждали %q", got, want)
	}
}

func TestPravoLawsPaginatesAndParses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("DocumentTypes") == "" || r.URL.Query().Get("PageSize") != "30" {
			http.Error(w, "bad", 400)
			return
		}
		page := r.URL.Query().Get("Index")
		fmt.Fprintf(w, `{"items":[{"eoNumber":"eo%s1","number":"1-ФЗ","documentDate":"2026-08-04T00:00:00","name":"\"О чём-то\""},{"eoNumber":"","number":"x","documentDate":"2026-08-04T00:00:00","name":"без номера"}],"pagesTotalCount":2}`, page)
	}))
	defer srv.Close()
	p := &Pravo{HTTP: srv.Client(), Base: srv.URL}
	docs, err := p.Laws(context.Background(), time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 || docs[0].EONumber != "eo11" || docs[1].EONumber != "eo21" || docs[0].Title != "О чём-то" {
		t.Fatalf("получили %+v", docs)
	}
}
