// Команда lawtool — юридический конвейер.
//
//	lawtool fetch [-since 2026-07-01] [-limit 30]  — собрать новые законы, извлечь структуру моделью, сохранить черновики
//	lawtool fetch-regional [-since ...] [-limit 200] — региональные законы: название и ссылка с официального портала, без модели
//	lawtool approve-regional [-yes]                 — подтвердить региональные черновики (только названия и ссылки)
//	lawtool retag [-model ...]                       — пересчитать теги аудитории у подтверждённых законов (тексты не меняются)
//	lawtool review                                 — проверить черновики вручную; в API попадают только подтверждённые
//	lawtool stats                                  — сколько подтверждённых, черновиков и отклонённых
//
// Ключ GigaChat берётся из GIGACHAT_AUTH_KEY. Модель только извлекает данные из текста; каждое поле подтверждено
// дословной цитатой, найденной в тексте закона.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"shtil/backend/internal/config"
	"shtil/backend/internal/db/sqlcgen"
	"shtil/backend/internal/legal"
	"shtil/backend/internal/llm"
	"shtil/backend/internal/llm/factory"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "использование: lawtool fetch|review|stats [флаги]")
		os.Exit(2)
	}
	if err := run(os.Args[1], os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, "ошибка:", err)
		os.Exit(1)
	}
}

