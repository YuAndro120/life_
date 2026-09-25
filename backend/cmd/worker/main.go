// Команда worker обрабатывает собранные посты.
//
//	worker once            — один проход: склейка, пересказ моделью, публикация
//	worker run             — бесконечный цикл (каждые -interval)
//	worker debug-clusters  — показать, как склеились бы посты за последние -hours (только чтение, для подбора порога)
//
// Ключ GigaChat берётся только из переменной окружения GIGACHAT_AUTH_KEY.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"shtil/backend/internal/cluster"
	"shtil/backend/internal/config"
	"shtil/backend/internal/llm"
	"shtil/backend/internal/llm/gigachat"
	"shtil/backend/internal/pipeline"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "использование: worker once|run|debug-clusters [флаги]")
		os.Exit(2)
	}
	cmd := os.Args[1]
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	interval := fs.Duration("interval", 5*time.Minute, "период цикла (run)")
	threshold := fs.Float64("threshold", cluster.DefaultConfig().Threshold, "порог склейки, косинусная близость 0..1")
	window := fs.Duration("window", cluster.DefaultConfig().Window, "окно склейки")
	hours := fs.Int("hours", 36, "за сколько часов показывать посты (debug-clusters)")
	batch := fs.Int("batch", 20, "сколько сюжетов пересказывать за проход (ограничивает расход токенов)")
	_ = fs.Parse(os.Args[2:])

	if err := run(cmd, *interval, *threshold, *window, *hours, *batch); err != nil {
		slog.Error("worker", "err", err)
		os.Exit(1)
	}
}

func run(cmd string, interval time.Duration, threshold float64, window time.Duration, hours, batch int) error {
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
	store := pipeline.NewPGStore(pool)

	clusterCfg := cluster.DefaultConfig()
	clusterCfg.Threshold, clusterCfg.Window = threshold, window

	if cmd == "debug-clusters" {
		return debugClusters(ctx, store, clusterCfg, hours)
	}

	var client llm.Client
	if key := os.Getenv("GIGACHAT_AUTH_KEY"); key != "" {
		c, err := gigachat.New(gigachat.Config{
			AuthKey: key, Scope: os.Getenv("GIGACHAT_SCOPE"), Model: os.Getenv("GIGACHAT_MODEL"), Log: slog.Default(),
		})
		if err != nil {
			return err
		}
		client = c
	} else {
		slog.Warn("GIGACHAT_AUTH_KEY не задан: посты только склеиваются, пересказ и публикация отключены")
	}

	w := pipeline.New(store, client, slog.Default())
	w.Cluster = clusterCfg
	w.Batch = batch

	report := func(s pipeline.Summary) {
		slog.Info("проход завершён",
			"новых_постов", s.NewPosts, "новых_сюжетов", s.NewStories, "присоединено", s.Attached,
			"пересказано", s.Digested, "ошибок", s.Failed, "опубликовано", s.Published, "снято", s.Unpublished,
			"токенов_вход", s.PromptTokens, "токенов_выход", s.CompletionTokens, "остановка", s.Stopped)
	}
	switch cmd {
	case "once":
		s, err := w.RunOnce(ctx)
		report(s)
		return err
	case "run":
		tick := time.NewTicker(interval)
		defer tick.Stop()
		for {
			s, err := w.RunOnce(ctx)
			if err != nil && ctx.Err() == nil {
				slog.Error("проход не удался", "err", err)
			} else {
				report(s)
			}
			select {
			case <-ctx.Done():
				return nil
			case <-tick.C:
			}
		}
	default:
		return fmt.Errorf("неизвестная команда %q", cmd)
	}
}

func debugClusters(ctx context.Context, store *pipeline.PGStore, cfg cluster.Config, hours int) error {
	since := time.Now().Add(-time.Duration(hours) * time.Hour)
	posts, err := store.RecentPosts(ctx, since)
	if err != nil {
		return err
	}
	docs := make([]cluster.Doc, len(posts))
	meta := map[int64]pipeline.RecentPost{}
	for i, p := range posts {
		docs[i] = p.Doc
		meta[p.ID] = p
	}
	assign, stories := cluster.Assign(cfg, nil, docs)
	score := map[int64]float64{}
	for _, a := range assign {
		score[a.DocID] = a.Score
	}
	sort.SliceStable(stories, func(i, j int) bool { return len(stories[i].Docs) > len(stories[j].Docs) })

	multi, single := 0, 0
	for _, s := range stories {
		if len(s.Docs) < 2 {
			single++
			continue
		}
		multi++
		srcs := map[int64]bool{}
		for _, d := range s.Docs {
			srcs[d.SourceID] = true
		}
		fmt.Printf("\nСюжет из %d постов, источников %d\n", len(s.Docs), len(srcs))
		for _, d := range s.Docs {
			line := strings.ReplaceAll(strings.SplitN(d.Text, "\n", 2)[0], "\t", " ")
			if r := []rune(line); len(r) > 100 {
				line = string(r[:100]) + "…"
			}
			fmt.Printf("  %.2f  [%s] %s\n", score[d.ID], meta[d.ID].SourceTitle, line)
		}
	}
	fmt.Printf("\nИтого за %d ч при пороге %.2f: постов %d, сюжетов из нескольких постов %d, одиночных %d\n",
		hours, cfg.Threshold, len(posts), multi, single)
	return nil
}
