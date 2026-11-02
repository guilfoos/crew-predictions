// Package main is the crew-predictions HTTP server entry point. It loads
// configuration, builds the store/handler dependency graph, registers the
// HTTP routes, and starts the daily refresh + match-poller goroutines in
// non-test mode.
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"

	"github.com/guilfoos/crew-predictions/internal/handlers"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run is split out from main so deferred cleanups (poller cancel, daily-refresh
// stop) actually execute on shutdown — log.Fatal/os.Exit skip defers, but
// returning from run() runs them before the top-level log.Fatal.
func run() error {
	_ = godotenv.Load()
	setupLogging()

	cfg := loadConfig()
	ctx := context.Background()

	stores, err := buildStores(ctx, cfg)
	if err != nil {
		return fmt.Errorf("buildStores: %w", err)
	}

	verifier := buildVerifier(ctx, cfg)
	deps := buildDeps(cfg, stores, verifier)
	defer deps.Close()

	if cfg.SessionSecret != "" {
		handlers.SetSessionSecret([]byte(cfg.SessionSecret))
	}

	mux := http.NewServeMux()
	registerRoutes(mux, deps)
	if cfg.TestMode {
		registerTestRoutes(mux, deps)
	}

	if !cfg.TestMode {
		etLoc, err := time.LoadLocation("America/New_York")
		if err != nil {
			return fmt.Errorf("load ET timezone: %w", err)
		}
		stop := startDailyRefresh(stores.Match, deps.RefreshFetcher, deps.MatchPoller, deps.TwoOneBot, deps.RecalcFn, etLoc)
		defer close(stop)

		pollerCtx, cancelPoller := context.WithCancel(ctx)
		defer cancelPoller()
		go deps.MatchPoller.Run(pollerCtx, 2*time.Minute)
	}

	slog.Info("server listening", "port", cfg.Port)
	return http.ListenAndServe(":"+cfg.Port, mux)
}

// setupLogging configures slog to emit JSON that Cloud Logging understands:
// the "level" field becomes "severity" (with WARN remapped to Google's
// "WARNING" spelling), and "msg" becomes "message".
func setupLogging() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				a.Key = "severity"
				if a.Value.String() == "WARN" {
					a.Value = slog.StringValue("WARNING")
				}
			}
			if a.Key == slog.MessageKey {
				a.Key = "message"
			}
			return a
		},
	})))
}
