package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/guilfoos/crew-predictions/internal/repository"
)

func TestProfileHandler_UsesPrecomputedAcesRadioPoints(t *testing.T) {
	users := repository.NewMemoryUserStore()
	ctx := context.Background()
	users.Upsert(ctx, repository.User{UserID: "u1", Handle: "CrewFan"})
	users.UpdateScores(ctx, "u1", 3, 42, 0, 0)

	h := NewProfileHandler(repository.NewMemoryPredictionStore(), repository.NewMemoryResultStore(), users, "Columbus Crew")
	req := httptest.NewRequest(http.MethodGet, "/api/profile/u1", http.NoBody)
	req.SetPathValue("userID", "u1")
	w := httptest.NewRecorder()
	h.Get(w, req)

	var body struct {
		AcesRadio struct {
			Points int `json:"points"`
		} `json:"acesRadio"`
	}
	json.NewDecoder(w.Body).Decode(&body)
	if body.AcesRadio.Points != 42 {
		t.Errorf("expected 42 from precomputed user doc, got %d", body.AcesRadio.Points)
	}
}

func TestProfileHandler_ReturnsHandleAndLocation(t *testing.T) {
	users := repository.NewMemoryUserStore()
	ctx := context.Background()
	users.Upsert(ctx, repository.User{UserID: "u1", Handle: "CrewFan", Location: "Columbus, OH"})

	h := NewProfileHandler(repository.NewMemoryPredictionStore(), repository.NewMemoryResultStore(), users, "Columbus Crew")
	req := httptest.NewRequest(http.MethodGet, "/api/profile/u1", http.NoBody)
	req.SetPathValue("userID", "u1")
	w := httptest.NewRecorder()
	h.Get(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body struct {
		Handle   string `json:"handle"`
		Location string `json:"location"`
	}
	json.NewDecoder(w.Body).Decode(&body)
	if body.Handle != "CrewFan" {
		t.Errorf("expected handle CrewFan, got %q", body.Handle)
	}
	if body.Location != "Columbus, OH" {
		t.Errorf("expected location Columbus, OH, got %q", body.Location)
	}
}

func TestProfileHandler_ReturnsPredictionCount(t *testing.T) {
	users := repository.NewMemoryUserStore()
	ctx := context.Background()
	users.Upsert(ctx, repository.User{UserID: "u1", Handle: "CrewFan"})
	users.UpdateScores(ctx, "u1", 2, 0, 0, 0)

	h := NewProfileHandler(repository.NewMemoryPredictionStore(), repository.NewMemoryResultStore(), users, "Columbus Crew")
	req := httptest.NewRequest(http.MethodGet, "/api/profile/u1", http.NoBody)
	req.SetPathValue("userID", "u1")
	w := httptest.NewRecorder()
	h.Get(w, req)

	var body struct {
		PredictionCount int `json:"predictionCount"`
	}
	json.NewDecoder(w.Body).Decode(&body)
	if body.PredictionCount != 2 {
		t.Errorf("expected predictionCount 2, got %d", body.PredictionCount)
	}
}

func TestProfileHandler_ReturnsLeaderboardStanding(t *testing.T) {
	users := repository.NewMemoryUserStore()
	ctx := context.Background()
	users.Upsert(ctx, repository.User{UserID: "u1", Handle: "TopFan"})
	users.UpdateScores(ctx, "u1", 1, 15, 0, 0)
	users.Upsert(ctx, repository.User{UserID: "u2", Handle: "OtherFan"})
	users.UpdateScores(ctx, "u2", 1, 0, 0, 0)

	h := NewProfileHandler(repository.NewMemoryPredictionStore(), repository.NewMemoryResultStore(), users, "Columbus Crew")
	req := httptest.NewRequest(http.MethodGet, "/api/profile/u1", http.NoBody)
	req.SetPathValue("userID", "u1")
	w := httptest.NewRecorder()
	h.Get(w, req)

	var body struct {
		AcesRadio struct {
			Points int `json:"points"`
			Rank   int `json:"rank"`
		} `json:"acesRadio"`
	}
	json.NewDecoder(w.Body).Decode(&body)
	if body.AcesRadio.Points != 15 {
		t.Errorf("expected 15 Aces Radio points, got %d", body.AcesRadio.Points)
	}
	if body.AcesRadio.Rank != 1 {
		t.Errorf("expected rank 1, got %d", body.AcesRadio.Rank)
	}
}

func TestProfileHandler_ReturnsGrouchyStanding(t *testing.T) {
	users := repository.NewMemoryUserStore()
	ctx := context.Background()
	users.Upsert(ctx, repository.User{UserID: "u1", Handle: "GrouchyFan"})
	users.UpdateScores(ctx, "u1", 1, 0, 0, 1)
	users.Upsert(ctx, repository.User{UserID: "u2", Handle: "OtherFan"})
	users.UpdateScores(ctx, "u2", 1, 0, 0, 0)

	h := NewProfileHandler(repository.NewMemoryPredictionStore(), repository.NewMemoryResultStore(), users, "Columbus Crew")
	req := httptest.NewRequest(http.MethodGet, "/api/profile/u1", http.NoBody)
	req.SetPathValue("userID", "u1")
	w := httptest.NewRecorder()
	h.Get(w, req)

	var body struct {
		Grouchy struct {
			Points int `json:"points"`
			Rank   int `json:"rank"`
		} `json:"grouchy"`
	}
	json.NewDecoder(w.Body).Decode(&body)
	if body.Grouchy.Points != 1 {
		t.Errorf("expected 1 Grouchy point, got %d", body.Grouchy.Points)
	}
	if body.Grouchy.Rank != 1 {
		t.Errorf("expected rank 1, got %d", body.Grouchy.Rank)
	}
}

func TestProfileHandler_RanksLowerStandingsCorrectly(t *testing.T) {
	users := repository.NewMemoryUserStore()
	ctx := context.Background()
	// u1 is leading with 15, u2 has 10 (rank 2), u3 and u4 are tied at 5 (both rank 3).
	users.Upsert(ctx, repository.User{UserID: "u1", Handle: "Leader"})
	users.UpdateScores(ctx, "u1", 1, 15, 0, 0)
	users.Upsert(ctx, repository.User{UserID: "u2", Handle: "Second"})
	users.UpdateScores(ctx, "u2", 1, 10, 0, 0)
	users.Upsert(ctx, repository.User{UserID: "u3", Handle: "TiedThirdA"})
	users.UpdateScores(ctx, "u3", 1, 5, 0, 0)
	users.Upsert(ctx, repository.User{UserID: "u4", Handle: "TiedThirdB"})
	users.UpdateScores(ctx, "u4", 1, 5, 0, 0)

	h := NewProfileHandler(repository.NewMemoryPredictionStore(), repository.NewMemoryResultStore(), users, "Columbus Crew")

	rankOf := func(userID string) int {
		req := httptest.NewRequest(http.MethodGet, "/api/profile/"+userID, http.NoBody)
		req.SetPathValue("userID", userID)
		w := httptest.NewRecorder()
		h.Get(w, req)
		var body struct {
			AcesRadio struct {
				Rank int `json:"rank"`
			} `json:"acesRadio"`
		}
		if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
			t.Fatalf("decode profile %q: %v", userID, err)
		}
		return body.AcesRadio.Rank
	}

	if got := rankOf("u1"); got != 1 {
		t.Errorf("u1 (15 pts, leader): expected rank 1, got %d", got)
	}
	if got := rankOf("u2"); got != 2 {
		t.Errorf("u2 (10 pts, second): expected rank 2, got %d", got)
	}
	// Tied at 5 — both should share rank 3 (skip-rank semantics).
	if got := rankOf("u3"); got != 3 {
		t.Errorf("u3 (5 pts, tied for third): expected rank 3, got %d", got)
	}
	if got := rankOf("u4"); got != 3 {
		t.Errorf("u4 (5 pts, tied for third): expected rank 3, got %d", got)
	}
}

