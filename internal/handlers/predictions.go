package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/guilfoos/crew-predictions/internal/models"
	"github.com/guilfoos/crew-predictions/internal/repository"
)

type PredictionsHandler struct {
	store   repository.PredictionStore
	fetcher func() ([]models.Match, error)
}

func NewPredictionsHandler(store repository.PredictionStore, fetcher func() ([]models.Match, error)) *PredictionsHandler {
	return &PredictionsHandler{store: store, fetcher: fetcher}
}

func (h *PredictionsHandler) Submit(w http.ResponseWriter, r *http.Request) {
	user := UserFromSession(r)
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	_ = r.ParseForm()
	matchID := r.FormValue("match_id")
	if matchID == "" || r.FormValue("home_goals") == "" || r.FormValue("away_goals") == "" {
		http.Error(w, "missing fields", http.StatusBadRequest)
		return
	}
	home, err := strconv.Atoi(r.FormValue("home_goals"))
	if err != nil {
		http.Error(w, "goals must be integers", http.StatusBadRequest)
		return
	}
	away, err := strconv.Atoi(r.FormValue("away_goals"))
	if err != nil {
		http.Error(w, "goals must be integers", http.StatusBadRequest)
		return
	}
	if home < 0 || home > 99 || away < 0 || away > 99 {
		http.Error(w, "goals must be between 0 and 99", http.StatusBadRequest)
		return
	}

	matches, err := h.fetcher()
	if err != nil {
		http.Error(w, "could not verify match", http.StatusInternalServerError)
		return
	}
	var match *models.Match
	for i := range matches {
		if matches[i].ID == matchID {
			match = &matches[i]
			break
		}
	}
	if match == nil {
		http.Error(w, "match not found", http.StatusNotFound)
		return
	}
	if match.Kickoff.Before(time.Now()) || match.Status == "STATUS_DELAYED" {
		slog.Info("prediction rejected: match locked", "matchID", matchID, "status", match.Status, "userID", user.UserID)
		http.Error(w, "predictions are locked for this match", http.StatusForbidden)
		return
	}

	if err := h.store.Save(r.Context(), repository.Prediction{
		MatchID:   matchID,
		UserID:    user.UserID,
		HomeGoals: home,
		AwayGoals: away,
	}); err != nil {
		slog.Error("prediction save failed", "matchID", matchID, "userID", user.UserID, "error", err)
		http.Error(w, "could not save prediction", http.StatusInternalServerError)
		return
	}
	slog.Info("prediction submitted", "matchID", matchID, "userID", user.UserID, "homeGoals", home, "awayGoals", away)
	if r.Header.Get("HX-Request") == "true" {
		_, _ = fmt.Fprintf(w, `<div data-testid="match-card"><div class="saved-score">%d – %d</div><div class="saved-label">Your Pick</div></div>`, home, away)
		return
	}
	http.Redirect(w, r, "/matches", http.StatusFound)
}
