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

func TestPollScoresHandler_CallsRecalcFnAfterSuccessfulPoll(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	matchStore.Seed([]models.Match{{
		ID: "m1", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas",
		Kickoff: time.Now().Add(-2 * time.Hour), Status: "STATUS_FULL_TIME", State: "post",
		HomeScore: "2", AwayScore: "0",
	}})
	fetcher := func() ([]models.Match, error) { return matchStore.GetAll() }

	called := 0
	recalcFn := func(_ context.Context) { called++ }
	h := handlers.NewPollScoresHandler(matchStore, repository.NewMemoryResultStore(), fetcher, recalcFn)
	req := httptest.NewRequest(http.MethodPost, "/admin/poll-scores", http.NoBody)
	w := httptest.NewRecorder()
	h.Poll(w, req)

	if called != 1 {
		t.Errorf("expected recalcFn called once, got %d", called)
	}
}

func TestPollScoresHandler_DoesNotCallRecalcFnOnFetcherFailure(t *testing.T) {
	called := 0
	recalcFn := func(_ context.Context) { called++ }
	fetcher := func() ([]models.Match, error) { return nil, fmt.Errorf("espn down") }
	h := handlers.NewPollScoresHandler(repository.NewMemoryMatchStore(), repository.NewMemoryResultStore(), fetcher, recalcFn)
	req := httptest.NewRequest(http.MethodPost, "/admin/poll-scores", http.NoBody)
	w := httptest.NewRecorder()
	h.Poll(w, req)
	if called != 0 {
		t.Errorf("expected recalcFn not called on fetch failure, got %d", called)
	}
}

