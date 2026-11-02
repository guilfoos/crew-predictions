package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/guilfoos/crew-predictions/internal/handlers"
	"github.com/guilfoos/crew-predictions/internal/repository"
)

func TestSeedSeasonHandler_SavesSnapshot(t *testing.T) {
	store := repository.NewMemorySeasonStore()
	sh := handlers.NewSeedSeasonHandler(store)

	req := httptest.NewRequest(http.MethodPost, "/admin/seed-season",
		strings.NewReader("season_id=2026&entry_handle=HistoryFan&entry_aces=15&entry_upper90=3&entry_grouchy=1&entry_count=5"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	sh.Submit(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
	snap, err := store.GetByID(context.Background(), "2026")
	if err != nil || snap == nil {
		t.Fatalf("expected snapshot, got nil (err=%v)", err)
	}
	if len(snap.Entries) != 1 || snap.Entries[0].Handle != "HistoryFan" || snap.Entries[0].AcesRadioPoints != 15 {
		t.Errorf("unexpected snapshot: %+v", snap)
	}
}

func TestSeedSeasonHandler_Returns400WhenSeasonIDMissing(t *testing.T) {
	sh := handlers.NewSeedSeasonHandler(repository.NewMemorySeasonStore())
	req := httptest.NewRequest(http.MethodPost, "/admin/seed-season",
		strings.NewReader("entry_handle=NoSeason"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	sh.Submit(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when season_id missing, got %d", w.Code)
	}
}

func TestSeedSeasonHandler_Returns500WhenSaveFails(t *testing.T) {
	sh := handlers.NewSeedSeasonHandler(repository.NewErrorSeasonStore())
	req := httptest.NewRequest(http.MethodPost, "/admin/seed-season",
		strings.NewReader("season_id=2026&entry_handle=Fan&entry_aces=10"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	sh.Submit(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when SeasonStore.Save fails, got %d", w.Code)
	}
}

func TestSeedSeasonHandler_MultipleCallsAppendEntries(t *testing.T) {
	store := repository.NewMemorySeasonStore()
	sh := handlers.NewSeedSeasonHandler(store)

	for _, handle := range []string{"Fan1", "Fan2"} {
		req := httptest.NewRequest(http.MethodPost, "/admin/seed-season",
			strings.NewReader("season_id=2026&entry_handle="+handle+"&entry_aces=10"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		sh.Submit(httptest.NewRecorder(), req)
	}

	snap, _ := store.GetByID(context.Background(), "2026")
	if len(snap.Entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(snap.Entries))
	}
}
