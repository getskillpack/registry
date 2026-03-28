package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/getskillpack/registry/internal/api"
	"github.com/getskillpack/registry/internal/middleware"
	"github.com/getskillpack/registry/internal/store"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))

	dataDir := os.Getenv("REGISTRY_DATA_DIR")
	if dataDir == "" {
		dataDir = "data"
	}
	st, err := store.NewFileStore(dataDir)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	addr := os.Getenv("REGISTRY_LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	writeTok := os.Getenv("REGISTRY_WRITE_TOKEN")
	srv := &api.Server{
		Store:       st,
		WriteToken:  writeTok,
		DefaultAddr: addr,
	}
	if srv.DefaultAddr != "" && srv.DefaultAddr[0] == ':' {
		srv.DefaultAddr = "localhost" + srv.DefaultAddr
	}
	mux := http.NewServeMux()
	srv.Register(mux)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		if fi, err := os.Stat(dataDir); err != nil || !fi.IsDir() {
			http.Error(w, "data dir unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	handler := http.Handler(mux)
	if rps, burst, ok := middleware.ParseRateLimitEnv(); ok {
		lim := middleware.NewIPRateLimiter(rps, burst, 50000)
		handler = middleware.Chain(mux, lim.RateLimit(), middleware.RequestLogger)
		slog.Info("rate limit enabled", "rps", rps, "burst", burst)
	} else {
		handler = middleware.Chain(mux, middleware.RequestLogger)
	}

	slog.Info("registry listening", "addr", addr, "data_dir", dataDir)
	log.Fatal(http.ListenAndServe(addr, handler))
}
