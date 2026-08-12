package handlers_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mcornell/crew-predictions/internal/handlers"
	"github.com/mcornell/crew-predictions/internal/models"
	"github.com/mcornell/crew-predictions/internal/repository"
	"github.com/mcornell/crew-predictions/internal/tasks"
)

func TestRefreshMatchesHandler_CallsOnRefreshCallback(t *testing.T) {
	store := repository.NewMemoryMatchStore()
	matches := []models.Match{
		{ID: "m-espn", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas", Kickoff: time.Now()},
	}
	fetcher := func() ([]models.Match, error) { return matches, nil }

	var called []models.Match
	h := handlers.NewRefreshMatchesHandler(store, fetcher, func(m []models.Match) { called = m })
	req := httptest.NewRequest(http.MethodPost, "/admin/refresh-matches", http.NoBody)
	w := httptest.NewRecorder()
	h.Refresh(w, req)

	if len(called) != 1 || called[0].ID != "m-espn" {
		t.Errorf("expected onRefresh called with fetched matches, got %+v", called)
	}
}

func TestRefreshMatchesHandler_SavesFetcherResultsToStore(t *testing.T) {
	store := repository.NewMemoryMatchStore()
	matches := []models.Match{
		{ID: "m-espn", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas", Kickoff: time.Now()},
	}
	fetcher := func() ([]models.Match, error) { return matches, nil }

	h := handlers.NewRefreshMatchesHandler(store, fetcher, nil)
	req := httptest.NewRequest(http.MethodPost, "/admin/refresh-matches", http.NoBody)
	w := httptest.NewRecorder()
	h.Refresh(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
	got, _ := store.GetAll()
	if len(got) != 1 || got[0].ID != "m-espn" {
		t.Errorf("expected 1 match with id m-espn, got %+v", got)
	}
}

type errSaveMatchStore struct{ repository.MatchStore }

func (e *errSaveMatchStore) SaveAll(_ []models.Match) error { return fmt.Errorf("store write failed") }

func TestRefreshMatchesHandler_Returns500WhenStoreSaveFails(t *testing.T) {
	store := &errSaveMatchStore{MatchStore: repository.NewMemoryMatchStore()}
	fetcher := func() ([]models.Match, error) {
		return []models.Match{{ID: "m1", Kickoff: time.Now()}}, nil
	}

	h := handlers.NewRefreshMatchesHandler(store, fetcher, nil)
	req := httptest.NewRequest(http.MethodPost, "/admin/refresh-matches", http.NoBody)
	w := httptest.NewRecorder()
	h.Refresh(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestRefreshMatchesHandler_Returns500WhenFetcherFails(t *testing.T) {
	store := repository.NewMemoryMatchStore()
	fetcher := func() ([]models.Match, error) { return nil, fmt.Errorf("espn down") }

	h := handlers.NewRefreshMatchesHandler(store, fetcher, nil)
	req := httptest.NewRequest(http.MethodPost, "/admin/refresh-matches", http.NoBody)
	w := httptest.NewRecorder()
	h.Refresh(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestRefreshMatchesHandler_SeedsChainTaskForUpcomingMatch(t *testing.T) {
	store := repository.NewMemoryMatchStore()
	kickoff := time.Now().Add(2 * time.Hour).UTC()
	matches := []models.Match{
		{ID: "m-soon", HomeTeam: "Columbus Crew", AwayTeam: "LAFC", Kickoff: kickoff, Status: "STATUS_SCHEDULED", State: "pre"},
	}
	fetcher := func() ([]models.Match, error) { return matches, nil }
	enqueuer := tasks.NewFakeEnqueuer()

	h := handlers.NewRefreshMatchesHandler(store, fetcher, nil).WithEnqueuer(enqueuer)
	req := httptest.NewRequest(http.MethodPost, "/admin/refresh-matches", http.NoBody)
	w := httptest.NewRecorder()
	h.Refresh(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
	calls := enqueuer.Calls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 enqueue for upcoming match, got %d", len(calls))
	}
	if calls[0].MatchID != "m-soon" {
		t.Errorf("enqueue matchID: got %q, want m-soon", calls[0].MatchID)
	}
	wantRunAt := kickoff.Add(-5 * time.Minute)
	if !calls[0].RunAt.Equal(wantRunAt) {
		t.Errorf("enqueue RunAt: got %v, want %v (kickoff - 5min)", calls[0].RunAt, wantRunAt)
	}

	stored, _ := store.GetAll()
	if len(stored) != 1 {
		t.Fatalf("expected 1 stored match, got %d", len(stored))
	}
	if !stored[0].ChainSeededFor.Equal(kickoff) {
		t.Errorf("ChainSeededFor: got %v, want %v (== Kickoff after seeding)", stored[0].ChainSeededFor, kickoff)
	}
}

func TestRefreshMatchesHandler_IdempotentWhenAlreadySeeded(t *testing.T) {
	// Pre-seed the store with a match that already has ChainSeededFor == Kickoff.
	kickoff := time.Now().Add(2 * time.Hour).UTC()
	store := repository.NewMemoryMatchStore()
	store.Seed([]models.Match{{
		ID: "m-soon", HomeTeam: "Columbus Crew", AwayTeam: "LAFC", Kickoff: kickoff,
		Status: "STATUS_SCHEDULED", State: "pre",
		ChainSeededFor: kickoff,
	}})
	// Fresh ESPN data returns the same match without ChainSeededFor (ESPN
	// doesn't know about it). The handler must merge from existing store state.
	fetcher := func() ([]models.Match, error) {
		return []models.Match{
			{ID: "m-soon", HomeTeam: "Columbus Crew", AwayTeam: "LAFC", Kickoff: kickoff, Status: "STATUS_SCHEDULED", State: "pre"},
		}, nil
	}
	enqueuer := tasks.NewFakeEnqueuer()

	h := handlers.NewRefreshMatchesHandler(store, fetcher, nil).WithEnqueuer(enqueuer)
	req := httptest.NewRequest(http.MethodPost, "/admin/refresh-matches", http.NoBody)
	h.Refresh(httptest.NewRecorder(), req)

	if got := len(enqueuer.Calls()); got != 0 {
		t.Errorf("idempotent seed: expected 0 enqueues, got %d", got)
	}
}

func TestRefreshMatchesHandler_RevivesDeadChainForInProgressMatch(t *testing.T) {
	// In-progress match with stale LastPollAt (10 min ago) → revival task.
	kickoff := time.Now().Add(-30 * time.Minute).UTC()
	stalePoll := time.Now().Add(-10 * time.Minute).UTC()
	store := repository.NewMemoryMatchStore()
	store.Seed([]models.Match{{
		ID: "m-stuck", HomeTeam: "Columbus Crew", AwayTeam: "LAFC", Kickoff: kickoff,
		Status: "STATUS_IN_PROGRESS", State: "in",
		LastPollAt: stalePoll,
	}})
	fetcher := func() ([]models.Match, error) {
		return []models.Match{
			{ID: "m-stuck", HomeTeam: "Columbus Crew", AwayTeam: "LAFC", Kickoff: kickoff, Status: "STATUS_IN_PROGRESS", State: "in"},
		}, nil
	}
	enqueuer := tasks.NewFakeEnqueuer()

	h := handlers.NewRefreshMatchesHandler(store, fetcher, nil).WithEnqueuer(enqueuer)
	req := httptest.NewRequest(http.MethodPost, "/admin/refresh-matches", http.NoBody)
	before := time.Now()
	h.Refresh(httptest.NewRecorder(), req)

	calls := enqueuer.Calls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 revival enqueue for stuck chain, got %d", len(calls))
	}
	if calls[0].MatchID != "m-stuck" {
		t.Errorf("revival matchID: got %q", calls[0].MatchID)
	}
	// Revival task is "immediate" (~ now), not kickoff-5min
	if calls[0].RunAt.Before(before) || calls[0].RunAt.After(before.Add(10*time.Second)) {
		t.Errorf("revival RunAt %v should be ~now (within 10s of %v)", calls[0].RunAt, before)
	}
}

func TestRefreshMatchesHandler_LeavesAliveChainAlone(t *testing.T) {
	// In-progress match with FRESH LastPollAt (1 min ago) → chain is alive, no enqueue.
	kickoff := time.Now().Add(-30 * time.Minute).UTC()
	freshPoll := time.Now().Add(-1 * time.Minute).UTC()
	store := repository.NewMemoryMatchStore()
	store.Seed([]models.Match{{
		ID: "m-healthy", HomeTeam: "Columbus Crew", AwayTeam: "LAFC", Kickoff: kickoff,
		Status: "STATUS_IN_PROGRESS", State: "in",
		LastPollAt: freshPoll,
	}})
	fetcher := func() ([]models.Match, error) {
		return []models.Match{
			{ID: "m-healthy", HomeTeam: "Columbus Crew", AwayTeam: "LAFC", Kickoff: kickoff, Status: "STATUS_IN_PROGRESS", State: "in"},
		}, nil
	}
	enqueuer := tasks.NewFakeEnqueuer()

	h := handlers.NewRefreshMatchesHandler(store, fetcher, nil).WithEnqueuer(enqueuer)
	req := httptest.NewRequest(http.MethodPost, "/admin/refresh-matches", http.NoBody)
	h.Refresh(httptest.NewRecorder(), req)

	if got := len(enqueuer.Calls()); got != 0 {
		t.Errorf("alive chain: expected 0 enqueues, got %d", got)
	}
}

func TestRefreshMatchesHandler_SkipsTerminalAndFarFutureMatches(t *testing.T) {
	// post match + kickoff > 8h out → both ignored
	store := repository.NewMemoryMatchStore()
	matches := []models.Match{
		{ID: "m-far", HomeTeam: "Columbus Crew", AwayTeam: "LAFC", Kickoff: time.Now().Add(12 * time.Hour), Status: "STATUS_SCHEDULED", State: "pre"},
		{ID: "m-done", HomeTeam: "Columbus Crew", AwayTeam: "LAFC", Kickoff: time.Now().Add(-2 * time.Hour), Status: "STATUS_FULL_TIME", State: "post"},
	}
	fetcher := func() ([]models.Match, error) { return matches, nil }
	enqueuer := tasks.NewFakeEnqueuer()

	h := handlers.NewRefreshMatchesHandler(store, fetcher, nil).WithEnqueuer(enqueuer)
	req := httptest.NewRequest(http.MethodPost, "/admin/refresh-matches", http.NoBody)
	h.Refresh(httptest.NewRecorder(), req)

	if got := len(enqueuer.Calls()); got != 0 {
		t.Errorf("far-future + terminal: expected 0 enqueues, got %d", got)
	}
}

type errEnqueuerRefresh struct{}

func (errEnqueuerRefresh) EnqueuePoll(_ context.Context, _ string, _ time.Time) error {
	return fmt.Errorf("simulated enqueue failure")
}

func TestRefreshMatchesHandler_EnqueueErrorIsLoggedAndDoesNotFailRefresh(t *testing.T) {
	// Refresh must still return 204 + persist matches when the enqueuer errors —
	// chain seeding is best-effort. The next refresh will retry.
	kickoff := time.Now().Add(2 * time.Hour).UTC()
	matches := []models.Match{
		{ID: "m-soon", HomeTeam: "Columbus Crew", AwayTeam: "LAFC", Kickoff: kickoff, Status: "STATUS_SCHEDULED", State: "pre"},
	}
	store := repository.NewMemoryMatchStore()
	fetcher := func() ([]models.Match, error) { return matches, nil }

	h := handlers.NewRefreshMatchesHandler(store, fetcher, nil).WithEnqueuer(errEnqueuerRefresh{})
	req := httptest.NewRequest(http.MethodPost, "/admin/refresh-matches", http.NoBody)
	w := httptest.NewRecorder()
	h.Refresh(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204 even when enqueue fails, got %d", w.Code)
	}
	stored, _ := store.GetAll()
	if len(stored) != 1 || stored[0].ID != "m-soon" {
		t.Errorf("expected match still persisted, got %+v", stored)
	}
}

type errGetAllRefreshStore struct {
	repository.MatchStore
}

func (e *errGetAllRefreshStore) GetAll() ([]models.Match, error) {
	return nil, fmt.Errorf("simulated GetAll failure")
}

func TestRefreshMatchesHandler_SoftFailsOnExistingReadError(t *testing.T) {
	// mergeChainFields swallows the GetAll error and proceeds with fresh-only.
	// Refresh still 204s and persists; LastPollAt/ChainSeededFor reset is the
	// accepted worst case (idempotent at the poll layer).
	store := &errGetAllRefreshStore{MatchStore: repository.NewMemoryMatchStore()}
	matches := []models.Match{
		{ID: "m-x", HomeTeam: "Columbus Crew", AwayTeam: "LAFC", Kickoff: time.Now()},
	}
	fetcher := func() ([]models.Match, error) { return matches, nil }

	h := handlers.NewRefreshMatchesHandler(store, fetcher, nil)
	req := httptest.NewRequest(http.MethodPost, "/admin/refresh-matches", http.NoBody)
	w := httptest.NewRecorder()
	h.Refresh(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204 on soft-fail merge, got %d", w.Code)
	}
}

func TestRefreshMatchesHandler_PreservesLastPollAtAcrossSaveAll(t *testing.T) {
	// Regression guard: SaveAll must not wipe LastPollAt/ChainSeededFor when
	// fresh ESPN data is merged over existing stored state.
	kickoff := time.Now().Add(-30 * time.Minute).UTC()
	freshPoll := time.Now().Add(-1 * time.Minute).UTC()
	store := repository.NewMemoryMatchStore()
	store.Seed([]models.Match{{
		ID: "m-preserve", HomeTeam: "Columbus Crew", AwayTeam: "LAFC", Kickoff: kickoff,
		Status: "STATUS_IN_PROGRESS", State: "in", LastPollAt: freshPoll,
	}})
	fetcher := func() ([]models.Match, error) {
		return []models.Match{
			{ID: "m-preserve", HomeTeam: "Columbus Crew", AwayTeam: "LAFC", Kickoff: kickoff, Status: "STATUS_IN_PROGRESS", State: "in", HomeScore: "1", AwayScore: "0"},
		}, nil
	}

	h := handlers.NewRefreshMatchesHandler(store, fetcher, nil).WithEnqueuer(tasks.NewFakeEnqueuer())
	req := httptest.NewRequest(http.MethodPost, "/admin/refresh-matches", http.NoBody)
	h.Refresh(httptest.NewRecorder(), req)

	stored, _ := store.GetAll()
	if len(stored) != 1 {
		t.Fatalf("expected 1 stored match, got %d", len(stored))
	}
	if !stored[0].LastPollAt.Equal(freshPoll) {
		t.Errorf("LastPollAt not preserved across refresh: got %v, want %v", stored[0].LastPollAt, freshPoll)
	}
	// Fresh ESPN data should overwrite scores
	if stored[0].HomeScore != "1" {
		t.Errorf("expected fresh score from ESPN to overwrite, got %q", stored[0].HomeScore)
	}
}
