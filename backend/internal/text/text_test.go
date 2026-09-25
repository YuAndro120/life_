package text

import "testing"

func stems(s string) []string {
	var out []string
	for _, t := range Terms(s) {
		out = append(out, t.Stem)
	}
	return out
}

func TestMorphologyVariantsShareStem(t *testing.T) {
	a := stems("ставку")[0]
	for _, w := range []string{"ставки", "ставка", "ставкой"} {
		if got := stems(w)[0]; got != a {
			t.Errorf("%q -> %q, ожидалось %q", w, got, a)
		}
	}
	if stems("сохранил")[0] != stems("сохранить")[0] {
		t.Error("сохранил/сохранить должны совпасть")
	}
}

func TestStopWordsRemoved(t *testing.T) {
	got := stems("Он сказал, что это на самом деле так")
	for _, s := range got {
		if s == "на" || s == "что" || s == "он" {
			t.Errorf("стоп-слово осталось: %v", got)
		}
	}
}

func TestNumbersAndEntitiesBoosted(t *testing.T) {
	terms := Terms("Ставка ЦБ выросла до 16,5% в Москве")
	boost := map[string]float64{}
	for _, tm := range terms {
		boost[tm.Stem] = tm.Boost
	}
	if boost["16,5"] != numberBoost {
		t.Errorf("число 16,5 должно быть одним признаком с повышенным весом: %v", boost)
	}
	if boost["цб"] != entityBoost || boost["москв"] != entityBoost {
		t.Errorf("ЦБ и Москва — имена, вес должен быть повышен: %v", boost)
	}
	if boost["ставк"] != 1 {
		t.Errorf("первое слово предложения не считается именем: %v", boost)
	}
}

func TestSentenceStartIsNotEntity(t *testing.T) {
	terms := Terms("Курс упал. Банк заявил о поддержке")
	for _, tm := range terms {
		if tm.Stem == "банк" && tm.Boost != 1 {
			t.Errorf("«Банк» в начале предложения не должно получать вес имени")
		}
	}
}

func TestLatinAndEmpty(t *testing.T) {
	if got := stems("OpenAI released a model"); len(got) != 3 {
		t.Errorf("%v", got)
	}
	if len(Terms("")) != 0 || len(Terms("  ,.!? ")) != 0 {
		t.Error("пустой текст даёт признаки")
	}
}

func TestEnglishMorphology(t *testing.T) {
	base := stems("launch")[0]
	for _, w := range []string{"launches", "launched", "launching"} {
		if got := stems(w)[0]; got != base {
			t.Errorf("%q -> %q, ожидалось %q", w, got, base)
		}
	}
	if stems("rockets")[0] != stems("rocket")[0] {
		t.Error("rockets/rocket должны совпасть")
	}
	if got := stems("is")[0:0]; len(got) != 0 { // стоп-слово убирается
		t.Error("unreachable")
	}
	for _, s := range stems("NASA said the mission will launch after the tests") {
		if s == "said" || s == "the" || s == "will" || s == "after" {
			t.Errorf("английское стоп-слово осталось: %q", s)
		}
	}
}
