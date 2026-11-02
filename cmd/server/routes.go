package main

import (
	"context"
	"log"
	"net/http"

	"github.com/guilfoos/crew-predictions/internal/handlers"
	"github.com/guilfoos/crew-predictions/internal/models"
)

// registerRoutes mounts every production route on mux, given the constructed
// Deps. Test-mode-only routes (seed/reset) are added separately via
// registerTestRoutes.
func registerRoutes(mux *http.ServeMux, deps Deps) {
	cfg, stores := deps.Cfg, deps.Stores
	fetchers := fetcherBundle{
		summary: deps.SummaryFetcher,
		refresh: deps.RefreshFetcher,
		match:   func() ([]models.Match, error) { return stores.Match.GetAll() },
	}

	rateLimitMw := func(h http.Handler) http.Handler { return h }
	if !cfg.TestMode {
		rl := handlers.NewRateLimiter(60, 60)
		rateLimitMw = rl.Middleware
	}

	// API endpoints (JSON)
	mh := handlers.NewMatchesHandler(stores.Prediction, stores.Match)
	mux.HandleFunc("GET /api/matches", mh.APIList)

	meh := handlers.NewMeHandler(stores.User)
	mux.HandleFunc("GET /api/me", meh.Get)

	ph := handlers.NewPredictionsHandler(stores.Prediction, fetchers.match)
	mux.HandleFunc("POST /api/predictions", ph.Submit)

	lh := handlers.NewLeaderboardHandler(stores.Prediction, stores.Result, stores.User, stores.Season, cfg.TargetTeam)
	mux.Handle("GET /api/leaderboard", rateLimitMw(http.HandlerFunc(lh.APIList)))
	mux.Handle("GET /api/leaderboard/{season}", rateLimitMw(http.HandlerFunc(lh.APIGetSeason)))

	prh := handlers.NewProfileHandler(stores.Prediction, stores.Result, stores.User, cfg.TargetTeam)
	mux.Handle("GET /api/profile/{userID}", rateLimitMw(http.HandlerFunc(prh.Get)))

	ssh := handlers.NewSeasonsHandler(stores.ConfigStore)
	mux.Handle("GET /api/seasons", rateLimitMw(http.HandlerFunc(ssh.APIList)))

	mdh := handlers.NewMatchDetailHandler(stores.Prediction, stores.Result, stores.Match, stores.User, cfg.TargetTeam, fetchers.summary)
	mux.HandleFunc("GET /api/matches/{matchId}", mdh.Get)

	// Admin endpoints (require AdminAuth)
	rh := handlers.NewResultsHandler(stores.Result, deps.RecalcFn)
	mux.HandleFunc("POST /admin/results", handlers.AdminAuth(rh.Submit))

	csh := handlers.NewCloseSeasonHandler(stores.User, stores.Season, stores.ConfigStore)
	mux.HandleFunc("POST /admin/seasons/close", handlers.AdminAuth(csh.Close))

	onRefresh := func(matches []models.Match) {
		if deps.MatchPoller != nil {
			deps.MatchPoller.Reset(matches)
		}
		deps.TwoOneBot.Predict(context.Background(), matches)
		deps.RecalcFn(context.Background())
	}
	rmh := handlers.NewRefreshMatchesHandler(stores.Match, fetchers.refresh, onRefresh)
	if deps.Enqueuer != nil {
		rmh = rmh.WithEnqueuer(deps.Enqueuer)
	}
	mux.HandleFunc("POST /admin/refresh-matches", handlers.AdminAuth(rmh.Refresh))

	psh := handlers.NewPollScoresHandler(stores.Match, stores.Result, fetchers.refresh, deps.RecalcFn)
	if deps.Enqueuer != nil {
		psh = psh.WithEnqueuer(deps.Enqueuer)
	}
	mux.HandleFunc("POST /admin/poll-scores", handlers.AdminAuth(psh.Poll))

	// Auth + session endpoints
	sh := handlers.NewSessionHandler(deps.Verifier, stores.User)
	hh := handlers.NewHandleHandler(stores.User)
	mux.HandleFunc("POST /auth/session", sh.Create)
	mux.HandleFunc("POST /auth/handle", hh.Update)
	mux.HandleFunc("GET /auth/logout", handlers.Logout)
	mux.HandleFunc("GET /auth/config.js", serveFirebaseConfig)

	// Production gates: missing critical secrets are fatal startup errors.
	if !cfg.TestMode && cfg.AdminKey == "" {
		log.Fatal("ADMIN_KEY env var must be set in production")
	}
	if !cfg.TestMode && cfg.SessionSecret == "" {
		log.Fatal("SESSION_SECRET env var must be set in production")
	}
}

type fetcherBundle struct {
	summary func(string) (models.MatchSummary, error)
	refresh func() ([]models.Match, error)
	match   func() ([]models.Match, error)
}
