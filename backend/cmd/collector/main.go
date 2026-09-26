// Команда collector забирает посты источников и синхронизирует каталог.
//
//	collector sync   — загрузить seeds/sources.yaml в БД (активны только проверенные по реестрам)
//	collector once   — один проход по источникам, которым пора
//	collector run    — бесконечный цикл (каждые -interval)
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"shtil/backend/internal/catalog"
	"shtil/backend/internal/collector"
	"shtil/backend/internal/config"
	"shtil/backend/internal/db/sqlcgen"
	"shtil/backend/internal/sources"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "использование: collector sync|once|run [флаги]")
		os.Exit(2)
	}
	cmd := os.Args[1]
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	catalogPath := fs.String("catalog", "seeds/sources.yaml", "каталог источников")
	interval := fs.Duration("interval", 10*time.Minute, "как часто опрашивать источник")
	_ = fs.Parse(os.Args[2:])

	if err := run(cmd, *catalogPath, *interval); err != nil {
		slog.Error("collector", "err", err)
		os.Exit(1)
	}
}

func run(cmd, catalogPath string, interval time.Duration) error {
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

	switch cmd {
	case "sync":
		return syncCatalog(ctx, pool, catalogPath)
	case "once", "run":
		agent := os.Getenv("COLLECTOR_USER_AGENT")
		if agent == "" {
			agent = "ShtilBot/0.1 (personal news digest)"
		}
		r := collector.NewRunner(collector.NewPGStore(pool), sources.NewHTTPFetcher(agent), slog.Default())
		r.Interval = interval
		if cmd == "once" {
			s, err := r.RunOnce(ctx)
			slog.Info("проход завершён", "sources", s.Sources, "failed", s.Failed, "fetched", s.Fetched, "new", s.New, "ads", s.Ads, "suspected", s.Suspected)
			return err
		}
		return loop(ctx, r, interval)
	default:
		return fmt.Errorf("неизвестная команда %q", cmd)
	}
}

func loop(ctx context.Context, r *collector.Runner, interval time.Duration) error {
	// Раз в минуту проверяем, каким источникам пора; сами интервалы и бэкофф решает Runner.Due.
	tick := time.NewTicker(time.Minute)
	defer tick.Stop()
	for {
		s, err := r.RunOnce(ctx)
		if err != nil && ctx.Err() == nil {
			slog.Error("проход не удался", "err", err)
		} else if s.Sources > 0 {
			slog.Info("проход завершён", "sources", s.Sources, "failed", s.Failed, "new", s.New, "ads", s.Ads)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-tick.C:
		}
	}
}

func syncCatalog(ctx context.Context, pool *pgxpool.Pool, path string) error {
	entries, err := catalog.Load(path)
	if err != nil {
		return err
	}
	q := sqlcgen.New(pool)
	var on, off int
	for _, e := range entries {
		active, reason := e.EffectiveActive()
		checked := pgtype.Date{}
		if t, ok := e.LegalCheckedAt(); ok {
			checked = pgtype.Date{Time: t, Valid: true}
		}
		err := q.UpsertSource(ctx, sqlcgen.UpsertSourceParams{
			Kind: e.Kind, Handle: e.Handle, Url: e.URL, Title: e.Title,
			TopicHint:      pgtype.Text{String: e.TopicHint, Valid: e.TopicHint != ""},
			Country:        pgtype.Text{String: e.Country, Valid: e.Country != ""},
			RegionCode:     pgtype.Text{String: e.Region, Valid: e.Region != ""},
			Lang:           e.Lang,
			LegalStatus:    "ok",
			LegalCheckedAt: checked,
			Active:         active,
		})
		if err != nil {
			return fmt.Errorf("источник %s: %w", e.Handle, err)
		}
		if active {
			on++
			slog.Info("источник включён", "handle", e.Handle)
		} else {
			off++
			slog.Warn("источник выключен", "handle", e.Handle, "причина", reason)
		}
	}
	slog.Info("каталог синхронизирован", "включено", on, "выключено", off)
	return nil
}
