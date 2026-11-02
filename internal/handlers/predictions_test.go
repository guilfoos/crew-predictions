package handlers_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/guilfoos/crew-predictions/internal/handlers"
	"github.com/guilfoos/crew-predictions/internal/models"
	"github.com/guilfoos/crew-predictions/internal/repository"
)

func sessionCookie(userID, handle string) *http.Cookie {
	data, _ := json.Marshal(map[string]string{"userID": userID, "handle": handle})
	return &http.Cookie{Name: "__session", Value: base64.StdEncoding.EncodeToString(data)}
}

func fetcherWithMatch(id string, kickoff time.Time) func() ([]models.Match, error) {
	return func() ([]models.Match, error) {
		return []models.Match{{ID: id, Kickoff: kickoff, Status: "STATUS_SCHEDULED"}}, nil
	}
}

func newHandler() *handlers.PredictionsHandler {
	future := time.Now().Add(24 * time.Hour)
	return handlers.NewPredictionsHandler(repository.NewMemoryPredictionStore(), fetcherWithMatch("match1", future))
}

func errFetcher() ([]models.Match, error) { return nil, fmt.Errorf("store down") }

func TestSubmitPrediction_Returns500WhenFetcherFails(t *testing.T) {
	handler := handlers.NewPredictionsHandler(repository.NewMemoryPredictionStore(), errFetcher)
	req := httptest.NewRequest(http.MethodPost, "/predictions",
		strings.NewReader("match_id=m1&home_goals=2&away_goals=1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(sessionCookie("google:abc123", "BlackAndGold@bsky.mock"))
	w := httptest.NewRecorder()
	handler.Submit(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestSubmitPrediction_RejectsUnauthenticated(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/predictions", http.NoBody)
	w := httptest.NewRecorder()
	newHandler().Submit(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestSubmitPrediction_RejectsMissingFields(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/predictions", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(sessionCookie("google:abc123", "BlackAndGold@bsky.mock"))
	w := httptest.NewRecorder()
	newHandler().Submit(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSubmitPrediction_RejectsNonIntegerGoals(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/predictions",
		strings.NewReader("match_id=abc&home_goals=two&away_goals=one"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(sessionCookie("google:abc123", "BlackAndGold@bsky.mock"))
	w := httptest.NewRecorder()
	newHandler().Submit(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSubmitPrediction_RejectsNonIntegerAwayGoals(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/predictions",
		strings.NewReader("match_id=abc&home_goals=2&away_goals=one"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(sessionCookie("google:abc123", "BlackAndGold@bsky.mock"))
	w := httptest.NewRecorder()
	newHandler().Submit(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSubmitPrediction_ReturnsMatchCardWithSavedScore(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/predictions",
		strings.NewReader("match_id=match1&home_goals=3&away_goals=1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.AddCookie(sessionCookie("google:abc123", "BlackAndGold@bsky.mock"))
	w := httptest.NewRecorder()
	newHandler().Submit(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "3") || !strings.Contains(w.Body.String(), "1") {
		t.Errorf("expected saved score in response, got: %s", w.Body.String())
	}
}

func TestSubmitPrediction_RedirectsOnNonHTMX(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/predictions",
		strings.NewReader("match_id=match1&home_goals=3&away_goals=1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(sessionCookie("google:abc123", "BlackAndGold@bsky.mock"))
	w := httptest.NewRecorder()
	newHandler().Submit(w, req)
	if w.Code != http.StatusFound {
		t.Errorf("expected 302, got %d", w.Code)
	}
	if w.Header().Get("Location") != "/matches" {
		t.Errorf("expected redirect to /matches, got %s", w.Header().Get("Location"))
	}
}

func TestSubmitPrediction_RejectsAfterKickoff(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour)
	handler := handlers.NewPredictionsHandler(
		repository.NewMemoryPredictionStore(),
		fetcherWithMatch("match1", past),
	)
	req := httptest.NewRequest(http.MethodPost, "/predictions",
		strings.NewReader("match_id=match1&home_goals=3&away_goals=1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(sessionCookie("google:abc123", "BlackAndGold@bsky.mock"))
	w := httptest.NewRecorder()
	handler.Submit(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestSubmitPrediction_RejectsDelayedMatch(t *testing.T) {
	future := time.Now().Add(24 * time.Hour) // future kickoff — only STATUS_DELAYED causes rejection
	fetcher := func() ([]models.Match, error) {
		return []models.Match{{ID: "match1", Kickoff: future, Status: "STATUS_DELAYED"}}, nil
	}
	handler := handlers.NewPredictionsHandler(repository.NewMemoryPredictionStore(), fetcher)
	req := httptest.NewRequest(http.MethodPost, "/predictions",
		strings.NewReader("match_id=match1&home_goals=2&away_goals=1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(sessionCookie("google:abc123", "BlackAndGold@bsky.mock"))
	w := httptest.NewRecorder()
	handler.Submit(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for delayed match, got %d", w.Code)
	}
}

func TestSubmitPrediction_RejectsUnknownMatch(t *testing.T) {
	future := time.Now().Add(24 * time.Hour)
	handler := handlers.NewPredictionsHandler(
		repository.NewMemoryPredictionStore(),
		fetcherWithMatch("other-match", future),
	)
	req := httptest.NewRequest(http.MethodPost, "/predictions",
		strings.NewReader("match_id=unknown&home_goals=3&away_goals=1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(sessionCookie("google:abc123", "BlackAndGold@bsky.mock"))
	w := httptest.NewRecorder()
	handler.Submit(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestSubmitPrediction_RejectsNegativeGoals(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/predictions",
		strings.NewReader("match_id=match1&home_goals=-1&away_goals=1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(sessionCookie("google:abc123", "BlackAndGold@bsky.mock"))
	w := httptest.NewRecorder()
	newHandler().Submit(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for negative goals, got %d", w.Code)
	}
}

func TestSubmitPrediction_RejectsTooLargeGoals(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/predictions",
		strings.NewReader("match_id=match1&home_goals=100&away_goals=0"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(sessionCookie("google:abc123", "BlackAndGold@bsky.mock"))
	w := httptest.NewRecorder()
	newHandler().Submit(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for goals > 99, got %d", w.Code)
	}
}

func TestSubmitPrediction_Returns500WhenSaveFails(t *testing.T) {
	future := time.Now().Add(24 * time.Hour)
	handler := handlers.NewPredictionsHandler(repository.NewErrorPredictionStore(), fetcherWithMatch("match1", future))
	req := httptest.NewRequest(http.MethodPost, "/predictions",
		strings.NewReader("match_id=match1&home_goals=2&away_goals=1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(sessionCookie("google:abc123", "BlackAndGold@bsky.mock"))
	w := httptest.NewRecorder()
	handler.Submit(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when Save fails, got %d", w.Code)
	}
}

func TestSubmitPrediction_SavesPredictionToStore(t *testing.T) {
	store := repository.NewMemoryPredictionStore()
	req := httptest.NewRequest(http.MethodPost, "/predictions",
		strings.NewReader("match_id=match1&home_goals=3&away_goals=1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.AddCookie(sessionCookie("google:abc123", "BlackAndGold@bsky.mock"))
	w := httptest.NewRecorder()
	handlers.NewPredictionsHandler(store, fetcherWithMatch("match1", time.Now().Add(24*time.Hour))).Submit(w, req)
	got, err := store.GetByMatchAndUser(req.Context(), "match1", "google:abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected prediction to be saved, got nil")
	}
	if got.HomeGoals != 3 || got.AwayGoals != 1 {
		t.Errorf("expected 3-1, got %d-%d", got.HomeGoals, got.AwayGoals)
	}
}
