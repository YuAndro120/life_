package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"shtil/backend/internal/api"
)

const (
	defaultFeedWindow = 36 * time.Hour
	dateLayout        = "2006-01-02"
	queryTimeout      = 5 * time.Second
)

type Server struct {
	store  api.Store
	now    func() time.Time
	webDir string // каталог веб-приложения; пусто — только API
}

func NewServer(store api.Store) *Server {
	return &Server{store: store, now: time.Now}
}

// Handler собирает маршруты API v1. Данные отдаются одинаково всем: ни профиля, ни идентификаторов пользователя.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/health", s.health)
	mux.Handle("GET /v1/feed", cached(http.HandlerFunc(s.feed)))
	mux.Handle("GET /v1/laws", cached(http.HandlerFunc(s.laws)))
	if s.webDir != "" {
		mux.Handle("GET /", s.web())
	}
	return mux
}

// NewHTTPServer настраивает http.Server с таймаутами.
func NewHTTPServer(addr string, h http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	status, code := "ok", http.StatusOK
	if err := s.store.Ping(ctx); err != nil {
		slog.Error("health: ping", "err", err)
		status, code = "degraded", http.StatusServiceUnavailable
	}
	writeJSON(w, code, map[string]string{"status": status})
}

func (s *Server) feed(w http.ResponseWriter, r *http.Request) {
	since := s.now().Add(-defaultFeedWindow)
	if raw := r.URL.Query().Get("since"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "since: ожидается время RFC 3339")
			return
		}
		since = t
	}
	ctx, cancel := context.WithTimeout(r.Context(), queryTimeout)
	defer cancel()
	feed, err := s.store.Feed(ctx, since.UTC())
	if err != nil {
		slog.Error("feed", "err", err)
		writeError(w, http.StatusInternalServerError, "внутренняя ошибка")
		return
	}
	writeJSON(w, http.StatusOK, feed)
}

func (s *Server) laws(w http.ResponseWriter, r *http.Request) {
	today := s.now().UTC().Truncate(24 * time.Hour)
	from, to := today, today.AddDate(1, 0, 0)
	for name, dst := range map[string]*time.Time{"from": &from, "to": &to} {
		if raw := r.URL.Query().Get(name); raw != "" {
			t, err := time.Parse(dateLayout, raw)
			if err != nil {
				writeError(w, http.StatusBadRequest, name+": ожидается дата ГГГГ-ММ-ДД")
				return
			}
			*dst = t
		}
	}
	if to.Before(from) {
		writeError(w, http.StatusBadRequest, "to раньше from")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), queryTimeout)
	defer cancel()
	laws, err := s.store.Laws(ctx, from, to)
	if err != nil {
		slog.Error("laws", "err", err)
		writeError(w, http.StatusInternalServerError, "внутренняя ошибка")
		return
	}
	writeJSON(w, http.StatusOK, laws)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("encode response", "err", err)
	}
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
