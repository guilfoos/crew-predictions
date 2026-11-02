package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/guilfoos/crew-predictions/internal/handlers"
)

// TestRegisterRoutes_AllProductionRoutesMounted verifies registerRoutes wires
// every public endpoint. We use mux.Handler() to ask the mux which pattern
// matches a given request, rather than calling ServeHTTP — many handlers
// legitimately return 404 (e.g. "match not found") so a 404 from ServeHTTP
// can't distinguish "route missing" from "handler returned 404". The
// matched pattern, in contrast, is "" only when no route is registered.
func TestRegisterRoutes_AllProductionRoutesMounted(t *testing.T) {
	cfg := Config{TestMode: true, TargetTeam: "Columbus Crew"}
	stores, err := buildStores(t.Context(), cfg)
	if err != nil {
		t.Fatalf("buildStores: %v", err)
	}
	deps := buildDeps(cfg, stores, handlers.NoopTokenVerifier{})

	mux := http.NewServeMux()
	registerRoutes(mux, deps)

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/matches"},
		{http.MethodGet, "/api/me"},
		{http.MethodPost, "/api/predictions"},
		{http.MethodGet, "/api/leaderboard"},
		{http.MethodGet, "/api/leaderboard/2026"},
		{http.MethodGet, "/api/seasons"},
		{http.MethodGet, "/api/profile/u1"},
		{http.MethodGet, "/api/matches/m1"},
		{http.MethodPost, "/admin/results"},
		{http.MethodPost, "/admin/seasons/close"},
		{http.MethodPost, "/admin/refresh-matches"},
		{http.MethodPost, "/admin/poll-scores"},
		{http.MethodPost, "/auth/session"},
		{http.MethodPost, "/auth/handle"},
		{http.MethodGet, "/auth/logout"},
		{http.MethodGet, "/auth/config.js"},
	}

	for _, r := range routes {
		req := httptest.NewRequest(r.method, r.path, http.NoBody)
		_, pattern := mux.Handler(req)
		if pattern == "" {
			t.Errorf("%s %s not registered", r.method, r.path)
		}
	}
}

// TestRegisterRoutes_RateLimiterAttachedInProduction confirms the rate
// limiter is wired in non-test mode. We can only observe this indirectly:
// flooding the leaderboard endpoint past the 60-rps burst should eventually
// produce a 429. The numeric threshold lives in registerRoutes; if it
// changes there, this test is the safety net.
func TestRegisterRoutes_RateLimiterAttachedInProduction(t *testing.T) {
	cfg := Config{
		TestMode:         false,
		FirestoreProject: "",
		TargetTeam:       "Columbus Crew",
		// AdminKey/SessionSecret must be non-empty or registerRoutes calls
		// log.Fatal. The values themselves aren't exercised by this test.
		AdminKey:      "test-admin-key",
		SessionSecret: "test-session-secret",
	}
	stores, err := buildStores(t.Context(), cfg)
	if err != nil {
		t.Fatalf("buildStores: %v", err)
	}
	deps := buildDeps(cfg, stores, handlers.NoopTokenVerifier{})

	mux := http.NewServeMux()
	registerRoutes(mux, deps)

	// 60 burst + 60 rps → blasting 200 requests as fast as we can should
	// trip the limiter at least once. If the rate limiter is silently a
	// no-op, every request returns 200.
	got429 := false
	for range 200 {
		req := httptest.NewRequest(http.MethodGet, "/api/leaderboard", http.NoBody)
		req.RemoteAddr = "10.0.0.1:1234"
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		if rr.Code == http.StatusTooManyRequests {
			got429 = true
			break
		}
	}
	if !got429 {
		t.Error("expected rate limiter to return 429 under load in production mode")
	}
}
