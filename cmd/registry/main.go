package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/getskillpack/registry"
	"github.com/getskillpack/registry/internal/api"
	"github.com/getskillpack/registry/internal/middleware"
	"github.com/getskillpack/registry/internal/store"
)

func main() {
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
		Store:               st,
		WriteToken:          writeTok,
		DefaultAddr:         addr,
		RegistryAPIMarkdown: registry.RegistryAPIMarkdown,
		OGCardSVG:           registry.OGCardSVG,
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
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if err := st.Ping(r.Context()); err != nil {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	if strings.EqualFold(strings.TrimSpace(os.Getenv("REGISTRY_LOG_FORMAT")), "json") {
		logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}
	slog.SetDefault(logger)

	rps, _ := strconv.ParseFloat(strings.TrimSpace(os.Getenv("REGISTRY_RATE_LIMIT_RPS")), 64)
	burst := 0
	if v := strings.TrimSpace(os.Getenv("REGISTRY_RATE_LIMIT_BURST")); v != "" {
		burst, _ = strconv.Atoi(v)
	}
	trustFwd := envBool(os.Getenv("REGISTRY_TRUST_FORWARDED_FOR"))

	handler := middleware.Chain(mux,
		middleware.Recover(logger),
		middleware.RequestLog(logger),
		middleware.RateLimit(middleware.RateLimitConfig{
			RPS:              rps,
			Burst:            burst,
			TrustForwarded:   trustFwd,
			SkipPathPrefixes: []string{"/healthz", "/readyz"},
		}),
	)

	logger.Info("registry listening",
		slog.String("addr", addr),
		slog.String("data", dataDir),
		slog.Float64("rate_limit_rps", rps),
		slog.Int("rate_limit_burst", burst),
		slog.Bool("trust_forwarded_for", trustFwd),
	)
	log.Fatal(http.ListenAndServe(addr, handler))
}

func envBool(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
