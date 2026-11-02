package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/guilfoos/crew-predictions/internal/repository"
)

type ResultsHandler struct {
	store    repository.ResultStore
	recalcFn func(context.Context)
}

func NewResultsHandler(store repository.ResultStore, recalcFn func(context.Context)) *ResultsHandler {
	return &ResultsHandler{store: store, recalcFn: recalcFn}
}

func (h *ResultsHandler) Submit(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	matchID := r.FormValue("match_id")
	if matchID == "" {
		http.Error(w, "match_id is required", http.StatusBadRequest)
		return
	}
	home, err := strconv.Atoi(r.FormValue("home_goals"))
	if err != nil {
		http.Error(w, "home_goals must be an integer", http.StatusBadRequest)
		return
	}
	away, err := strconv.Atoi(r.FormValue("away_goals"))
	if err != nil {
		http.Error(w, "away_goals must be an integer", http.StatusBadRequest)
		return
	}
	if home < 0 || home > 99 || away < 0 || away > 99 {
		http.Error(w, "goals must be between 0 and 99", http.StatusBadRequest)
		return
	}
	if err := h.store.SaveResult(r.Context(), repository.Result{
		MatchID:   matchID,
		HomeTeam:  r.FormValue("home_team"),
		AwayTeam:  r.FormValue("away_team"),
		HomeGoals: home,
		AwayGoals: away,
	}); err != nil {
		http.Error(w, "could not save result", http.StatusInternalServerError)
		return
	}
	h.recalcFn(r.Context())
	w.WriteHeader(http.StatusNoContent)
}
