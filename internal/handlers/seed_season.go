package handlers

import (
	"net/http"
	"strconv"

	"github.com/guilfoos/crew-predictions/internal/repository"
)

type SeedSeasonHandler struct {
	seasons repository.SeasonStore
}

func NewSeedSeasonHandler(seasons repository.SeasonStore) *SeedSeasonHandler {
	return &SeedSeasonHandler{seasons: seasons}
}

func (h *SeedSeasonHandler) Submit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	seasonID := r.FormValue("season_id")
	if seasonID == "" {
		http.Error(w, "season_id required", http.StatusBadRequest)
		return
	}
	handle := r.FormValue("entry_handle")
	aces, _ := strconv.Atoi(r.FormValue("entry_aces"))
	upper90, _ := strconv.Atoi(r.FormValue("entry_upper90"))
	grouchy, _ := strconv.Atoi(r.FormValue("entry_grouchy"))
	count, _ := strconv.Atoi(r.FormValue("entry_count"))

	ctx := r.Context()
	existing, _ := h.seasons.GetByID(ctx, seasonID)
	var snap repository.SeasonSnapshot
	if existing != nil {
		snap = *existing
	} else {
		snap = repository.SeasonSnapshot{ID: seasonID}
	}
	snap.Entries = append(snap.Entries, repository.SeasonEntry{
		Handle:          handle,
		AcesRadioPoints: aces,
		Upper90Points:   upper90,
		GrouchyPoints:   grouchy,
		PredictionCount: count,
		Rank:            len(snap.Entries) + 1,
	})
	if err := h.seasons.Save(ctx, snap); err != nil {
		http.Error(w, "save failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
