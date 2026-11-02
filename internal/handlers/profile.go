package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"

	"github.com/guilfoos/crew-predictions/internal/repository"
)

type ProfileHandler struct {
	users repository.UserStore
}

func NewProfileHandler(_ repository.PredictionStore, _ repository.ResultStore, users repository.UserStore, _ string) *ProfileHandler {
	return &ProfileHandler{users: users}
}

type standingEntry struct {
	Points int `json:"points"`
	Rank   int `json:"rank"`
}

func (h *ProfileHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := r.PathValue("userID")

	user, err := h.users.GetByID(ctx, userID)
	if err != nil {
		http.Error(w, "could not load user", http.StatusInternalServerError)
		return
	}
	if user == nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	allUsers, err := h.users.GetAll(ctx)
	if err != nil {
		http.Error(w, "could not load leaderboard", http.StatusInternalServerError)
		return
	}

	acesTotals := make(map[string]int, len(allUsers))
	u90Totals := make(map[string]int, len(allUsers))
	grouchyTotals := make(map[string]int, len(allUsers))
	for _, u := range allUsers {
		if u.PredictionCount > 0 {
			acesTotals[u.UserID] = u.AcesRadioPoints
			u90Totals[u.UserID] = u.Upper90Points
			grouchyTotals[u.UserID] = u.GrouchyPoints
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"userID":          user.UserID,
		"handle":          user.Handle,
		"location":        user.Location,
		"predictionCount": user.PredictionCount,
		"acesRadio":       rankFor(userID, acesTotals),
		"upper90Club":     rankFor(userID, u90Totals),
		"grouchy":         rankFor(userID, grouchyTotals),
	}); err != nil {
		log.Printf("profile: encode response: %v", err)
	}
}

func rankFor(userID string, totals map[string]int) standingEntry {
	points, ok := totals[userID]
	if !ok {
		return standingEntry{Points: 0, Rank: 0}
	}
	type kv struct {
		key    string
		points int
	}
	sorted := make([]kv, 0, len(totals))
	for k, v := range totals {
		sorted = append(sorted, kv{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].points > sorted[j].points })
	rank := 1
	for i, entry := range sorted {
		if i > 0 && entry.points < sorted[i-1].points {
			rank = i + 1
		}
		if entry.key == userID {
			return standingEntry{Points: points, Rank: rank}
		}
	}
	panic("rankFor: userID not found in sorted slice — this is a bug")
}
