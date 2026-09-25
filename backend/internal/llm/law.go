package llm

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

// AudienceTags — фиксированный набор тегов аудитории (plan.md §7). region:<код> добавляется отдельно.
var AudienceTags = []string{
	"all", "gender:male", "gender:female",
	"age:u20", "age:20_25", "age:26_35", "age:36_50", "age:50p",
	"work:employee", "work:ip", "work:ip_usn", "work:selfemployed", "work:student",
	"housing:renter", "housing:owner", "housing:mortgage",
	"transport:driver", "military:registered",
}

// LawInput — закон для извлечения. Text уже нормализован (legal.Normalize).
type LawInput struct {
	Title     string
	Number    string
	Fragments []Fragment // нумерованные фрагменты текста (SplitFragments)
}

// Quotes — фрагменты закона, на которых держится каждое поле. Выбирает модель по номеру, текст подставляет код.
type Quotes struct {
	WhatChanged  string `json:"what_changed"`
	WhoAffected  string `json:"who_affected"`
	EffectiveDay string `json:"effective"`
}

// LawDraft — результат извлечения. Модель ничего не пересказывает «своими словами» сверх коротких полей
// и не выдумывает: каждое утверждение подтверждается цитатой из текста.
type LawDraft struct {
	Relevant     bool // касается граждан или ИП
	Title        string
	WhatChanged  string
	WhoAffected  string
	Actions      []string
	AudienceTags []string
	EffectiveAt  string // YYYY-MM-DD или пусто, если дата не задана календарной датой
	Quotes       Quotes
}

type LawExtractor interface {
	ExtractLaw(ctx context.Context, in LawInput) (LawDraft, Usage, error)
}

var isoDate = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// Validate проверяет черновик. Цитаты (Quotes) подставлены кодом из фрагментов закона, поэтому подлинны;
// дата вступления должна содержаться в своём фрагменте. Для нерелевантных законов проверяется только флаг.
func (d LawDraft) Validate() error {
	if !d.Relevant {
		return nil
	}
	if n := utf8.RuneCountInString(d.Title); n < 20 || n > 140 {
		return fmt.Errorf("заголовок: %d знаков, нужно 20–140", n)
	}
	if strings.ContainsAny(d.Title, "!") || strings.HasSuffix(d.Title, ".") {
		return errors.New("заголовок: без восклицаний и точки в конце")
	}
	for name, v := range map[string]string{"что изменилось": d.WhatChanged, "кого касается": d.WhoAffected} {
		if n := utf8.RuneCountInString(v); n < 15 || n > 450 {
			return fmt.Errorf("%s: %d знаков, нужно 15–450", name, n)
		}
	}
	if len(d.Actions) > 4 {
		return errors.New("слишком много действий")
	}
	for _, a := range d.Actions {
		if utf8.RuneCountInString(a) < 5 || utf8.RuneCountInString(a) > 120 {
			return fmt.Errorf("действие %q: длина вне 5–120", a)
		}
	}
	if len(d.AudienceTags) == 0 {
		return errors.New("нет тегов аудитории")
	}
	for _, t := range d.AudienceTags {
		if !slices.Contains(AudienceTags, t) {
			return fmt.Errorf("тег %q не из списка", t)
		}
	}
	if d.EffectiveAt != "" {
		if !isoDate.MatchString(d.EffectiveAt) {
			return fmt.Errorf("дата вступления %q не в формате YYYY-MM-DD", d.EffectiveAt)
		}
		if _, err := time.Parse("2006-01-02", d.EffectiveAt); err != nil {
			return fmt.Errorf("дата вступления: %w", err)
		}
		if d.Quotes.EffectiveDay == "" {
			return errors.New("дата вступления без цитаты")
		}
	}
	if d.Quotes.WhatChanged == "" || d.Quotes.WhoAffected == "" {
		return errors.New("нет фрагментов-обоснований для «что изменилось» и «кого касается»")
	}
	if d.EffectiveAt != "" && !DateInFragment(d.EffectiveAt, d.Quotes.EffectiveDay) {
		return fmt.Errorf("дата %s не подтверждается фрагментом закона", d.EffectiveAt)
	}
	return nil
}

var monthWords = []string{"января", "февраля", "марта", "апреля", "мая", "июня", "июля", "августа", "сентября", "октября", "ноября", "декабря"}

// DateInFragment: в фрагменте есть день, месяц (словом) и год даты YYYY-MM-DD, либо число в виде dd.mm.yyyy.
func DateInFragment(iso, fragment string) bool {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return false
	}
	f := strings.ToLower(strings.Join(strings.Fields(strings.ReplaceAll(fragment, "\u00a0", " ")), " "))
	if strings.Contains(f, t.Format("02.01.2006")) {
		return true
	}
	dayWord := regexp.MustCompile(`(^|\D)0?` + fmt.Sprint(t.Day()) + `\s+` + monthWords[t.Month()-1] + `\s+` + fmt.Sprint(t.Year()))
	return dayWord.MatchString(f)
}

// Fragment — нумерованный фрагмент текста закона; модель ссылается на номера, а не копирует текст.
type Fragment struct {
	N    int
	Text string
}

var sentenceEnd = regexp.MustCompile(`([.;:]) (?:[А-ЯЁA-Z0-9]|«)`)

// SplitFragments режет текст на фрагменты до maxRunes знаков по границам предложений.
func SplitFragments(text string, maxRunes int) []Fragment {
	var pieces []string
	rest := text
	for len(rest) > 0 {
		loc := sentenceEnd.FindStringSubmatchIndex(rest)
		if loc == nil {
			pieces = append(pieces, rest)
			break
		}
		cut := loc[3] // сразу после знака препинания
		pieces = append(pieces, rest[:cut])
		rest = strings.TrimLeft(rest[cut:], " ")
	}
	var out []Fragment
	var cur strings.Builder
	flush := func() {
		if t := strings.TrimSpace(cur.String()); t != "" {
			out = append(out, Fragment{N: len(out) + 1, Text: t})
		}
		cur.Reset()
	}
	for _, p := range pieces {
		if cur.Len() > 0 && utf8.RuneCountInString(cur.String())+utf8.RuneCountInString(p) > maxRunes {
			flush()
		}
		for utf8.RuneCountInString(p) > maxRunes { // очень длинное «предложение» режем жёстко
			r := []rune(p)
			cur.WriteString(string(r[:maxRunes]))
			flush()
			p = string(r[maxRunes:])
		}
		if cur.Len() > 0 {
			cur.WriteByte(' ')
		}
		cur.WriteString(p)
	}
	flush()
	return out
}

// FragmentsForModel — начало закона и его конец (там обычно вступление в силу) в пределах head+tail знаков.
func FragmentsForModel(fr []Fragment, head, tail int) []Fragment {
	total := 0
	for _, f := range fr {
		total += utf8.RuneCountInString(f.Text)
	}
	if total <= head+tail {
		return fr
	}
	var out []Fragment
	n := 0
	i := 0
	for ; i < len(fr) && n < head; i++ {
		out = append(out, fr[i])
		n += utf8.RuneCountInString(fr[i].Text)
	}
	m := 0
	k := len(fr)
	for k > i && m < tail {
		k--
		m += utf8.RuneCountInString(fr[k].Text)
	}
	return append(out, fr[k:]...)
}
