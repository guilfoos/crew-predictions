package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"

	"github.com/guilfoos/crew-predictions/internal/models"
	"github.com/guilfoos/crew-predictions/internal/repository"
)

type MatchesHandler struct {
	store      repository.PredictionStore
	matchStore repository.MatchStore
}

func NewMatchesHandler(store repository.PredictionStore, matchStore repository.MatchStore) *MatchesHandler {
	return &MatchesHandler{store: store, matchStore: matchStore}
}

type apiMatch struct {
	ID           string              `json:"id"`
	HomeTeam     string              `json:"homeTeam"`
	AwayTeam     string              `json:"awayTeam"`
	Kickoff      string              `json:"kickoff"`
	Status       string              `json:"status"`
	HomeScore    string              `json:"homeScore"`
	AwayScore    string              `json:"awayScore"`
	State        string              `json:"state"`
	DisplayClock string              `json:"displayClock,omitempty"`
	Venue        string              `json:"venue,omitempty"`
	HomeRecord   string              `json:"homeRecord,omitempty"`
	AwayRecord   string              `json:"awayRecord,omitempty"`
	HomeForm     string              `json:"homeForm,omitempty"`
	AwayForm     string              `json:"awayForm,omitempty"`
	Events       []models.MatchEvent `json:"events,omitempty"`
}

type apiPrediction struct {
	HomeGoals int `json:"homeGoals"`
	AwayGoals int `json:"awayGoals"`
}

func (h *MatchesHandler) APIList(w http.ResponseWriter, r *http.Request) {
	matches, err := h.matchStore.GetAll()
	if err != nil {
		http.Error(w, "couldn't fetch matches", http.StatusInternalServerError)
		return
	}

	sort.Slice(matches, func(i, j int) bool { return matches[i].Kickoff.Before(matches[j].Kickoff) })

	out := make([]apiMatch, len(matches))
	for i, m := range matches {
		out[i] = apiMatch{
			ID:           m.ID,
			HomeTeam:     m.HomeTeam,
			AwayTeam:     m.AwayTeam,
			Kickoff:      m.Kickoff.Format("2006-01-02T15:04:05Z"),
			Status:       m.Status,
			HomeScore:    m.HomeScore,
			AwayScore:    m.AwayScore,
			State:        m.State,
			DisplayClock: m.DisplayClock,
			Venue:        m.Venue,
			HomeRecord:   m.HomeRecord,
			AwayRecord:   m.AwayRecord,
			HomeForm:     m.HomeForm,
			AwayForm:     m.AwayForm,
			Events:       m.Events,
		}
	}

	preds := map[string]apiPrediction{}
	if user := UserFromSession(r); user != nil {
		for _, m := range matches {
			p, _ := h.store.GetByMatchAndUser(r.Context(), m.ID, user.UserID)
			if p != nil {
				preds[m.ID] = apiPrediction{HomeGoals: p.HomeGoals, AwayGoals: p.AwayGoals}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"matches": out, "predictions": preds}); err != nil {
		log.Printf("matches: encode response: %v", err)
	}
}
