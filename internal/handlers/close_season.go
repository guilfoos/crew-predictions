package handlers

import (
	"net/http"
	"time"

	"github.com/guilfoos/crew-predictions/internal/repository"
	"github.com/guilfoos/crew-predictions/internal/seasons"
)

type CloseSeasonHandler struct {
	users  repository.UserStore
	snaps  repository.SeasonStore
	config repository.ConfigStore
}

func NewCloseSeasonHandler(users repository.UserStore, snaps repository.SeasonStore, config repository.ConfigStore) *CloseSeasonHandler {
	return &CloseSeasonHandler{users: users, snaps: snaps, config: config}
}

func (h *CloseSeasonHandler) Close(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	activeID := h.config.GetActiveSeason(ctx)

	if err := seasons.CloseSeason(ctx, activeID, h.users, h.snaps, time.Now()); err != nil {
		http.Error(w, "close season failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Advance to the next season.
	activeDef, _ := seasons.SeasonByID(activeID)
	nextDef := seasons.SeasonForDate(activeDef.End)
	if nextDef.ID != activeID {
		if err := h.config.SetActiveSeason(ctx, nextDef.ID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}
