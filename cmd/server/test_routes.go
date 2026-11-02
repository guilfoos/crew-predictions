package main

import (
	"log/slog"
	"net/http"

	"github.com/guilfoos/crew-predictions/internal/handlers"
	"github.com/guilfoos/crew-predictions/internal/repository"
)

// registerTestRoutes mounts the seed/reset endpoints used exclusively by the
// e2e suite. These routes are never exposed in production: the caller (main)
// gates the call on cfg.TestMode, and the seed handlers themselves require
// the in-memory store types so a misconfigured production deploy can't
// accidentally enable them.
//
// The DELETE /admin/reset endpoint requires every store to be a memory
// implementation (it type-asserts each one). If any store has been swapped
// for a Firestore-backed variant the registration is skipped and a warning
// is logged.
func registerTestRoutes(mux *http.ServeMux, deps Deps) {
	stores := deps.Stores

	memPred, predOK := stores.Prediction.(*repository.MemoryPredictionStore)
	memResult, resultOK := stores.Result.(*repository.MemoryResultStore)
	memUser, userOK := stores.User.(*repository.MemoryUserStore)
	memSeason, seasonOK := stores.Season.(*repository.MemorySeasonStore)
	memConfig, configOK := stores.ConfigStore.(*repository.MemoryConfigStore)

	if !(predOK && resultOK && userOK && seasonOK && configOK) {
		slog.Warn("test routes skipped: stores are not all memory-backed")
		return
	}

	mux.HandleFunc("DELETE /admin/reset", func(w http.ResponseWriter, _ *http.Request) {
		memPred.Reset()
		memResult.Reset()
		memUser.Reset()
		stores.Match.Reset()
		memSeason.Reset()
		memConfig.Reset()
		w.WriteHeader(http.StatusNoContent)
	})

	seedPred := handlers.NewSeedPredictionHandler(stores.Prediction)
	mux.HandleFunc("POST /admin/seed-prediction", seedPred.Submit)

	seedUser := handlers.NewSeedUserHandler(stores.User)
	mux.HandleFunc("POST /admin/seed-user", seedUser.Submit)

	seedMatch := handlers.NewSeedMatchHandler(stores.MemMatch)
	mux.HandleFunc("POST /admin/seed-match", seedMatch.Submit)

	seedMatchEvents := handlers.NewSeedMatchEventsHandler(stores.MemMatch)
	mux.HandleFunc("POST /admin/seed-match-events", seedMatchEvents.Submit)

	seedSeason := handlers.NewSeedSeasonHandler(stores.Season)
	mux.HandleFunc("POST /admin/seed-season", seedSeason.Submit)

	slog.Info("test routes registered: /admin/reset, /admin/seed-*")
}
