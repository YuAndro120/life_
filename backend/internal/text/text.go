// Package text разбирает русский и английский текст новостей на признаки для сравнения постов.
package text

import (
	"strings"
	"unicode"
)

// Term — нормализованное слово (основа) с весом: числа и имена собственные важнее обычных слов.
type Term struct {
	Stem  string
	Boost float64
}

const (
	numberBoost = 2.0
	entityBoost = 1.5
	stemLen     = 5
)

// Terms превращает текст в признаки: нижний регистр, без стоп-слов, слова обрезаны до основы.
// Числа (16, 2026, 1:1 -> «1», «1») и слова с заглавной буквы не в начале предложения получают повышенный вес.
func Terms(s string) []Term {
	var out []Term
	wordStart := true // начало предложения: заглавная буква тут ничего не значит
	for _, tok := range split(s) {
		lower := strings.ToLower(tok.text)
		first := []rune(tok.text)[0]
		boost := 1.0
		switch {
		case unicode.IsDigit(first):
			boost = numberBoost
		case !wordStart && unicode.IsUpper(first):
			boost = entityBoost // имя, организация или аббревиатура (ЦБ, МВД, УСН)
		}
		wordStart = tok.endsSentence
		if boost == 1.0 && stop[lower] {
			continue
		}
		out = append(out, Term{Stem: stem(lower), Boost: boost})
	}
	return out
}

type token struct {
	text         string
	endsSentence bool
}

// split режет на слова и числа (16,5 и 3.14 остаются одним числом), помечая границы предложений.
func split(s string) []token {
	var toks []token
	runes := []rune(s)
	i := 0
	for i < len(runes) {
		r := runes[i]
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			if (r == '.' || r == '!' || r == '?' || r == '\n') && len(toks) > 0 {
				toks[len(toks)-1].endsSentence = true
			}
			i++
			continue
		}
		j := i
		for j < len(runes) {
			c := runes[j]
			if unicode.IsLetter(c) || unicode.IsDigit(c) {
				j++
				continue
			}
			// десятичная запятая или точка внутри числа
			if (c == ',' || c == '.') && j > i && unicode.IsDigit(runes[j-1]) && j+1 < len(runes) && unicode.IsDigit(runes[j+1]) {
				j++
				continue
			}
			break
		}
		toks = append(toks, token{text: string(runes[i:j])})
		i = j
	}
	if len(toks) > 0 {
		toks[0].endsSentence = false
	}
	return toks
}

// stem — грубая основа: окончания русских слов отбрасываются обрезкой до stemLen букв.
// Этого достаточно, чтобы «ставку», «ставки», «ставка» совпали, и не нужен словарь.
func stem(w string) string {
	r := []rune(w)
	if len(r) > stemLen && unicode.Is(unicode.Cyrillic, r[0]) {
		return string(r[:stemLen])
	}
	return stemLatin(w)
}

// stemLatin — простая основа для английских слов: launches/launched/launching -> launch, rockets -> rocket.
func stemLatin(w string) string {
	for _, suf := range []string{"ing", "ed", "es", "ly", "s"} {
		if len(w) > len(suf)+3 && strings.HasSuffix(w, suf) {
			w = strings.TrimSuffix(w, suf)
			break
		}
	}
	if r := []rune(w); len(r) > 8 {
		return string(r[:8])
	}
	return w
}

var stop = func() map[string]bool {
	m := map[string]bool{}
	for _, w := range strings.Fields(`и в во не что он на я с со как а то все она так его но да ты к у же вы за бы по только ее мне было вот
от меня еще нет о из ему теперь когда даже ну вдруг ли если уже или ни быть был него до вас нибудь опять уж вам ведь там потом себя ничего
ей может они тут где есть надо ней для мы тебя их чем была сам чтоб без будто чего раз тоже себе под будет ж тогда кто этот того потому
этого какой совсем ним здесь этом один почти мой тем чтобы нее сейчас были куда зачем всех никогда можно при наконец два об другой хоть
после над больше тот через эти нас про всего них какая много разве три эту моя впрочем свою этой перед иногда чуть том нельзя такой им более
всегда конечно всю между заявил заявила сообщил сообщила сообщает отметил отметила рассказал рассказала также которые который которая
the a an of to in on for and or is are was were be by with at from that this it as said says will would has have had after over new not but its their they you can out up about more than who what when how also into his her our us one two been being just amid ahead per than then there these those while during before against between`) {
		m[w] = true
	}
	return m
}()
