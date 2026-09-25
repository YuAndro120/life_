package llm

import (
	"strings"
	"testing"
)

func good() Digest {
	return Digest{
		Topic: "economy", InfoType: "official", Heaviness: "neutral",
		Title:   "Банк России сохранил ключевую ставку на уровне 16 процентов годовых",
		Summary: "Совет директоров Банка России принял решение не менять ключевую ставку. Регулятор отметил замедление инфляции.",
	}
}

func TestValidateAcceptsGoodDigest(t *testing.T) {
	if err := good().Validate(); err != nil {
		t.Fatal(err)
	}
	d := good()
	d.RegionCode, d.Meaning = "77", "Условия по вкладам заметно не изменятся."
	if err := d.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejects(t *testing.T) {
	cases := map[string]func(*Digest){
		"тема не из списка":  func(d *Digest) { d.Topic = "Экономика" },
		"тип не из списка":   func(d *Digest) { d.InfoType = "news" },
		"накал не из списка": func(d *Digest) { d.Heaviness = "scary" },
		"плохой регион":      func(d *Digest) { d.RegionCode = "Москва" },
		"короткий заголовок": func(d *Digest) { d.Title = "Ставка сохранена" },
		"восклицание": func(d *Digest) {
			d.Title = "Банк России сохранил ключевую ставку на уровне 16 процентов!"
		},
		"точка в конце": func(d *Digest) { d.Title += "." },
		"кликбейт «шок»": func(d *Digest) {
			d.Title = "Шок: Банк России сохранил ключевую ставку на уровне 16 процентов"
		},
		"кликбейт «срочно»": func(d *Digest) {
			d.Title = "Срочно Банк России сохранил ключевую ставку на уровне 16 процентов"
		},
		"КРИЧАЩИЕ слова": func(d *Digest) {
			d.Title = "БАНК РОССИИ СОХРАНИЛ ключевую ставку на уровне 16 процентов"
		},
		"пустой пересказ":          func(d *Digest) { d.Summary = "" },
		"пересказ равен заголовку": func(d *Digest) { d.Summary = d.Title },
	}
	for name, mod := range cases {
		d := good()
		mod(&d)
		if err := d.Validate(); err == nil {
			t.Errorf("%s: ожидалась ошибка", name)
		}
	}
}

func TestAbbreviationsAreNotShouting(t *testing.T) {
	d := good()
	d.Title = "ЦБ и МВД обсудили новые правила для ИП на УСН и самозанятых в России"
	if err := d.Validate(); err != nil {
		t.Fatalf("аббревиатуры допустимы: %v", err)
	}
}

func TestFactualMeaningIsAllowed(t *testing.T) {
	d := good()
	d.Meaning = "Ставка по вкладам и кредитам в ближайшие недели заметно не изменится, как отметил регулятор."
	if err := d.Validate(); err != nil {
		t.Fatalf("фактическое «значит» допустимо: %v", err)
	}
}

func TestSanitizeMeaning(t *testing.T) {
	cases := []struct {
		name, meaning string
		dropped       bool
	}{
		{"предположение «могут»", "Изменения могут повлиять на доступность валюты для бизнеса.", true},
		{"предположение «возможно»", "Возможно, вклады станут менее выгодными.", true},
		{"предположение «вероятно»", "Вероятно, ставки снизятся.", true},
		{"слишком длинное", strings.Repeat("а", 400), true},
		{"фактическое", "Ставка по вкладам и кредитам в ближайшие недели заметно не изменится.", false},
		{"слово «может» внутри другого слова не считается", "Компания Можетово объявила о слиянии.", false},
		{"пустое", "", false},
	}
	for _, c := range cases {
		d := good()
		d.Meaning = c.meaning
		reason := d.SanitizeMeaning()
		if (reason != "") != c.dropped || (c.dropped && d.Meaning != "") || (!c.dropped && d.Meaning != c.meaning) {
			t.Errorf("%s: причина=%q, значит=%q", c.name, reason, d.Meaning)
		}
	}
}