func TestPollScoresHandler_Returns500WhenFetcherFails(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	resultStore := repository.NewMemoryResultStore()
	fetcher := func() ([]models.Match, error) { return nil, fmt.Errorf("espn down") }

	h := handlers.NewPollScoresHandler(matchStore, resultStore, fetcher, func(_ context.Context) {})
	req := httptest.NewRequest(http.MethodPost, "/admin/poll-scores", http.NoBody)
	w := httptest.NewRecorder()
	h.Poll(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestPollScoresHandler_CallsFetcherAndWritesResult(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	resultStore := repository.NewMemoryResultStore()

	matchStore.Seed([]models.Match{{
		ID: "m-done", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas",
		Kickoff: time.Now().Add(-2 * time.Hour), Status: "STATUS_FULL_TIME", State: "post",
		HomeScore: "2", AwayScore: "0",
	}})

	fetcher := func() ([]models.Match, error) { return matchStore.GetAll() }

	h := handlers.NewPollScoresHandler(matchStore, resultStore, fetcher, func(_ context.Context) {})
	req := httptest.NewRequest(http.MethodPost, "/admin/poll-scores", http.NoBody)
	w := httptest.NewRecorder()

	h.Poll(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
	result, err := resultStore.GetResult(context.Background(), "m-done")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil || result.HomeGoals != 2 || result.AwayGoals != 0 {
		t.Errorf("expected result 2-0, got %+v", result)
	}
}

func TestPollScoresHandler_EnqueuesNextTaskForInProgressMatch(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	matchStore.Seed([]models.Match{{
		ID: "m-live", HomeTeam: "Columbus Crew", AwayTeam: "LAFC",
		Kickoff: time.Now().Add(-20 * time.Minute), Status: "STATUS_IN_PROGRESS", State: "in",
	}})
	fetcher := func() ([]models.Match, error) { return matchStore.GetAll() }
	enqueuer := tasks.NewFakeEnqueuer()

	h := handlers.NewPollScoresHandler(matchStore, repository.NewMemoryResultStore(), fetcher, func(_ context.Context) {}).
		WithEnqueuer(enqueuer)

	req := httptest.NewRequest(http.MethodPost, "/admin/poll-scores?matchID=m-live", http.NoBody)
	w := httptest.NewRecorder()
	before := time.Now()
	h.Poll(w, req)
	after := time.Now()

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
	calls := enqueuer.Calls()
	if len(calls) != 1 {
		t.Fatalf("expected exactly one enqueued task for in-progress match, got %d", len(calls))
	}
	if calls[0].MatchID != "m-live" {
		t.Errorf("enqueued task matchID: got %q, want %q", calls[0].MatchID, "m-live")
	}
	minRunAt := before.Add(2 * time.Minute)
	maxRunAt := after.Add(2 * time.Minute)
	if calls[0].RunAt.Before(minRunAt) || calls[0].RunAt.After(maxRunAt) {
		t.Errorf("enqueued RunAt %v outside expected window [%v, %v]", calls[0].RunAt, minRunAt, maxRunAt)
	}
}

func TestPollScoresHandler_DoesNotEnqueueWhenMatchIsTerminal(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	matchStore.Seed([]models.Match{{
		ID: "m-done", HomeTeam: "Columbus Crew", AwayTeam: "LAFC",
		Kickoff: time.Now().Add(-2 * time.Hour), Status: "STATUS_FULL_TIME", State: "post",
		HomeScore: "2", AwayScore: "1",
	}})
	fetcher := func() ([]models.Match, error) { return matchStore.GetAll() }
	enqueuer := tasks.NewFakeEnqueuer()

	h := handlers.NewPollScoresHandler(matchStore, repository.NewMemoryResultStore(), fetcher, func(_ context.Context) {}).
		WithEnqueuer(enqueuer)

	req := httptest.NewRequest(http.MethodPost, "/admin/poll-scores?matchID=m-done", http.NoBody)
	w := httptest.NewRecorder()
	h.Poll(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
	if got := len(enqueuer.Calls()); got != 0 {
		t.Errorf("expected no enqueue for terminal match, got %d call(s)", got)
	}
}

func TestPollScoresHandler_SafetyBailoutMarksAbandonedAfter4h(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	matchStore.Seed([]models.Match{{
		ID: "m-runaway", HomeTeam: "Columbus Crew", AwayTeam: "LAFC",
		// 5h ago — past the 4h ceiling
		Kickoff: time.Now().Add(-5 * time.Hour), Status: "STATUS_IN_PROGRESS", State: "in",
	}})
	fetcher := func() ([]models.Match, error) { return matchStore.GetAll() }
	enqueuer := tasks.NewFakeEnqueuer()

	h := handlers.NewPollScoresHandler(matchStore, repository.NewMemoryResultStore(), fetcher, func(_ context.Context) {}).
		WithEnqueuer(enqueuer)

	req := httptest.NewRequest(http.MethodPost, "/admin/poll-scores?matchID=m-runaway", http.NoBody)
	w := httptest.NewRecorder()
	before := time.Now()
	h.Poll(w, req)
	after := time.Now()

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
	if got := len(enqueuer.Calls()); got != 0 {
		t.Errorf("safety bailout: expected NO chain enqueue, got %d", got)
	}

	stored, _ := matchStore.GetAll()
	var found *models.Match
	for i, m := range stored {
		if m.ID == "m-runaway" {
			found = &stored[i]
			break
		}
	}
	if found == nil {
		t.Fatal("expected m-runaway in store")
	}
	if found.AbandonedAt.IsZero() {
		t.Errorf("safety bailout: expected AbandonedAt to be set, got zero")
	}
	if found.AbandonedAt.Before(before) || found.AbandonedAt.After(after) {
		t.Errorf("safety bailout: AbandonedAt %v outside expected window [%v, %v]", found.AbandonedAt, before, after)
	}
}

func TestPollScoresHandler_NoBailoutWithin4h(t *testing.T) {
	// Sanity check: a match 3.5h past kickoff (still within safety window)
	// should still get a chain enqueue, NOT be marked abandoned.
	matchStore := repository.NewMemoryMatchStore()
	matchStore.Seed([]models.Match{{
		ID: "m-extra-time", HomeTeam: "Columbus Crew", AwayTeam: "LAFC",
		Kickoff: time.Now().Add(-3*time.Hour - 30*time.Minute), Status: "STATUS_IN_PROGRESS", State: "in",
	}})
	fetcher := func() ([]models.Match, error) { return matchStore.GetAll() }
	enqueuer := tasks.NewFakeEnqueuer()

	h := handlers.NewPollScoresHandler(matchStore, repository.NewMemoryResultStore(), fetcher, func(_ context.Context) {}).
		WithEnqueuer(enqueuer)

	req := httptest.NewRequest(http.MethodPost, "/admin/poll-scores?matchID=m-extra-time", http.NoBody)
	w := httptest.NewRecorder()
	h.Poll(w, req)

	if got := len(enqueuer.Calls()); got != 1 {
		t.Errorf("within 4h: expected 1 chain enqueue, got %d", got)
	}
	stored, _ := matchStore.GetAll()
	for _, m := range stored {
		if m.ID == "m-extra-time" && !m.AbandonedAt.IsZero() {
			t.Errorf("within 4h: AbandonedAt should be zero, got %v", m.AbandonedAt)
		}
	}
}

// errEnqueuer always fails on EnqueuePoll. Used to exercise the chain
// continuation error-logging branches without hitting real Cloud Tasks.
type errEnqueuer struct{}

func (errEnqueuer) EnqueuePoll(_ context.Context, _ string, _ time.Time) error {
	return fmt.Errorf("simulated enqueue failure")
}

// errGetAllMatchStore wraps a MatchStore and returns an error from GetAll
// after the first call, so PollOnce can succeed but maybeEnqueueNext's
// follow-up read fails.
type errGetAllMatchStore struct {
	repository.MatchStore
	calls int
}

func (e *errGetAllMatchStore) GetAll() ([]models.Match, error) {
	e.calls++
	if e.calls > 1 {
		return nil, fmt.Errorf("simulated GetAll failure")
	}
	return e.MatchStore.GetAll()
}

func TestPollScoresHandler_ChainEnqueueErrorIsLoggedAndSwallowed(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	matchStore.Seed([]models.Match{{
		ID: "m-live", HomeTeam: "Columbus Crew", AwayTeam: "LAFC",
		Kickoff: time.Now().Add(-30 * time.Minute), Status: "STATUS_IN_PROGRESS", State: "in",
	}})
	fetcher := func() ([]models.Match, error) { return matchStore.GetAll() }

	h := handlers.NewPollScoresHandler(matchStore, repository.NewMemoryResultStore(), fetcher, func(_ context.Context) {}).
		WithEnqueuer(errEnqueuer{})

	req := httptest.NewRequest(http.MethodPost, "/admin/poll-scores?matchID=m-live", http.NoBody)
	w := httptest.NewRecorder()
	h.Poll(w, req)

	// Handler must still respond 204 — chain enqueue failure is best-effort,
	// not a 500 (the poll itself succeeded).
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204 even when chain enqueue fails, got %d", w.Code)
	}
}

func TestPollScoresHandler_ChainContinuationSkipsNonMatchingMatches(t *testing.T) {
	// When multiple matches exist in the store, maybeEnqueueNext must iterate
	// past non-matching IDs until it finds the queried one. Covers the
	// `if m.ID != matchID { continue }` branch.
	matchStore := repository.NewMemoryMatchStore()
	matchStore.Seed([]models.Match{
		{ID: "m-other", HomeTeam: "Crew", AwayTeam: "RBNY", Kickoff: time.Now(), Status: "STATUS_FULL_TIME", State: "post"},
		{ID: "m-target", HomeTeam: "Crew", AwayTeam: "LAFC", Kickoff: time.Now().Add(-30 * time.Minute), Status: "STATUS_IN_PROGRESS", State: "in"},
	})
	fetcher := func() ([]models.Match, error) { return matchStore.GetAll() }
	enqueuer := tasks.NewFakeEnqueuer()

	h := handlers.NewPollScoresHandler(matchStore, repository.NewMemoryResultStore(), fetcher, func(_ context.Context) {}).
		WithEnqueuer(enqueuer)

	req := httptest.NewRequest(http.MethodPost, "/admin/poll-scores?matchID=m-target", http.NoBody)
	w := httptest.NewRecorder()
	h.Poll(w, req)

	calls := enqueuer.Calls()
	if len(calls) != 1 || calls[0].MatchID != "m-target" {
		t.Errorf("expected single enqueue for m-target, got %+v", calls)
	}
}

// errSaveAllMatchStore wraps a MatchStore and fails SaveAll only after the
// initial PollOnce save succeeds. This lets the chain bailout path's
// AbandonedAt persist call hit the error branch.
type errSaveAllMatchStore struct {
	repository.MatchStore
	saves int
}

func (e *errSaveAllMatchStore) SaveAll(m []models.Match) error {
	e.saves++
	if e.saves > 1 {
		return fmt.Errorf("simulated SaveAll failure")
	}
	return e.MatchStore.SaveAll(m)
}

func TestPollScoresHandler_BailoutSavePersistFailureIsLogged(t *testing.T) {
	// A 5h-old in-progress match triggers the safety bailout; if persisting
	// AbandonedAt back to the store fails, the handler logs and still 204s.
	inner := repository.NewMemoryMatchStore()
	inner.Seed([]models.Match{{
		ID: "m-abandoned", HomeTeam: "Columbus Crew", AwayTeam: "LAFC",
		Kickoff: time.Now().Add(-5 * time.Hour), Status: "STATUS_IN_PROGRESS", State: "in",
	}})
	matchStore := &errSaveAllMatchStore{MatchStore: inner}
	fetcher := func() ([]models.Match, error) { return inner.GetAll() }
	enqueuer := tasks.NewFakeEnqueuer()

	h := handlers.NewPollScoresHandler(matchStore, repository.NewMemoryResultStore(), fetcher, func(_ context.Context) {}).
		WithEnqueuer(enqueuer)

	req := httptest.NewRequest(http.MethodPost, "/admin/poll-scores?matchID=m-abandoned", http.NoBody)
	w := httptest.NewRecorder()
	h.Poll(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204 when bailout persist fails, got %d", w.Code)
	}
	if got := len(enqueuer.Calls()); got != 0 {
		t.Errorf("bailout path must not enqueue, got %d enqueues", got)
	}
}

func TestPollScoresHandler_ChainReadErrorIsLoggedAndSwallowed(t *testing.T) {
	inner := repository.NewMemoryMatchStore()
	inner.Seed([]models.Match{{
		ID: "m-live", HomeTeam: "Columbus Crew", AwayTeam: "LAFC",
		Kickoff: time.Now().Add(-30 * time.Minute), Status: "STATUS_IN_PROGRESS", State: "in",
	}})
	matchStore := &errGetAllMatchStore{MatchStore: inner}
	// fetcher uses the inner store directly so PollOnce succeeds; the wrapped
	// store's GetAll fails only on maybeEnqueueNext's later call.
	fetcher := func() ([]models.Match, error) { return inner.GetAll() }
	enqueuer := tasks.NewFakeEnqueuer()

	h := handlers.NewPollScoresHandler(matchStore, repository.NewMemoryResultStore(), fetcher, func(_ context.Context) {}).
		WithEnqueuer(enqueuer)

	req := httptest.NewRequest(http.MethodPost, "/admin/poll-scores?matchID=m-live", http.NoBody)
	w := httptest.NewRecorder()
	h.Poll(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204 even when chain read fails, got %d", w.Code)
	}
	if got := len(enqueuer.Calls()); got != 0 {
		t.Errorf("expected 0 enqueues when chain read fails, got %d", got)
	}
}
