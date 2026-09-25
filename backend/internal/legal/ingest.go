package legal

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"shtil/backend/internal/llm"
)

// Draft — законопроект-черновик, готовый к записи в БД (verified = false).
type Draft struct {
	Doc         Doc
	Extract     llm.LawDraft
	PassedAt    *time.Time
	EffectiveAt *time.Time
	Status      string
	SourceURL   string
}

// Store — то, что нужно сборщику от хранилища.
type Store interface {
	Seen(ctx context.Context, eoNumber string) (bool, error)
	SaveDraft(ctx context.Context, d Draft) error
	SaveRejected(ctx context.Context, doc Doc, reason string) error
}

type Ingester struct {
	Pravo     *Pravo
	Kremlin   *Kremlin
	Extractor llm.LawExtractor
	Store     Store
	Log       *slog.Logger
	Now       func() time.Time
	// Sleep подменяется в тестах; RateLimitWait — пауза после 429 от модели (воркер делит с нами лимит запросов).
	Sleep         func(time.Duration)
	RateLimitWait time.Duration
}

type Report struct {
	Listed, Seen, Drafts, Rejected, Failed int
	PromptTokens, CompletionTokens         int
}

// notForCitizens — законы, которые заведомо не касаются граждан: экономим токены и не показываем человеку.
var notForCitizens = regexp.MustCompile(`(?i)ратификац|денонсац|о соглашении между|международн\w+ договор|о награжден|о протоколе`)

// Run собирает до limit ещё не виденных законов, подписанных не раньше from. Каждый обрабатывается один раз.
func (g *Ingester) Run(ctx context.Context, from time.Time, limit int) (Report, error) {
	var rep Report
	docs, err := g.Pravo.Laws(ctx, from, limit*3)
	if err != nil {
		return rep, fmt.Errorf("список законов: %w", err)
	}
	rep.Listed = len(docs)
	processed := 0
	for _, d := range docs {
		if processed >= limit {
			break
		}
		if err := ctx.Err(); err != nil {
			return rep, err
		}
		seen, err := g.Store.Seen(ctx, d.EONumber)
		if err != nil {
			return rep, err
		}
		if seen {
			rep.Seen++
			continue
		}
		if notForCitizens.MatchString(d.Title) {
			if err := g.Store.SaveRejected(ctx, d, "не касается граждан по названию"); err != nil {
				return rep, err
			}
			rep.Rejected++
			continue
		}
		processed++
		if err := g.one(ctx, d, &rep); err != nil {
			if errors.Is(err, llm.ErrRefused) {
				g.Log.Warn("модерация модели отказалась", "номер", d.Number)
				rep.Rejected++
				if err := g.Store.SaveRejected(ctx, d, "модерация GigaChat отказалась обрабатывать текст: разобрать вручную"); err != nil {
					return rep, err
				}
				continue
			}
			if errors.Is(err, llm.ErrInvalid) || errors.Is(err, ErrNoText) || errors.Is(err, llm.ErrRateLimited) {
				g.Log.Warn("закон пропущен", "номер", d.Number, "err", err)
				rep.Failed++
				continue // без записи: повторим при следующем запуске
			}
			return rep, err
		}
	}
	return rep, nil
}

func (g *Ingester) one(ctx context.Context, d Doc, rep *Report) error {
	pageURL, text, err := g.Kremlin.Text(ctx, d)
	if err != nil {
		return err
	}
	var ex llm.LawDraft
	for try := 0; ; try++ {
		var usage llm.Usage
		ex, usage, err = g.Extractor.ExtractLaw(ctx, llm.LawInput{Title: d.Title, Number: d.Number, Text: text})
		rep.PromptTokens += usage.PromptTokens
		rep.CompletionTokens += usage.CompletionTokens
		if !errors.Is(err, llm.ErrRateLimited) || try == 4 {
			break
		}
		g.Log.Warn("модель просит подождать", "пауза", g.RateLimitWait)
		if g.Sleep != nil {
			g.Sleep(g.RateLimitWait)
		} else {
			time.Sleep(g.RateLimitWait)
		}
	}
	if err != nil {
		return err
	}
	if !ex.Relevant {
		g.Log.Info("не касается граждан", "номер", d.Number)
		rep.Rejected++
		return g.Store.SaveRejected(ctx, d, "модель: не касается граждан и ИП")
	}
	draft := Draft{Doc: d, Extract: ex, SourceURL: pageURL, PassedAt: PassedDate(text)}
	if ex.EffectiveAt != "" {
		if t, err := time.Parse("2006-01-02", ex.EffectiveAt); err == nil {
			draft.EffectiveAt = &t
		}
	}
	draft.Status = StatusFor(draft.EffectiveAt, g.Now())
	rep.Drafts++
	g.Log.Info("черновик закона", "номер", d.Number, "вступает", ex.EffectiveAt, "теги", strings.Join(ex.AudienceTags, ","))
	return g.Store.SaveDraft(ctx, draft)
}

// StatusFor: подписанный закон, вступивший в силу, — in_force; иначе — signed.
func StatusFor(effective *time.Time, now time.Time) string {
	if effective != nil && !effective.After(now) {
		return "in_force"
	}
	return "signed"
}

var months = map[string]time.Month{
	"января": 1, "февраля": 2, "марта": 3, "апреля": 4, "мая": 5, "июня": 6,
	"июля": 7, "августа": 8, "сентября": 9, "октября": 10, "ноября": 11, "декабря": 12,
}

var passedRe = regexp.MustCompile(`(?i)Принят\s+Государственной\s+Думой\s+(\d{1,2})\s+([а-я]+)\s+(\d{4})`)

// PassedDate достаёт дату принятия Госдумой из текста («Принят Государственной Думой 23 июля 2026 года»).
func PassedDate(text string) *time.Time {
	m := passedRe.FindStringSubmatch(text)
	if m == nil {
		return nil
	}
	month, ok := months[strings.ToLower(m[2])]
	if !ok {
		return nil
	}
	var day, year int
	fmt.Sscan(m[1], &day)
	fmt.Sscan(m[3], &year)
	t := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	if t.Day() != day { // 31 февраля и т. п.
		return nil
	}
	return &t
}
