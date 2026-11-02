package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"

	"github.com/guilfoos/crew-predictions/internal/repository"
)

type LeaderboardHandler struct {
	users   repository.UserStore
	seasons repository.SeasonStore
}

func NewLeaderboardHandler(_ repository.PredictionStore, _ repository.ResultStore, users repository.UserStore, seasons repository.SeasonStore, _ string) *LeaderboardHandler {
	return &LeaderboardHandler{users: users, seasons: seasons}
}

type leaderboardEntry struct {
	UserID          string `json:"userID"`
	Handle          string `json:"handle"`
	AcesRadioPoints int    `json:"acesRadioPoints"`
	Upper90Points   int    `json:"upper90ClubPoints"`
	GrouchyPoints   int    `json:"grouchyPoints"`
	HasProfile      bool   `json:"hasProfile"`
}

func (h *LeaderboardHandler) APIGetSeason(w http.ResponseWriter, r *http.Request) {
	seasonID := r.PathValue("season")
	snap, err := h.seasons.GetByID(r.Context(), seasonID)
	if err != nil {
		http.Error(w, "couldn't load season", http.StatusInternalServerError)
		return
	}
	if snap == nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"entries": snap.Entries}); err != nil {
		log.Printf("leaderboard season: encode response: %v", err)
	}
}

func (h *LeaderboardHandler) APIList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	allUsers, err := h.users.GetAll(ctx)
	if err != nil {
		http.Error(w, "couldn't load leaderboard", http.StatusInternalServerError)
		return
	}

	entries := make([]leaderboardEntry, 0, len(allUsers))
	for _, u := range allUsers {
		if u.PredictionCount == 0 {
			continue
		}
		entries = append(entries, leaderboardEntry{
			UserID:          u.UserID,
			Handle:          u.Handle,
			AcesRadioPoints: u.AcesRadioPoints,
			Upper90Points:   u.Upper90Points,
			GrouchyPoints:   u.GrouchyPoints,
			HasProfile:      true,
		})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].AcesRadioPoints > entries[j].AcesRadioPoints })

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"entries": entries}); err != nil {
		log.Printf("leaderboard: encode response: %v", err)
	}
}
