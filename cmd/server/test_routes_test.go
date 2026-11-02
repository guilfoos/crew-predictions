package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/guilfoos/crew-predictions/internal/handlers"
	"github.com/guilfoos/crew-predictions/internal/repository"
)

// memSavablePrediction is a one-off test fixture so the reset assertions
// have something concrete to delete.
func memSavablePrediction() repository.Prediction {
	return repository.Prediction{MatchID: "m1", UserID: "u1", HomeGoals: 2, AwayGoals: 1}
}

func newTestModeDeps(t *testing.T) Deps {
	t.Helper()
	cfg := Config{TestMode: true, TargetTeam: "Columbus Crew"}
	stores, err := buildStores(t.Context(), cfg)
	if err != nil {
		t.Fatalf("buildStores: %v", err)
	}
	return buildDeps(cfg, stores, handlers.NoopTokenVerifier{})
}

func TestRegisterTestRoutes_RegistersResetEndpoint(t *testing.T) {
	mux := http.NewServeMux()
	registerTestRoutes(mux, newTestModeDeps(t))

	req := httptest.NewRequest(http.MethodDelete, "/admin/reset", http.NoBody)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204 from /admin/reset, got %d", rr.Code)
	}
}

func TestRegisterTestRoutes_RegistersAllSeedEndpoints(t *testing.T) {
	mux := http.NewServeMux()
	registerTestRoutes(mux, newTestModeDeps(t))

	// Hit each seed route with an empty body. The exact response code varies
	// (mostly 400 for missing fields) — what matters here is that the route
	// is *handled* (not 404), which proves registration happened.
	paths := []string{
		"/admin/seed-prediction",
		"/admin/seed-user",
		"/admin/seed-match",
		"/admin/seed-match-events",
		"/admin/seed-season",
	}
	for _, p := range paths {
		req := httptest.NewRequest(http.MethodPost, p, http.NoBody)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		if rr.Code == http.StatusNotFound {
			t.Errorf("%s not registered (got 404)", p)
		}
	}
}

func TestRegisterTestRoutes_SkipsWhenStoresNotMemoryBacked(t *testing.T) {
	// Swap in a non-memory PredictionStore implementation. The gate should
	// log a warning and skip *all* registrations, so /admin/reset returns 404.
	deps := newTestModeDeps(t)
	deps.Stores.Prediction = repository.NewErrorPredictionStore()

	mux := http.NewServeMux()
	registerTestRoutes(mux, deps)

	req := httptest.NewRequest(http.MethodDelete, "/admin/reset", http.NoBody)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected /admin/reset to be unregistered (404) when stores are not all memory-backed, got %d", rr.Code)
	}
}

func TestRegisterTestRoutes_ResetClearsAllMemoryStores(t *testing.T) {
	deps := newTestModeDeps(t)
	mux := http.NewServeMux()
	registerTestRoutes(mux, deps)

	memPred := deps.Stores.Prediction.(*repository.MemoryPredictionStore)
	memUser := deps.Stores.User.(*repository.MemoryUserStore)

	// Seed some data, then issue the reset, then verify it's gone.
	if err := memPred.Save(t.Context(), memSavablePrediction()); err != nil {
		t.Fatalf("seed prediction: %v", err)
	}
	if err := memUser.Upsert(t.Context(), repository.User{UserID: "u1", Handle: "User"}); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/admin/reset", http.NoBody)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("reset returned %d", rr.Code)
	}

	if all, _ := memPred.GetAll(t.Context()); len(all) != 0 {
		t.Errorf("expected predictions cleared, got %d", len(all))
	}
	if u, _ := memUser.GetByID(t.Context(), "u1"); u != nil {
		t.Errorf("expected user cleared, got %+v", u)
	}
}
