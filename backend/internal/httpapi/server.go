package httpapi

import (
	"encoding/json"
	"net/http"
	"time"
)

// NewHandler собирает маршруты API v1.
func NewHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/health", health)
	return mux
}

// NewServer настраивает http.Server с таймаутами.
func NewServer(addr string, h http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
