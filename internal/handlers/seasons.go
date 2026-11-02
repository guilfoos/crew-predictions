package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/guilfoos/crew-predictions/internal/repository"
	"github.com/guilfoos/crew-predictions/internal/seasons"
)

type SeasonsHandler struct {
	config repository.ConfigStore
}

func NewSeasonsHandler(config repository.ConfigStore) *SeasonsHandler {
	return &SeasonsHandler{config: config}
}

type seasonResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IsCurrent bool   `json:"isCurrent"`
}

func (h *SeasonsHandler) APIList(w http.ResponseWriter, r *http.Request) {
	activeID := h.config.GetActiveSeason(r.Context())
	all := seasons.AllSeasons()
	out := make([]seasonResponse, len(all))
	for i, s := range all {
		out[i] = seasonResponse{ID: s.ID, Name: s.Name, IsCurrent: s.ID == activeID}
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"seasons": out}); err != nil {
		log.Printf("seasons: encode response: %v", err)
	}
}