func TestProfileHandler_Returns500WhenGetAllFails(t *testing.T) {
	h := NewProfileHandler(repository.NewMemoryPredictionStore(), repository.NewMemoryResultStore(), repository.NewErrorGetAllWithExistingUserStore(), "Columbus Crew")
	req := httptest.NewRequest(http.MethodGet, "/api/profile/u1", http.NoBody)
	req.SetPathValue("userID", "u1")
	w := httptest.NewRecorder()
	h.Get(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestProfileHandler_Returns500WhenUserStoreFails(t *testing.T) {
	h := NewProfileHandler(repository.NewMemoryPredictionStore(), repository.NewMemoryResultStore(), repository.NewErrorGetByIDUserStore(), "Columbus Crew")
	req := httptest.NewRequest(http.MethodGet, "/api/profile/u1", http.NoBody)
	req.SetPathValue("userID", "u1")
	w := httptest.NewRecorder()
	h.Get(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestProfileHandler_Returns404ForUnknownUser(t *testing.T) {
	h := NewProfileHandler(repository.NewMemoryPredictionStore(), repository.NewMemoryResultStore(), repository.NewMemoryUserStore(), "Columbus Crew")
	req := httptest.NewRequest(http.MethodGet, "/api/profile/nobody", http.NoBody)
	req.SetPathValue("userID", "nobody")
	w := httptest.NewRecorder()
	h.Get(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestProfileHandler_ReturnsZeroRankWhenNoScoredPredictions(t *testing.T) {
	users := repository.NewMemoryUserStore()
	ctx := context.Background()
	users.Upsert(ctx, repository.User{UserID: "u1", Handle: "NewFan", PredictionCount: 0})

	h := NewProfileHandler(repository.NewMemoryPredictionStore(), repository.NewMemoryResultStore(), users, "Columbus Crew")
	req := httptest.NewRequest(http.MethodGet, "/api/profile/u1", http.NoBody)
	req.SetPathValue("userID", "u1")
	w := httptest.NewRecorder()
	h.Get(w, req)

	var body struct {
		AcesRadio struct {
			Points int `json:"points"`
			Rank   int `json:"rank"`
		} `json:"acesRadio"`
	}
	json.NewDecoder(w.Body).Decode(&body)
	if body.AcesRadio.Rank != 0 {
		t.Errorf("expected rank 0 when no scored predictions, got %d", body.AcesRadio.Rank)
	}
}
