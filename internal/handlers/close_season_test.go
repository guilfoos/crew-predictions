package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/guilfoos/crew-predictions/internal/handlers"
	"github.com/guilfoos/crew-predictions/internal/repository"
)

func TestCloseSeasonHandler_ClosesActiveSeasonAndReturns204(t *testing.T) {
	ctx := context.Background()
	users := repository.NewMemoryUserStore()
	snaps := repository.NewMemorySeasonStore()
	cfg := repository.NewMemoryConfigStore("2026")

	users.Upsert(ctx, repository.User{UserID: "u1", Handle: "Fan"})
	users.UpdateScores(ctx, "u1", 1, 15, 0, 0)

	h := handlers.NewCloseSeasonHandler(users, snaps, cfg)
	req := httptest.NewRequest(http.MethodPost, "/admin/seasons/close", http.NoBody)
	rr := httptest.NewRecorder()
	h.Close(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rr.Code)
	}

	snap, err := snaps.GetByID(ctx, "2026")
	if err != nil || snap == nil {
		t.Fatalf("expected snapshot to exist, got err=%v snap=%v", err, snap)
	}
	if len(snap.Entries) != 1 || snap.Entries[0].Handle != "Fan" {
		t.Errorf("unexpected entries: %+v", snap.Entries)
	}

	u, _ := users.GetByID(ctx, "u1")
	if u.AcesRadioPoints != 0 {
		t.Errorf("expected scores reset to 0, got %d", u.AcesRadioPoints)
	}
}

func TestCloseSeasonHandler_Returns500WhenCloseSeasonFails(t *testing.T) {
	cfg := repository.NewMemoryConfigStore("2026")
	h := handlers.NewCloseSeasonHandler(repository.NewErrorGetAllUserStore(), repository.NewMemorySeasonStore(), cfg)
	req := httptest.NewRequest(http.MethodPost, "/admin/seasons/close", http.NoBody)
	rr := httptest.NewRecorder()
	h.Close(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

func TestCloseSeasonHandler_AdvancesActiveSeason(t *testing.T) {
	ctx := context.Background()
	users := repository.NewMemoryUserStore()
	snaps := repository.NewMemorySeasonStore()
	cfg := repository.NewMemoryConfigStore("2026")

	h := handlers.NewCloseSeasonHandler(users, snaps, cfg)
	req := httptest.NewRequest(http.MethodPost, "/admin/seasons/close", http.NoBody)
	rr := httptest.NewRecorder()
	h.Close(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d body=%s", rr.Code, rr.Body.String())
	}

	active := cfg.GetActiveSeason(ctx)
	if active != "2027-sprint" {
		t.Errorf("expected active season to advance to 2027-sprint, got %q", active)
	}
}
