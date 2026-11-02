package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/guilfoos/crew-predictions/internal/models"
	"github.com/guilfoos/crew-predictions/internal/repository"
)

type SeedMatchEventsHandler struct {
	store repository.MatchStore
}

// NewSeedMatchEventsHandler accepts the MatchStore interface (rather than the
// concrete *MemoryMatchStore) so error-path tests can inject failing stores.
func NewSeedMatchEventsHandler(store repository.MatchStore) *SeedMatchEventsHandler {
	return &SeedMatchEventsHandler{store: store}
}

type seedMatchEventsBody struct {
	MatchID string              `json:"matchID"`
	Events  []models.MatchEvent `json:"events"`
}

func (h *SeedMatchEventsHandler) Submit(w http.ResponseWriter, r *http.Request) {
	var body seedMatchEventsBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	matches, err := h.store.GetAll()
	if err != nil {
		http.Error(w, "could not load matches", http.StatusInternalServerError)
		return
	}
	found := false
	for i, m := range matches {
		if m.ID == body.MatchID {
			matches[i].Events = body.Events
			found = true
			break
		}
	}
	if !found {
		http.Error(w, "match not found", http.StatusNotFound)
		return
	}
	if err := h.store.SaveAll(matches); err != nil {
		http.Error(w, "could not save matches", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
