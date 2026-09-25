package llm

import (
	"strings"
	"testing"
)

func goodLaw() LawDraft {
	return LawDraft{
		Relevant:     true,
		Title:        "Меняется срок уведомлений об авансовых платежах для ИП",
		WhatChanged:  "Срок уведомления об авансовых платежах сокращается до 20 числа.",
		WhoAffected:  "Индивидуальные предприниматели на упрощённой системе.",
		AudienceTags: []string{"work:ip_usn"},
		EffectiveAt:  "2026-10-01",
		Quotes: Quotes{
			WhatChanged:  "Срок уведомления об исчисленных суммах авансовых платежей сокращается до 20 числа.",
			WhoAffected:  "Индивидуальные предприниматели на упрощённой системе подают уведомление.",
			EffectiveDay: "Настоящий Федеральный закон вступает в силу с 1 октября 2026 года.",
		},
	}
}

func TestLawValidateAccepts(t *testing.T) {
	if err := goodLaw().Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (LawDraft{Relevant: false}).Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestLawValidateRejects(t *testing.T) {
	cases := map[string]func(*LawDraft){
		"нет фрагмента «что»":  func(d *LawDraft) { d.Quotes.WhatChanged = "" },
		"нет фрагмента «кого»": func(d *LawDraft) { d.Quotes.WhoAffected = "" },
		"тег не из списка":     func(d *LawDraft) { d.AudienceTags = []string{"work:pilot"} },
		"нет тегов":            func(d *LawDraft) { d.AudienceTags = nil },
		"дата не ISO":          func(d *LawDraft) { d.EffectiveAt = "1 октября 2026" },
		"несуществующая дата":  func(d *LawDraft) { d.EffectiveAt = "2026-02-31" },
		"дата без фрагмента":   func(d *LawDraft) { d.Quotes.EffectiveDay = "" },
		"дата не из фрагмента": func(d *LawDraft) { d.EffectiveAt = "2027-10-01" },
		"восклицание": func(d *LawDraft) {
			d.Title = "Срочно меняются правила для всех предпринимателей!"
		},
		"слишком длинное поле": func(d *LawDraft) { d.WhatChanged = strings.Repeat("а", 500) },
		"слишком много действий": func(d *LawDraft) {
			d.Actions = []string{"первое дело", "второе дело", "третье дело", "четвёртое дело", "пятое дело"}
		},
	}
	for name, mod := range cases {
		d := goodLaw()
		mod(&d)
		if err := d.Validate(); err == nil {
			t.Errorf("%s: ожидалась ошибка", name)
		}
	}
}

func TestDateInFragment(t *testing.T) {
	cases := []struct {
		iso, frag string
		want      bool
	}{
		{"2026-10-01", "вступает в силу с 1 октября 2026 года", true},
		{"2026-10-01", "вступает в силу с 01.10.2026", true},
		{"2026-10-01", "вступает в силу с 1 октября 2026 года", true},
		{"2026-10-01", "вступает в силу с 11 октября 2026 года", false},
		{"2026-10-01", "вступает в силу с 1 ноября 2026 года", false},
		{"2026-10-01", "вступает в силу с 1 октября 2027 года", false},
		{"2026-03-15", "с 15 марта 2026 года", true},
		{"мусор", "с 15 марта 2026 года", false},
	}
	for _, c := range cases {
		if got := DateInFragment(c.iso, c.frag); got != c.want {
			t.Errorf("%s в %q: %v, ждали %v", c.iso, c.frag, got, c.want)
		}
	}
}

func TestSplitFragmentsKeepsTextAndLimit(t *testing.T) {
	text := "Статья 1. Внести изменения в кодекс. Статья 2. Настоящий закон вступает в силу с 1 октября 2026 года. " + strings.Repeat("Длинное предложение без точек ", 30)
	fr := SplitFragments(text, 80)
	if len(fr) < 4 {
		t.Fatalf("мало фрагментов: %d", len(fr))
	}
	var joined []string
	for i, f := range fr {
		if f.N != i+1 {
			t.Fatalf("нумерация: %d на месте %d", f.N, i+1)
		}
		if n := len([]rune(f.Text)); n > 80 {
			t.Fatalf("фрагмент %d длиннее лимита: %d", f.N, n)
		}
		joined = append(joined, f.Text)
	}
	all := strings.Join(strings.Fields(strings.Join(joined, " ")), "")
	if !strings.HasPrefix(all, strings.Join(strings.Fields("Статья 1. Внести изменения в кодекс."), "")) {
		t.Fatal("текст потерян")
	}
	found := false
	for _, f := range fr {
		if strings.Contains(f.Text, "вступает в силу с 1 октября 2026 года") {
			found = true
		}
	}
	if !found {
		t.Fatal("фраза о вступлении в силу разорвана между фрагментами")
	}
}

func TestFragmentsForModelKeepsHeadAndTail(t *testing.T) {
	var fr []Fragment
	for i := 1; i <= 100; i++ {
		fr = append(fr, Fragment{N: i, Text: strings.Repeat("а", 100)})
	}
	got := FragmentsForModel(fr, 500, 300)
	if got[0].N != 1 || got[len(got)-1].N != 100 || len(got) >= 100 {
		t.Fatalf("неверная выдержка: %d фрагментов", len(got))
	}
	if len(FragmentsForModel(fr[:3], 500, 300)) != 3 {
		t.Fatal("короткий текст не должен резаться")
	}
}
