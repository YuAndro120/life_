package ads

import "testing"

func TestDetect(t *testing.T) {
	cases := []struct {
		name string
		text string
		want Verdict
	}{
		{"erid с двоеточием", "Реклама. ООО «Ромашка», ИНН 7701234567. erid: 2VtzqwP8abc", Definite},
		{"erid в ссылке", "Читайте на сайте https://example.org/?utm=1&erid=2VfnxvABC123", Definite},
		{"erid без токена не считается", "Слово erid в тексте про терминологию", None},
		{"хэштег", "Скидки на всё #реклама", Definite},
		{"на правах рекламы", "Материал опубликован на правах рекламы", Definite},
		{"реклама и ИНН", "Реклама. ИП Иванов И.И. ИНН: 770123456789", Definite},
		{"новость про рынок рекламы", "Рынок интернет-рекламы вырос на 12% за полугодие", None},
		{"реклама в начале строки", "Реклама. Открой свой бизнес с нами", Suspected},
		{"ИНН и ООО без слова реклама", "ООО «Вектор» ИНН 7702345678 предлагает курсы", Suspected},
		{"партнёрский материал", "Партнёрский материал: как выбрать ноутбук", Suspected},
		{"промокод", "Используйте промокод LIFE и получите скидку", Suspected},
		{"обычная новость", "Банк России объявил решение по ключевой ставке", None},
		{"ИНН в новости про налоги", "ФНС напомнила, что ИНН нужен каждому налогоплательщику", None},
		{"пусто", "", None},
	}
	for _, c := range cases {
		if got := Detect(c.text).Verdict; got != c.want {
			t.Errorf("%s: получено %d, ожидалось %d (%q)", c.name, got, c.want, c.text)
		}
	}
}

func TestReasonIsFilled(t *testing.T) {
	if r := Detect("erid: ABCDEF123"); r.Reason == "" || r.Verdict != Definite {
		t.Fatalf("%+v", r)
	}
	if r := Detect("обычный текст"); r.Reason != "" {
		t.Fatalf("%+v", r)
	}
}
