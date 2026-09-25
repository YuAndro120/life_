// Package ads определяет рекламные посты по обязательной маркировке (ФЗ «О рекламе»):
// пометка «Реклама», данные рекламодателя (ИНН) и токен erid.
package ads

import "regexp"

type Verdict int

const (
	// None — признаков рекламы нет.
	None Verdict = iota
	// Suspected — пост похож на рекламу, но маркировка неполная; окончательно решит LLM (фаза 4).
	Suspected
	// Definite — есть надёжный признак маркировки: пост исключается из сюжетов.
	Definite
)

type Result struct {
	Verdict Verdict
	// Reason — какой признак сработал (для логов и отладки).
	Reason string
}

// В Go `\b` учитывает только ASCII-буквы, поэтому для кириллицы границы слов задаём явно.
const (
	before = `(?:^|[^\p{L}\p{N}_])`
	after  = `(?:$|[^\p{L}\p{N}_])`
)

func word(pattern string) *regexp.Regexp {
	return regexp.MustCompile(`(?i)` + before + pattern + after)
}

var (
	eridToken   = word(`erid\s*[:=]?\s*[A-Za-z0-9]{6,}`)
	eridInURL   = regexp.MustCompile(`(?i)[?&]erid=[A-Za-z0-9]{6,}`)
	hashtagAd   = regexp.MustCompile(`(?i)#\s*реклама(?:$|[^\p{L}\p{N}_])`)
	rightsOfAd  = regexp.MustCompile(`(?i)на\s+правах\s+рекламы`)
	adWord      = word(`реклама`)
	adLineStart = regexp.MustCompile(`(?im)^\s*реклама\s*[.:—-]`)
	inn         = word(`инн\s*:?\s*\d{10}(?:\d{2})?`)
	company     = regexp.MustCompile(`(?:^|[^\p{L}\p{N}_])(?:ООО|АО|ПАО|ИП|ОАО)(?:$|[^\p{L}\p{N}_])`)
	partner     = regexp.MustCompile(`(?i)партн[её]рск(?:ий|ая|ое)\s+(?:материал|пост|интеграция)`)
	promo       = word(`промокод`)
)

// Detect применяет правила по убыванию надёжности. Слово «реклама» само по себе (например, в новости
// о рынке рекламы) рекламой не считается.
func Detect(text string) Result {
	switch {
	case eridToken.MatchString(text), eridInURL.MatchString(text):
		return Result{Definite, "erid"}
	case hashtagAd.MatchString(text):
		return Result{Definite, "#реклама"}
	case rightsOfAd.MatchString(text):
		return Result{Definite, "на правах рекламы"}
	case adWord.MatchString(text) && inn.MatchString(text):
		return Result{Definite, "реклама + ИНН"}
	case adLineStart.MatchString(text):
		return Result{Suspected, "«Реклама.» в начале строки"}
	case inn.MatchString(text) && company.MatchString(text):
		return Result{Suspected, "ИНН рекламодателя без слова «реклама»"}
	case partner.MatchString(text):
		return Result{Suspected, "партнёрский материал"}
	case promo.MatchString(text):
		return Result{Suspected, "промокод"}
	}
	return Result{None, ""}
}
