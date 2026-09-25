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
	Title  string
	Number string
	Text   string
}

// Quotes — дословные фрагменты текста, на которых держится каждое поле. Их проверяет код и человек в lawtool review.
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

// Validate проверяет черновик по тексту закона. Для нерелевантных законов проверяется только флаг.
func (d LawDraft) Validate(text string) error {
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
	norm := normalizeForQuote(text)
	for name, q := range map[string]string{
		"что изменилось": d.Quotes.WhatChanged, "кого касается": d.Quotes.WhoAffected, "вступление в силу": d.Quotes.EffectiveDay,
	} {
		if q == "" {
			if name == "вступление в силу" {
				continue
			}
			return fmt.Errorf("нет цитаты для поля «%s»", name)
		}
		if utf8.RuneCountInString(q) < 12 {
			return fmt.Errorf("цитата для «%s» слишком короткая", name)
		}
		if !strings.Contains(norm, normalizeForQuote(q)) {
			return fmt.Errorf("цитата для «%s» не найдена в тексте", name)
		}
	}
	return nil
}

// normalizeForQuote приводит текст и цитату к одному виду: регистр, пробелы, кавычки и тире.
func normalizeForQuote(s string) string {
	s = strings.NewReplacer(" ", " ", "«", `"`, "»", `"`, "“", `"`, "”", `"`, "—", "-", "–", "-", "…", "...").Replace(s)
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}

// ExcerptForModel — начало закона плюс конец (там обычно статья о вступлении в силу), чтобы не тратить лишние токены.
func ExcerptForModel(text string, head, tail int) string {
	r := []rune(text)
	if len(r) <= head+tail {
		return text
	}
	return string(r[:head]) + " [...] " + string(r[len(r)-tail:])
}
