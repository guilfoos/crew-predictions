package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/guilfoos/crew-predictions/internal/handlers"
	"github.com/guilfoos/crew-predictions/internal/models"
	"github.com/guilfoos/crew-predictions/internal/repository"
)

func TestSeedMatchEventsHandler_AppendsEventsToExistingMatch(t *testing.T) {
	store := repository.NewMemoryMatchStore()
	store.Seed([]models.Match{{
		ID: "m-1", HomeTeam: "Crew", AwayTeam: "FC Dallas", State: "in",
	}})

	h := handlers.NewSeedMatchEventsHandler(store)
	body := map[string]any{
		"matchID": "m-1",
		"events": []map[string]any{
			{"clock": "23'", "typeID": "goal", "team": "Crew", "players": []string{"Picard"}},
			{"clock": "39'", "typeID": "yellow-card", "team": "FC Dallas", "players": []string{"Smith"}},
		},
	}
	buf, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/admin/seed-match-events", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Submit(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}

	matches, _ := store.GetAll()
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	m := matches[0]
	if len(m.Events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(m.Events))
	}
	if m.Events[0].TypeID != "goal" || m.Events[0].Players[0] != "Picard" {
		t.Errorf("event 0 wrong: %+v", m.Events[0])
	}
	if m.Events[1].TypeID != "yellow-card" {
		t.Errorf("event 1 wrong: %+v", m.Events[1])
	}
}

func TestSeedMatchEventsHandler_Returns404ForUnknownMatch(t *testing.T) {
	store := repository.NewMemoryMatchStore()
	h := handlers.NewSeedMatchEventsHandler(store)
	body := []byte(`{"matchID":"unknown","events":[]}`)
	req := httptest.NewRequest("POST", "/admin/seed-match-events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Submit(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestSeedMatchEventsHandler_Returns500WhenGetAllFails(t *testing.T) {
	h := handlers.NewSeedMatchEventsHandler(repository.NewErrorGetAllMatchStore())
	body := []byte(`{"matchID":"m-1","events":[]}`)
	req := httptest.NewRequest("POST", "/admin/seed-match-events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Submit(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when MatchStore.GetAll fails, got %d", w.Code)
	}
}

func TestSeedMatchEventsHandler_Returns500WhenSaveAllFails(t *testing.T) {
	store := repository.NewErrorSaveAllMatchStore([]models.Match{{ID: "m-1"}})
	h := handlers.NewSeedMatchEventsHandler(store)
	body := []byte(`{"matchID":"m-1","events":[]}`)
	req := httptest.NewRequest("POST", "/admin/seed-match-events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Submit(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when MatchStore.SaveAll fails, got %d", w.Code)
	}
}

func TestSeedMatchEventsHandler_Returns400OnBadJSON(t *testing.T) {
	store := repository.NewMemoryMatchStore()
	h := handlers.NewSeedMatchEventsHandler(store)
	req := httptest.NewRequest("POST", "/admin/seed-match-events", bytes.NewReader([]byte(`not json`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Submit(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