func run(cmd string, args []string) error {
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	since := fs.String("since", time.Now().AddDate(0, -3, 0).Format("2006-01-02"), "подписаны не раньше этой даты (fetch)")
	yes := fs.Bool("yes", false, "подтвердить без вопросов (approve-regional)")
	limit := fs.Int("limit", 30, "сколько новых законов обработать за запуск (fetch)")
	model := fs.String("model", "", "модель для разбора законов (по умолчанию из окружения провайдера)")
	_ = fs.Parse(args)

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	store := legal.NewPGStore(pool)

	switch cmd {
	case "fetch":
		return fetch(ctx, store, *since, *limit, *model)
	case "retag":
		return retag(ctx, store, *model)
	case "fetch-regional":
		from, err := time.Parse("2006-01-02", *since)
		if err != nil {
			return fmt.Errorf("-since: %w", err)
		}
		rep, err := legal.RunRegional(ctx, legal.NewPravo(), store, from, *limit)
		fmt.Printf("в списке: %d, подходящих по названию: %d, уже видели: %d, новых черновиков: %d\n", rep.Listed, rep.Relevant, rep.Seen, rep.Saved)
		return err
	case "approve-regional":
		if !*yes {
			c, err := store.Queries().CountRegionalDrafts(ctx)
			if err != nil {
				return err
			}
			fmt.Printf("региональных черновиков: %d. Это названия законов регионов и ссылки на официальный портал, без пересказа.\nПодтвердить все: lawtool approve-regional -yes\n", c)
			return nil
		}
		regions, err := store.Queries().ApproveRegionalTitles(ctx)
		fmt.Printf("подтверждено региональных законов: %d\n", len(regions))
		return err
	case "review":
		return review(ctx, store.Queries(), bufio.NewReader(os.Stdin))
	case "stats":
		c, err := store.Queries().CountLaws(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("подтверждено: %d, черновиков: %d, отклонено: %d\n", c.Verified, c.Drafts, c.Rejected)
		return nil
	default:
		return fmt.Errorf("неизвестная команда %q", cmd)
	}
}

func fetch(ctx context.Context, store *legal.PGStore, since string, limit int, modelName string) error {
	from, err := time.Parse("2006-01-02", since)
	if err != nil {
		return fmt.Errorf("-since: %w", err)
	}
	_, extractor, name, err := factory.New(slog.Default(), modelName)
	if err != nil {
		return err
	}
	if extractor == nil {
		return fmt.Errorf("не задан ключ модели для провайдера %s", name)
	}
	slog.Info("модель для законов", "провайдер", name)
	g := &legal.Ingester{
		Pravo: legal.NewPravo(), Kremlin: legal.NewKremlin(), Extractor: extractor,
		Store: store, Log: slog.Default(), Now: time.Now, RateLimitWait: 20 * time.Second,
	}
	rep, err := g.Run(ctx, from, limit)
	fmt.Printf("в списке: %d, уже видели: %d, черновиков: %d, отклонено: %d, ошибок: %d, токенов: %d+%d\n",
		rep.Listed, rep.Seen, rep.Drafts, rep.Rejected, rep.Failed, rep.PromptTokens, rep.CompletionTokens)
	return err
}

func review(ctx context.Context, q *sqlcgen.Queries, in *bufio.Reader) error {
	drafts, err := q.ListLawDrafts(ctx)
	if err != nil {
		return err
	}
	if len(drafts) == 0 {
		fmt.Println("черновиков нет")
		return nil
	}
	ask := func(prompt string) string {
		fmt.Print(prompt)
		s, _ := in.ReadString('\n')
		return strings.TrimSpace(s)
	}
	for i, d := range drafts {
		var actions []string
		_ = json.Unmarshal(d.Actions, &actions)
		var quotes llm.Quotes
		_ = json.Unmarshal(d.Quotes, &quotes)
		p := draftParams{d: d, actions: actions}
		for {
			fmt.Printf("\n══ %d из %d · %s · %s ══\n", i+1, len(drafts), d.ActNumber.String, d.SourceUrl.String)
			p.print(quotes)
			switch ask("[v] подтвердить  [e] править  [r] отклонить  [s] пропустить  [q] выйти > ") {
			case "v":
				if err := p.save(ctx, q); err != nil {
					return err
				}
				if err := q.VerifyLaw(ctx, d.ID); err != nil {
					return err
				}
				fmt.Println("подтверждено")
			case "e":
				p.edit(ask)
				continue
			case "r":
				reason := ask("причина: ")
				if err := q.RejectLaw(ctx, sqlcgen.RejectLawParams{ID: d.ID, RejectReason: pgtype.Text{String: reason, Valid: reason != ""}}); err != nil {
					return err
				}
				fmt.Println("отклонено")
			case "q":
				return nil
			}
			break
		}
	}
	return nil
}

type draftParams struct {
	d       sqlcgen.ListLawDraftsRow
	actions []string
}

func (p *draftParams) print(q llm.Quotes) {
	d := p.d
	eff := "не указана"
	if d.EffectiveAt.Valid {
		eff = d.EffectiveAt.Time.Format("2006-01-02")
	}
	fmt.Printf("Заголовок:     %s\nЧто изменилось: %s\n   цитата: «%s»\nКого касается: %s\n   цитата: «%s»\nДействия:      %s\nТеги:          %s\nВступает:      %s\n   цитата: «%s»\nСтатус:        %s\n",
		d.Title, d.WhatChanged, q.WhatChanged, d.WhoAffected, q.WhoAffected,
		strings.Join(p.actions, "; "), strings.Join(d.AudienceTags, ", "), eff, q.EffectiveDay, d.Status)
}

func (p *draftParams) edit(ask func(string) string) {
	switch f := ask("поле [title|what|who|actions|tags|effective]: "); f {
	case "title":
		p.d.Title = ask("новый заголовок: ")
	case "what":
		p.d.WhatChanged = ask("что изменилось: ")
	case "who":
		p.d.WhoAffected = ask("кого касается: ")
	case "actions":
		p.actions = nil
		for _, a := range strings.Split(ask("действия через «;»: "), ";") {
			if a = strings.TrimSpace(a); a != "" {
				p.actions = append(p.actions, a)
			}
		}
	case "tags":
		var tags []string
		for _, t := range strings.Split(ask("теги через запятую: "), ",") {
			if t = strings.TrimSpace(t); t != "" {
				tags = append(tags, t)
			}
		}
		p.d.AudienceTags = tags
	case "effective":
		v := ask("дата YYYY-MM-DD или пусто: ")
		if v == "" {
			p.d.EffectiveAt = pgtype.Date{}
		} else if t, err := time.Parse("2006-01-02", v); err == nil {
			p.d.EffectiveAt = pgtype.Date{Time: t, Valid: true}
			p.d.Status = legal.StatusFor(&t, time.Now())
		} else {
			fmt.Println("неверная дата")
		}
	}
}

// save записывает правки. Теги проверяются по списку, чтобы правка не внесла «свой» тег.
func (p *draftParams) save(ctx context.Context, q *sqlcgen.Queries) error {
	for _, t := range p.d.AudienceTags {
		if !strings.HasPrefix(t, "region:") && !contains(llm.AudienceTags, t) {
			return fmt.Errorf("тег %q не из списка", t)
		}
	}
	actions, _ := json.Marshal(nonNil(p.actions))
	return q.UpdateLawDraft(ctx, sqlcgen.UpdateLawDraftParams{
		ID: p.d.ID, Title: p.d.Title, WhatChanged: p.d.WhatChanged, WhoAffected: p.d.WhoAffected, Actions: actions,
		AudienceTags: p.d.AudienceTags, RegionCode: p.d.RegionCode, Status: p.d.Status, EffectiveAt: p.d.EffectiveAt,
	})
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// retag заново разбирает тексты подтверждённых законов и обновляет только теги аудитории (например, после расширения набора тегов).
// Название, «что изменилось» и «кого касается», подтверждённые человеком, не трогаются; разница печатается.
func retag(ctx context.Context, store *legal.PGStore, modelName string) error {
	_, extractor, name, err := factory.New(slog.Default(), modelName)
	if err != nil {
		return err
	}
	if extractor == nil {
		return fmt.Errorf("не задан ключ модели для провайдера %s", name)
	}
	slog.Info("модель для законов", "провайдер", name)
	q := store.Queries()
	rows, err := q.ListVerifiedFederalForRetag(ctx)
	if err != nil {
		return err
	}
	kremlin := legal.NewKremlin()
	changed, failed := 0, 0
	var prompt, completion int
	for _, r := range rows {
		doc := legal.Doc{
			EONumber: r.EoNumber.String, Number: strings.TrimPrefix(r.ActNumber.String, "№ "), Signed: r.SignedAt.Time, Title: r.Title,
		}
		_, text, err := kremlin.Text(ctx, doc)
		if err != nil {
			fmt.Printf("%s: текст не найден: %v\n", r.ActNumber.String, err)
			failed++
			continue
		}
		var ex llm.LawDraft
		var usage llm.Usage
		for try := 0; try < 5; try++ {
			ex, usage, err = extractor.ExtractLaw(ctx, llm.LawInput{Title: doc.Title, Number: doc.Number, Fragments: llm.SplitFragments(text, 300)})
			prompt += usage.PromptTokens
			completion += usage.CompletionTokens
			if !errors.Is(err, llm.ErrRateLimited) {
				break
			}
			time.Sleep(20 * time.Second)
		}
		if err != nil || !ex.Relevant {
			fmt.Printf("%s: разбор не удался или закон признан не касающимся граждан, теги не меняю (%v)\n", r.ActNumber.String, err)
			failed++
			continue
		}
		if sameTags(r.AudienceTags, ex.AudienceTags) {
			fmt.Printf("%s: теги те же (%s)\n", r.ActNumber.String, strings.Join(r.AudienceTags, ","))
			continue
		}
		if err := q.UpdateLawTags(ctx, sqlcgen.UpdateLawTagsParams{ID: r.ID, AudienceTags: ex.AudienceTags}); err != nil {
			return err
		}
		changed++
		fmt.Printf("%s: %s → %s\n", r.ActNumber.String, strings.Join(r.AudienceTags, ","), strings.Join(ex.AudienceTags, ","))
	}
	fmt.Printf("законов: %d, теги изменены: %d, не удалось: %d, токенов: %d+%d\n", len(rows), changed, failed, prompt, completion)
	return nil
}

func sameTags(a, b []string) bool {
	x, y := append([]string(nil), a...), append([]string(nil), b...)
	sort.Strings(x)
	sort.Strings(y)
	return strings.Join(x, ",") == strings.Join(y, ",")
}
