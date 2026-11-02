package handlers

import (
	"net/http"
	"time"

	"github.com/guilfoos/crew-predictions/internal/models"
	"github.com/guilfoos/crew-predictions/internal/repository"
)

func deriveStateFromStatus(status string) string {
	switch status {
	case "STATUS_FIRST_HALF", "STATUS_SECOND_HALF", "STATUS_HALFTIME",
		"STATUS_IN_PROGRESS", "STATUS_END_PERIOD", "STATUS_OVERTIME",
		"STATUS_EXTRA_TIME", "STATUS_SHOOTOUT":
		return "in"
	case "STATUS_FULL_TIME", "STATUS_FINAL", "STATUS_FT",
		"STATUS_FULL_PEN", "STATUS_ABANDONED":
		return "post"
	default:
		return "pre"
	}
}

type SeedMatchHandler struct {
	store *repository.MemoryMatchStore
}

func NewSeedMatchHandler(store *repository.MemoryMatchStore) *SeedMatchHandler {
	return &SeedMatchHandler{store: store}
}

func (h *SeedMatchHandler) Submit(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	if r.FormValue("id") == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	kickoff, err := time.Parse(time.RFC3339, r.FormValue("kickoff"))
	if err != nil {
		http.Error(w, "kickoff must be RFC3339", http.StatusBadRequest)
		return
	}
	state := r.FormValue("state")
	if state == "" {
		state = deriveStateFromStatus(r.FormValue("status"))
	}
	h.store.Seed([]models.Match{{
		ID:         r.FormValue("id"),
		HomeTeam:   r.FormValue("home_team"),
		AwayTeam:   r.FormValue("away_team"),
		Kickoff:    kickoff,
		Status:     r.FormValue("status"),
		State:      state,
		HomeScore:  r.FormValue("home_score"),
		AwayScore:  r.FormValue("away_score"),
		Venue:      r.FormValue("venue"),
		HomeRecord: r.FormValue("home_record"),
		AwayRecord: r.FormValue("away_record"),
		HomeForm:   r.FormValue("home_form"),
		AwayForm:   r.FormValue("away_form"),
	}})
	w.WriteHeader(http.StatusNoContent)
}
