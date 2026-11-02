package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/guilfoos/crew-predictions/internal/handlers"
	"github.com/guilfoos/crew-predictions/internal/repository"
)

func TestSeedPredictionHandler_SavesPrediction(t *testing.T) {
	store := repository.NewMemoryPredictionStore()
	sh := handlers.NewSeedPredictionHandler(store)

	req := httptest.NewRequest(http.MethodPost, "/admin/seed-prediction",
		strings.NewReader("match_id=match-scoring-1&user_id=google:user1&handle=user1@bsky.mock&home_goals=2&away_goals=0"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	sh.Submit(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
	got, err := store.GetByMatchAndUser(context.Background(), "match-scoring-1", "google:user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected prediction to be saved, got nil")
	}
	if got.HomeGoals != 2 || got.AwayGoals != 0 {
		t.Errorf("expected 2-0, got %d-%d", got.HomeGoals, got.AwayGoals)
	}
}

func TestSeedPredictionHandler_RejectsBadAwayGoals(t *testing.T) {
	sh := handlers.NewSeedPredictionHandler(repository.NewMemoryPredictionStore())

	req := httptest.NewRequest(http.MethodPost, "/admin/seed-prediction",
		strings.NewReader("match_id=m1&user_id=u1&handle=h&home_goals=2&away_goals=bad"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	sh.Submit(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSeedPredictionHandler_RejectsBadGoals(t *testing.T) {
	sh := handlers.NewSeedPredictionHandler(repository.NewMemoryPredictionStore())

	req := httptest.NewRequest(http.MethodPost, "/admin/seed-prediction",
		strings.NewReader("match_id=m1&user_id=u1&handle=h&home_goals=bad&away_goals=0"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	sh.Submit(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSeedUserHandler_UpsertsSetsHandleAndUserID(t *testing.T) {
	store := repository.NewMemoryUserStore()
	sh := handlers.NewSeedUserHandler(store)

	req := httptest.NewRequest(http.MethodPost, "/admin/seed-user",
		strings.NewReader("user_id=google:fan1&handle=fan1@bsky.mock"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	sh.Submit(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
	u, err := store.GetByID(context.Background(), "google:fan1")
	if err != nil || u == nil {
		t.Fatalf("expected user to exist, got err=%v u=%v", err, u)
	}
	if u.Handle != "fan1@bsky.mock" {
		t.Errorf("expected handle fan1@bsky.mock, got %q", u.Handle)
	}
}

func TestSeedUserHandler_RejectsMissingUserID(t *testing.T) {
	sh := handlers.NewSeedUserHandler(repository.NewMemoryUserStore())

	req := httptest.NewRequest(http.MethodPost, "/admin/seed-user",
		strings.NewReader("handle=fan1@bsky.mock"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	sh.Submit(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
