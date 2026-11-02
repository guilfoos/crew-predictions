package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/guilfoos/crew-predictions/internal/handlers"
	"github.com/guilfoos/crew-predictions/internal/repository"
)

func TestSeedMatchHandler_SavesMatch(t *testing.T) {
	store := repository.NewMemoryMatchStore()
	h := handlers.NewSeedMatchHandler(store)

	req := httptest.NewRequest(http.MethodPost, "/admin/seed-match",
		strings.NewReader("id=m1&home_team=Columbus+Crew&away_team=LA+Galaxy&kickoff=2026-05-01T19:30:00Z&status=STATUS_SCHEDULED"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Submit(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
	matches, err := store.GetAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matches) != 1 || matches[0].ID != "m1" {
		t.Errorf("expected match m1, got %+v", matches)
	}
}

func TestSeedMatchHandler_SavesMatchWithState(t *testing.T) {
	store := repository.NewMemoryMatchStore()
	h := handlers.NewSeedMatchHandler(store)

	req := httptest.NewRequest(http.MethodPost, "/admin/seed-match",
		strings.NewReader("id=m2&home_team=Columbus+Crew&away_team=LA+Galaxy&kickoff=2026-05-01T19:30:00Z&status=STATUS_SCHEDULED&state=in"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Submit(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
	matches, _ := store.GetAll()
	if len(matches) != 1 || matches[0].State != "in" {
		t.Errorf("expected state=in, got %+v", matches)
	}
}

func TestSeedMatchHandler_SavesMatchWithScores(t *testing.T) {
	store := repository.NewMemoryMatchStore()
	h := handlers.NewSeedMatchHandler(store)

	req := httptest.NewRequest(http.MethodPost, "/admin/seed-match",
		strings.NewReader("id=m3&home_team=Columbus+Crew&away_team=FC+Dallas&kickoff=2026-05-01T19:30:00Z&status=STATUS_FULL_TIME&state=post&home_score=3&away_score=1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Submit(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
	matches, _ := store.GetAll()
	if len(matches) != 1 || matches[0].HomeScore != "3" || matches[0].AwayScore != "1" {
		t.Errorf("expected HomeScore=3 AwayScore=1, got %+v", matches)
	}
}

func TestSeedMatchHandler_DerivesStateFromStatusWhenEmpty(t *testing.T) {
	cases := []struct {
		status        string
		expectedState string
	}{
		{"STATUS_SCHEDULED", "pre"},
		{"STATUS_FULL_TIME", "post"},
		{"STATUS_FINAL", "post"},
		{"STATUS_FIRST_HALF", "in"},
		{"STATUS_IN_PROGRESS", "in"},
	}
	for _, tc := range cases {
		store := repository.NewMemoryMatchStore()
		h := handlers.NewSeedMatchHandler(store)
		body := "id=mx&home_team=Columbus+Crew&away_team=LA+Galaxy&kickoff=2026-05-01T19:30:00Z&status=" + tc.status
		req := httptest.NewRequest(http.MethodPost, "/admin/seed-match", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		h.Submit(w, req)
		matches, _ := store.GetAll()
		if len(matches) != 1 || matches[0].State != tc.expectedState {
			t.Errorf("status=%s: expected state=%q, got %q", tc.status, tc.expectedState, matches[0].State)
		}
	}
}

func TestSeedMatchHandler_RejectsEmptyID(t *testing.T) {
	h := handlers.NewSeedMatchHandler(repository.NewMemoryMatchStore())
	req := httptest.NewRequest(http.MethodPost, "/admin/seed-match",
		strings.NewReader("id=&home_team=Columbus+Crew&away_team=LA+Galaxy&kickoff=2026-05-01T19:30:00Z&status=STATUS_SCHEDULED"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Submit(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty id, got %d", w.Code)
	}
}

func TestSeedMatchHandler_SavesVenue(t *testing.T) {
	store := repository.NewMemoryMatchStore()
	h := handlers.NewSeedMatchHandler(store)

	req := httptest.NewRequest(http.MethodPost, "/admin/seed-match",
		strings.NewReader("id=m1&home_team=Columbus+Crew&away_team=FC+Dallas&kickoff=2026-05-01T19:30:00Z&status=STATUS_SCHEDULED&venue=ScottsMiracle-Gro+Field"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Submit(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
	matches, err := store.GetAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matches) != 1 || matches[0].Venue != "ScottsMiracle-Gro Field" {
		t.Errorf("expected venue 'ScottsMiracle-Gro Field', got %+v", matches)
	}
}

func TestSeedMatchHandler_SavesRecordsAndForm(t *testing.T) {
	store := repository.NewMemoryMatchStore()
	h := handlers.NewSeedMatchHandler(store)

	req := httptest.NewRequest(http.MethodPost, "/admin/seed-match",
		strings.NewReader("id=m1&home_team=Columbus+Crew&away_team=FC+Dallas&kickoff=2026-05-01T19:30:00Z&status=STATUS_SCHEDULED&home_record=5-3-2&away_record=4-4-2&home_form=WWWLL&away_form=LWDWL"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Submit(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
	matches, _ := store.GetAll()
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	m := matches[0]
	if m.HomeRecord != "5-3-2" {
		t.Errorf("HomeRecord: got %q, want %q", m.HomeRecord, "5-3-2")
	}
	if m.AwayRecord != "4-4-2" {
		t.Errorf("AwayRecord: got %q, want %q", m.AwayRecord, "4-4-2")
	}
	if m.HomeForm != "WWWLL" {
		t.Errorf("HomeForm: got %q, want %q", m.HomeForm, "WWWLL")
	}
	if m.AwayForm != "LWDWL" {
		t.Errorf("AwayForm: got %q, want %q", m.AwayForm, "LWDWL")
	}
}

func TestSeedMatchHandler_RejectsBadKickoff(t *testing.T) {
	h := handlers.NewSeedMatchHandler(repository.NewMemoryMatchStore())
	req := httptest.NewRequest(http.MethodPost, "/admin/seed-match",
		strings.NewReader("id=m1&home_team=Columbus+Crew&away_team=LA+Galaxy&kickoff=not-a-date&status=STATUS_SCHEDULED"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.Submit(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
