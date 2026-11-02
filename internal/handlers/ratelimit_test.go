//go:build !integration

package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/guilfoos/crew-predictions/internal/handlers"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestRateLimiter_AllowsRequestsUnderLimit(t *testing.T) {
	rl := handlers.NewRateLimiter(60, 2)
	h := rl.Middleware(okHandler())

	for i := range 2 {
		req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
		req.RemoteAddr = "1.2.3.4:1234"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i, w.Code)
		}
	}
}

func TestRateLimiter_Returns429WhenLimitExceeded(t *testing.T) {
	rl := handlers.NewRateLimiter(60, 1)
	h := rl.Middleware(okHandler())

	req1 := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req1.RemoteAddr = "1.2.3.4:1234"
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", w1.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req2.RemoteAddr = "1.2.3.4:1234"
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req2)
	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("second request: expected 429, got %d", w2.Code)
	}
}

func TestRateLimiter_SeparateBucketsPerIP(t *testing.T) {
	rl := handlers.NewRateLimiter(60, 1)
	h := rl.Middleware(okHandler())

	for _, ip := range []string{"1.2.3.4:1234", "5.6.7.8:5678"} {
		req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
		req.RemoteAddr = ip
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("IP %s: expected 200, got %d", ip, w.Code)
		}
	}
}
