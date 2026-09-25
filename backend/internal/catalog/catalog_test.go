package catalog

import (
	"os"
	"strings"
	"testing"
)

func parse(t *testing.T, yaml string) ([]Entry, error) {
	t.Helper()
	return Parse(strings.NewReader(yaml))
}

func TestActiveRequiresLegalCheck(t *testing.T) {
	entries, err := parse(t, `
sources:
  - {kind: rss, handle: checked, url: "https://a.test/rss", title: A, active: true, legal_checked: 2026-09-25}
  - {kind: rss, handle: unchecked, url: "https://b.test/rss", title: B, active: true}
  - {kind: rss, handle: disabled, url: "https://c.test/rss", title: C, active: false, legal_checked: 2026-09-25}
`)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"checked": true, "unchecked": false, "disabled": false}
	for _, e := range entries {
		got, reason := e.EffectiveActive()
		if got != want[e.Handle] {
			t.Errorf("%s: active=%v (%s)", e.Handle, got, reason)
		}
		if !got && reason == "" {
			t.Errorf("%s: у выключенного источника должна быть причина", e.Handle)
		}
	}
}

func TestTelegramURLIsDerived(t *testing.T) {
	entries, err := parse(t, "sources:\n  - {kind: tg, handle: SomeChannel, title: X, active: false}\n")
	if err != nil {
		t.Fatal(err)
	}
	if entries[0].URL != "https://t.me/s/SomeChannel" {
		t.Fatalf("%q", entries[0].URL)
	}
}

func TestValidation(t *testing.T) {
	bad := map[string]string{
		"неизвестный kind":     `{kind: podcast, handle: a, url: "https://a.test", title: A}`,
		"пустой title":         `{kind: rss, handle: a, url: "https://a.test", title: ""}`,
		"плохой handle":        `{kind: rss, handle: "a b", url: "https://a.test", title: A}`,
		"не http url":          `{kind: rss, handle: a, url: "ftp://a.test", title: A}`,
		"tg не на t.me":        `{kind: tg, handle: a, url: "https://a.test/s/a", title: A}`,
		"неизвестная тема":     `{kind: rss, handle: a, url: "https://a.test", title: A, topic_hint: weather}`,
		"плохая дата проверки": `{kind: rss, handle: a, url: "https://a.test", title: A, legal_checked: "вчера"}`,
		"неизвестное поле":     `{kind: rss, handle: a, url: "https://a.test", title: A, actve: true}`,
	}
	for name, entry := range bad {
		if _, err := parse(t, "sources:\n  - "+entry+"\n"); err == nil {
			t.Errorf("%s: ожидалась ошибка", name)
		}
	}
}

func TestDuplicatesRejected(t *testing.T) {
	_, err := parse(t, `
sources:
  - {kind: rss, handle: a, url: "https://a.test/1", title: A}
  - {kind: rss, handle: a, url: "https://a.test/2", title: A2}
`)
	if err == nil {
		t.Fatal("дубликат kind/handle должен отклоняться")
	}
}

// Реальный каталог проекта должен быть корректным, а непроверенные источники — выключенными.
func TestRealCatalog(t *testing.T) {
	entries, err := Load("../../seeds/sources.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) < 10 {
		t.Fatalf("в каталоге %d источников", len(entries))
	}
	for _, e := range entries {
		active, _ := e.EffectiveActive()
		if _, checked := e.LegalCheckedAt(); active && !checked {
			t.Errorf("%s активен без проверки по реестрам", e.Handle)
		}
	}
	if _, err := os.Stat("../../seeds/sources.yaml"); err != nil {
		t.Fatal(err)
	}
}

func TestCountryAndLangValidation(t *testing.T) {
	ok, err := parse(t, "sources:\n  - {kind: rss, handle: g, url: \"https://a.test/rss\", title: G, country: GB, lang: en}\n  - {kind: rss, handle: r, url: \"https://b.test/rss\", title: R}\n")
	if err != nil {
		t.Fatal(err)
	}
	if ok[0].Country != "GB" || ok[0].Lang != "en" || ok[1].Lang != "ru" || ok[1].Country != "" {
		t.Errorf("%+v", ok)
	}
	for name, entry := range map[string]string{
		"страна строчными":        `{kind: rss, handle: a, url: "https://a.test", title: A, country: gb}`,
		"страна не двухбуквенная": `{kind: rss, handle: a, url: "https://a.test", title: A, country: USA}`,
		"язык заглавными":         `{kind: rss, handle: a, url: "https://a.test", title: A, lang: EN}`,
	} {
		if _, err := parse(t, "sources:\n  - "+entry+"\n"); err == nil {
			t.Errorf("%s: ожидалась ошибка", name)
		}
	}
}

func TestSpaceTopicHintAllowed(t *testing.T) {
	if _, err := parse(t, "sources:\n  - {kind: gov, handle: nasa, url: \"https://www.nasa.gov/feed/\", title: NASA, topic_hint: space, country: US, lang: en}\n"); err != nil {
		t.Fatal(err)
	}
}
