package sources

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/text/encoding/charmap"
)

var now = time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)

func open(t *testing.T, name string) *os.File {
	t.Helper()
	f, err := os.Open("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

func TestParseFeedCBRWithBOM(t *testing.T) {
	posts, err := ParseFeed(open(t, "cbr.xml"), now)
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 2 {
		t.Fatalf("постов %d", len(posts))
	}
	p := posts[0]
	if p.ExternalID != "ID_9973" || p.URL != "https://www.cbr.ru/rbr/rbr_fr/" {
		t.Errorf("id/url: %q %q", p.ExternalID, p.URL)
	}
	if p.Text != "Решения Банка России в отношении участников финансового рынка" {
		t.Errorf("text: %q", p.Text)
	}
	if want := time.Date(2026, 9, 25, 9, 41, 53, 0, time.UTC); !p.PublishedAt.Equal(want) {
		t.Errorf("published %v, want %v", p.PublishedAt, want)
	}
}

func TestParseFeedTASSCDATAAndEntities(t *testing.T) {
	posts, err := ParseFeed(open(t, "tass.xml"), now)
	if err != nil {
		t.Fatal(err)
	}
	if posts[0].ExternalID != "https://tass.ru/proisshestviya/28151127" || posts[0].URL != posts[0].ExternalID {
		t.Errorf("guid/link из CDATA: %+v", posts[0])
	}
	want := "Подготовку в центре \"Воин\" прошли свыше 3,5 тыс. курсантов\nВ 2026 году в сменах приняли участие 13 тыс. человек из 55 регионов"
	if posts[1].Text != want {
		t.Errorf("text:\n%q\nwant\n%q", posts[1].Text, want)
	}
}

func TestParseAtom(t *testing.T) {
	posts, err := ParseFeed(open(t, "atom.xml"), now)
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 2 {
		t.Fatalf("постов %d", len(posts))
	}
	if posts[0].URL != "https://example.org/1" {
		t.Errorf("должна выбраться ссылка rel=alternate, получено %q", posts[0].URL)
	}
	if want := "Вышла новая версия\nСписок изменений: быстрее и & стабильнее."; posts[0].Text != want {
		t.Errorf("text %q", posts[0].Text)
	}
	if posts[1].ExternalID != "https://example.org/2" {
		t.Errorf("без id берётся ссылка: %q", posts[1].ExternalID)
	}
	if want := time.Date(2026, 9, 24, 15, 30, 0, 0, time.UTC); !posts[1].PublishedAt.Equal(want) {
		t.Errorf("published %v", posts[1].PublishedAt)
	}
}

func TestParseFeedWindows1251(t *testing.T) {
	utf := `<?xml version="1.0" encoding="windows-1251"?><rss version="2.0"><channel><item><title>Ставка снижена</title><link>https://e.org/1</link><pubDate>Fri, 25 Sep 2026 12:00:00 +0300</pubDate></item></channel></rss>`
	enc, err := charmap.Windows1251.NewEncoder().String(utf)
	if err != nil {
		t.Fatal(err)
	}
	posts, err := ParseFeed(strings.NewReader(enc), now)
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 1 || posts[0].Text != "Ставка снижена" {
		t.Fatalf("%+v", posts)
	}
}

func TestParseFeedMissingDateUsesNow(t *testing.T) {
	posts, err := ParseFeed(strings.NewReader(`<rss version="2.0"><channel><item><title>Без даты</title><link>https://e.org/x</link></item></channel></rss>`), now)
	if err != nil || len(posts) != 1 {
		t.Fatalf("%v %v", posts, err)
	}
	if !posts[0].PublishedAt.Equal(now) {
		t.Errorf("published %v", posts[0].PublishedAt)
	}
}

func TestParseFeedSkipsItemsWithoutID(t *testing.T) {
	posts, err := ParseFeed(strings.NewReader(`<rss version="2.0"><channel><item><title>Пустой</title></item></channel></rss>`), now)
	if err != nil || len(posts) != 0 {
		t.Fatalf("%v %v", posts, err)
	}
}

func TestParseFeedRejectsNonFeeds(t *testing.T) {
	for _, in := range []string{"<html><body>404</body></html>", "не xml вообще", ""} {
		if _, err := ParseFeed(strings.NewReader(in), now); err == nil {
			t.Errorf("ожидалась ошибка для %q", in)
		}
	}
}

func TestParseTelegramPreview(t *testing.T) {
	posts, err := ParseTelegramPreview(open(t, "tg.html"), "testchan", now)
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 2 {
		t.Fatalf("постов %d (пост 102 — только фото, должен быть пропущен)", len(posts))
	}
	p := posts[0]
	if p.ExternalID != "101" || p.URL != "https://t.me/testchan/101" {
		t.Errorf("id/url: %q %q", p.ExternalID, p.URL)
	}
	want := "⛔️ Новая открытая модель вышла\n\nЛицензия разрешает «коммерцию» & запуск у себя.\nПодробнее"
	if p.Text != want {
		t.Errorf("text:\n%q\nwant\n%q", p.Text, want)
	}
	if !p.PublishedAt.Equal(time.Date(2026, 9, 25, 8, 15, 0, 0, time.UTC)) {
		t.Errorf("published %v", p.PublishedAt)
	}
	if posts[1].ExternalID != "103" {
		t.Errorf("второй пост: %+v", posts[1])
	}
}

func TestParseTelegramPreviewEmptyPage(t *testing.T) {
	posts, err := ParseTelegramPreview(strings.NewReader("<html><body>Channel not found</body></html>"), "x", now)
	if err != nil || len(posts) != 0 {
		t.Fatalf("%v %v", posts, err)
	}
}

func TestHTMLToText(t *testing.T) {
	cases := map[string]string{
		"<p>Раз</p><p>Два</p>":                             "Раз\nДва",
		"Строка<br>вторая<br/>третья":                      "Строка\nвторая\nтретья",
		"a&nbsp;&nbsp;b   c":                               "a b c",
		"<script>alert(1)</script>Текст<style>x{}</style>": "Текст",
		"&lt;b&gt;не тег&lt;/b&gt;":                        "<b>не тег</b>",
		"":                                                 "",
		"<div>Один</div>\n\n\n<div>Два</div>":              "Один\n\nДва",
	}
	for in, want := range cases {
		if got := HTMLToText(in); got != want {
			t.Errorf("HTMLToText(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTruncateLongText(t *testing.T) {
	long := strings.Repeat("я", MaxTextRunes+500)
	posts, err := ParseFeed(strings.NewReader(`<rss version="2.0"><channel><item><title>T</title><link>https://e.org/l</link><description>`+long+`</description></item></channel></rss>`), now)
	if err != nil {
		t.Fatal(err)
	}
	if n := len([]rune(posts[0].Text)); n != MaxTextRunes {
		t.Fatalf("длина %d", n)
	}
	_ = bytes.MinRead
}
