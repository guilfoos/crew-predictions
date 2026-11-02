package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/guilfoos/crew-predictions/internal/repository"
)

// Stores groups every repository the server depends on. Built once at startup;
// passed by value (or as a Deps field) to whatever needs the data layer.
//
// MemMatch is exposed alongside Match because the test-mode seed handlers
// (POST /admin/seed-match, POST /admin/seed-match-events) need the concrete
// MemoryMatchStore for write access; the public MatchStore interface is
// satisfied by either the memory store directly or a write-through wrapper
// over Firestore.
type Stores struct {
	User        repository.UserStore
	Prediction  repository.PredictionStore
	Result      repository.ResultStore
	Match       repository.MatchStore
	MemMatch    *repository.MemoryMatchStore
	Season      repository.SeasonStore
	ConfigStore repository.ConfigStore
}

// buildStores constructs every store based on Config. When FirestoreProject
// is set, Firestore-backed implementations are used (with write-through
// caching for matches); otherwise everything is in-memory. TestMode is
// independent of the project — TEST_MODE always uses memory regardless of
// project setting, but production with no project also uses memory (local
// dev without Firestore).
func buildStores(ctx context.Context, cfg Config) (Stores, error) {
	useFirestore := cfg.FirestoreProject != "" && !cfg.TestMode
	memMatch := repository.NewMemoryMatchStore()
	stores := Stores{
		Season:      repository.NewMemorySeasonStore(),
		ConfigStore: repository.NewMemoryConfigStore("2026"),
		MemMatch:    memMatch,
		Match:       memMatch,
	}

	if !useFirestore {
		stores.User = repository.NewMemoryUserStore()
		stores.Prediction = repository.NewMemoryPredictionStore()
		stores.Result = repository.NewMemoryResultStore()
		slog.Info("store: in-memory (set GOOGLE_CLOUD_PROJECT to use Firestore)")
		return stores, nil
	}

	user, err := repository.NewFirestoreUserStore(ctx, cfg.FirestoreProject)
	if err != nil {
		return Stores{}, fmt.Errorf("firestore users: %w", err)
	}
	stores.User = user

	pred, err := repository.NewFirestorePredictionStore(ctx, cfg.FirestoreProject)
	if err != nil {
		return Stores{}, fmt.Errorf("firestore predictions: %w", err)
	}
	stores.Prediction = pred

	result, err := repository.NewFirestoreResultStore(ctx, cfg.FirestoreProject)
	if err != nil {
		return Stores{}, fmt.Errorf("firestore results: %w", err)
	}
	stores.Result = result

	fsMatches, err := repository.NewFirestoreMatchStore(cfg.FirestoreProject)
	if err != nil {
		return Stores{}, fmt.Errorf("firestore matches: %w", err)
	}
	stores.Match = repository.NewWriteThroughMatchStore(memMatch, fsMatches)
	// Pre-populate memory from Firestore so match data survives restarts.
	if stored, err := fsMatches.GetAll(); err != nil {
		slog.Warn("could not load matches from Firestore", "error", err)
	} else if len(stored) > 0 {
		if err := memMatch.SaveAll(stored); err != nil {
			slog.Warn("could not warm match cache from Firestore", "error", err)
		} else {
			slog.Info("startup: matches loaded from Firestore", "count", len(stored))
		}
	}
	slog.Info("store: Firestore", "project", cfg.FirestoreProject)

	return stores, nil
}
