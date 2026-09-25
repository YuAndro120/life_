package llm

import (
	"strings"
	"testing"
)

const lawText = "Статья 1. Внести в Налоговый кодекс изменения: срок уведомления об исчисленных суммах авансовых платежей сокращается до 20 числа. " +
	"Статья 2. Настоящий Федеральный закон вступает в силу с 1 октября 2026 года. Индивидуальные предприниматели на упрощённой системе подают уведомление."

func goodLaw() LawDraft {
	return LawDraft{
		Relevant:     true,
		Title:        "Меняется срок уведомлений об авансовых платежах для ИП",
		WhatChanged:  "Срок уведомления об авансовых платежах сокращается до 20 числа.",
		WhoAffected:  "Индивидуальные предприниматели на упрощённой системе.",
		AudienceTags: []string{"work:ip_usn"},
		EffectiveAt:  "2026-10-01",
		Quotes: Quotes{
			WhatChanged:  "срок уведомления об исчисленных суммах авансовых платежей сокращается до 20 числа",
			WhoAffected:  "Индивидуальные предприниматели на упрощённой системе подают уведомление",
			EffectiveDay: "вступает в силу с 1 октября 2026 года",
		},
	}
}

func TestLawValidateAccepts(t *testing.T) {
	if err := goodLaw().Validate(lawText); err != nil {
		t.Fatal(err)
	}
}

func TestLawValidateIrrelevantNeedsNothingElse(t *testing.T) {
	if err := (LawDraft{Relevant: false}).Validate(lawText); err != nil {
		t.Fatal(err)
	}
}

func TestLawValidateRejects(t *testing.T) {
	cases := map[string]func(*LawDraft){
		"выдуманная цитата": func(d *LawDraft) {
			d.Quotes.WhatChanged = "штрафы для всех предпринимателей увеличиваются вдвое"
		},
		"пустая цитата":       func(d *LawDraft) { d.Quotes.WhoAffected = "" },
		"тег не из списка":    func(d *LawDraft) { d.AudienceTags = []string{"work:pilot"} },
		"нет тегов":           func(d *LawDraft) { d.AudienceTags = nil },
		"дата не ISO":         func(d *LawDraft) { d.EffectiveAt = "1 октября 2026" },
		"несуществующая дата": func(d *LawDraft) { d.EffectiveAt = "2026-02-31" },
		"дата без цитаты":     func(d *LawDraft) { d.Quotes.EffectiveDay = "" },
		"цитата даты выдумана": func(d *LawDraft) {
			d.Quotes.EffectiveDay = "вступает в силу со дня подписания"
		},
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
		if err := d.Validate(lawText); err == nil {
			t.Errorf("%s: ожидалась ошибка", name)
		}
	}
}

func TestQuoteMatchIgnoresCaseSpacesAndQuotes(t *testing.T) {
	d := goodLaw()
	d.Quotes.WhatChanged = "СРОК УВЕДОМЛЕНИЯ  об исчисленных\nсуммах авансовых платежей сокращается до 20 числа"
	if err := d.Validate(lawText); err != nil {
		t.Fatal(err)
	}
}

func TestExcerptKeepsHeadAndTail(t *testing.T) {
	long := strings.Repeat("а", 100) + strings.Repeat("б", 100)
	got := ExcerptForModel(long, 10, 10)
	if !strings.HasPrefix(got, strings.Repeat("а", 10)) || !strings.HasSuffix(got, strings.Repeat("б", 10)) || !strings.Contains(got, "[...]") {
		t.Fatalf("неверная выдержка: %q", got)
	}
	if ExcerptForModel("коротко", 10, 10) != "коротко" {
		t.Fatal("короткий текст не должен меняться")
	}
}
