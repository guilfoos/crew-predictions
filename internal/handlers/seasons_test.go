package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/guilfoos/crew-predictions/internal/handlers"
	"github.com/guilfoos/crew-predictions/internal/repository"
)

func TestSeasonsHandler_ListReturnsAllSeasons(t *testing.T) {
	sh := handlers.NewSeasonsHandler(repository.NewMemoryConfigStore("2026"))
	req := httptest.NewRequest(http.MethodGet, "/api/seasons", http.NoBody)
	w := httptest.NewRecorder()
	sh.APIList(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body struct {
		Seasons []struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			IsCurrent bool   `json:"isCurrent"`
		} `json:"seasons"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(body.Seasons) < 4 {
		t.Errorf("expected at least 4 seasons, got %d", len(body.Seasons))
	}
	if body.Seasons[0].ID != "2026" || body.Seasons[0].Name != "2026 Season" {
		t.Errorf("unexpected first season: %+v", body.Seasons[0])
	}
	if !body.Seasons[0].IsCurrent {
		t.Errorf("expected 2026 season to be current, got isCurrent=false")
	}
	if body.Seasons[1].IsCurrent {
		t.Errorf("expected 2027-sprint season to not be current")
	}
}
