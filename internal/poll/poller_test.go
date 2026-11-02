package poll_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/guilfoos/crew-predictions/internal/models"
	"github.com/guilfoos/crew-predictions/internal/poll"
	"github.com/guilfoos/crew-predictions/internal/repository"
)

func TestPollOnce_WritesResultForTerminalMatch(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	resultStore := repository.NewMemoryResultStore()

	matches := []models.Match{
		{
			ID:        "m-done",
			HomeTeam:  "Columbus Crew",
			AwayTeam:  "FC Dallas",
			Status:    "STATUS_FULL_TIME",
			State:     "post",
			HomeScore: "3",
			AwayScore: "1",
		},
	}
	fetcher := func() ([]models.Match, error) { return matches, nil }

	if err := poll.PollOnce(context.Background(), matchStore, resultStore, fetcher); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := resultStore.GetResult(context.Background(), "m-done")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected result to be saved, got nil")
	}
	if result.HomeGoals != 3 || result.AwayGoals != 1 {
		t.Errorf("expected 3-1, got %d-%d", result.HomeGoals, result.AwayGoals)
	}
}

func TestPollOnce_SkipsNonTerminalMatch(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	resultStore := repository.NewMemoryResultStore()

	matches := []models.Match{
		{ID: "m-live", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas", Status: "STATUS_IN_PROGRESS", State: "in", HomeScore: "1", AwayScore: "0"},
	}
	fetcher := func() ([]models.Match, error) { return matches, nil }

	if err := poll.PollOnce(context.Background(), matchStore, resultStore, fetcher); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, _ := resultStore.GetResult(context.Background(), "m-live")
	if result != nil {
		t.Errorf("expected no result saved for in-progress match, got %+v", result)
	}
}

func TestPollOnce_UpdatesMatchStoreWithFetchedData(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	resultStore := repository.NewMemoryResultStore()

	matches := []models.Match{
		{ID: "m-live", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas", Status: "STATUS_IN_PROGRESS", State: "in", HomeScore: "2", AwayScore: "0"},
	}
	fetcher := func() ([]models.Match, error) { return matches, nil }

	if err := poll.PollOnce(context.Background(), matchStore, resultStore, fetcher); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stored, _ := matchStore.GetAll()
	if len(stored) != 1 || stored[0].HomeScore != "2" {
		t.Errorf("expected matchStore updated with live score, got %+v", stored)
	}
}

func TestPollOnce_WritesResultForFinalAET(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	resultStore := repository.NewMemoryResultStore()

	matches := []models.Match{
		{ID: "m-aet", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas", Status: "STATUS_FINAL_AET", State: "post", HomeScore: "2", AwayScore: "2"},
	}
	fetcher := func() ([]models.Match, error) { return matches, nil }

	poll.PollOnce(context.Background(), matchStore, resultStore, fetcher)

	result, _ := resultStore.GetResult(context.Background(), "m-aet")
	if result == nil || result.HomeGoals != 2 || result.AwayGoals != 2 {
		t.Errorf("expected result for STATUS_FINAL_AET, got %+v", result)
	}
}

func TestPollOnce_WritesResultForFinalPEN(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	resultStore := repository.NewMemoryResultStore()

	matches := []models.Match{
		{ID: "m-pen", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas", Status: "STATUS_FINAL_PEN", State: "post", HomeScore: "1", AwayScore: "1"},
	}
	fetcher := func() ([]models.Match, error) { return matches, nil }

	poll.PollOnce(context.Background(), matchStore, resultStore, fetcher)

	result, _ := resultStore.GetResult(context.Background(), "m-pen")
	if result == nil {
		t.Errorf("expected result for STATUS_FINAL_PEN, got nil")
	}
}

func TestPollOnce_ReturnsErrorWhenSaveAllFails(t *testing.T) {
	resultStore := repository.NewMemoryResultStore()
	matches := []models.Match{
		{ID: "m1", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas", Status: "STATUS_FULL_TIME", HomeScore: "1", AwayScore: "0"},
	}
	fetcher := func() ([]models.Match, error) { return matches, nil }

	errStore := &errMatchStore{}
	err := poll.PollOnce(context.Background(), errStore, resultStore, fetcher)
	if err == nil {
		t.Error("expected error when SaveAll fails, got nil")
	}
}

type errMatchStore struct{}

func (e *errMatchStore) GetAll() ([]models.Match, error) { return nil, nil }
func (e *errMatchStore) SaveAll(_ []models.Match) error  { return fmt.Errorf("store failed") }
func (e *errMatchStore) Reset()                          {}

func TestSaveResult_SkipsNonNumericHomeScore(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	resultStore := repository.NewMemoryResultStore()
	matches := []models.Match{
		{ID: "m1", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas", Status: "STATUS_FULL_TIME", HomeScore: "TBD", AwayScore: "1"},
	}
	fetcher := func() ([]models.Match, error) { return matches, nil }

	if err := poll.PollOnce(context.Background(), matchStore, resultStore, fetcher); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result, _ := resultStore.GetResult(context.Background(), "m1")
	if result != nil {
		t.Errorf("expected no result saved for non-numeric home score, got %+v", result)
	}
}

func TestSaveResult_SkipsNonNumericAwayScore(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	resultStore := repository.NewMemoryResultStore()
	matches := []models.Match{
		{ID: "m1", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas", Status: "STATUS_FULL_TIME", HomeScore: "2", AwayScore: "TBD"},
	}
	fetcher := func() ([]models.Match, error) { return matches, nil }

	if err := poll.PollOnce(context.Background(), matchStore, resultStore, fetcher); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result, _ := resultStore.GetResult(context.Background(), "m1")
	if result != nil {
		t.Errorf("expected no result saved for non-numeric away score, got %+v", result)
	}
}

func TestPollOnce_ReturnsErrorWhenFetcherFails(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	resultStore := repository.NewMemoryResultStore()
	fetcher := func() ([]models.Match, error) { return nil, fmt.Errorf("espn down") }

	err := poll.PollOnce(context.Background(), matchStore, resultStore, fetcher)
	if err == nil {
		t.Error("expected error when fetcher fails, got nil")
	}
}

func TestPollOnce_ReturnsErrorWhenSaveResultFails(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	matches := []models.Match{
		{ID: "m1", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas", Status: "STATUS_FULL_TIME", HomeScore: "2", AwayScore: "1"},
	}
	fetcher := func() ([]models.Match, error) { return matches, nil }

	err := poll.PollOnce(context.Background(), matchStore, repository.NewErrorResultStore(), fetcher)
	if err == nil {
		t.Error("expected error when SaveResult fails, got nil")
	}
}

func TestPollOnce_IsIdempotent(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	resultStore := repository.NewMemoryResultStore()

	matches := []models.Match{
		{ID: "m-done", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas", Status: "STATUS_FULL_TIME", State: "post", HomeScore: "1", AwayScore: "0"},
	}
	fetcher := func() ([]models.Match, error) { return matches, nil }

	poll.PollOnce(context.Background(), matchStore, resultStore, fetcher)
	if err := poll.PollOnce(context.Background(), matchStore, resultStore, fetcher); err != nil {
		t.Fatalf("second poll failed: %v", err)
	}

	result, _ := resultStore.GetResult(context.Background(), "m-done")
	if result == nil || result.HomeGoals != 1 || result.AwayGoals != 0 {
		t.Errorf("expected idempotent result, got %+v", result)
	}
}

func TestPollOnce_StampsLastPollAtOnAllMatches(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	resultStore := repository.NewMemoryResultStore()

	matches := []models.Match{
		{ID: "m-live", HomeTeam: "Columbus Crew", AwayTeam: "LAFC", Status: "STATUS_IN_PROGRESS", State: "in"},
		{ID: "m-done", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas", Status: "STATUS_FULL_TIME", State: "post", HomeScore: "2", AwayScore: "0"},
	}
	fetcher := func() ([]models.Match, error) { return matches, nil }

	before := time.Now()
	if err := poll.PollOnce(context.Background(), matchStore, resultStore, fetcher); err != nil {
		t.Fatalf("PollOnce: %v", err)
	}
	after := time.Now()

	stored, err := matchStore.GetAll()
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(stored) != 2 {
		t.Fatalf("expected 2 stored matches, got %d", len(stored))
	}
	for _, m := range stored {
		if m.LastPollAt.IsZero() {
			t.Errorf("match %s: expected LastPollAt to be set, got zero value", m.ID)
		}
		if m.LastPollAt.Before(before) || m.LastPollAt.After(after) {
			t.Errorf("match %s: LastPollAt %v outside expected window [%v, %v]", m.ID, m.LastPollAt, before, after)
		}
	}
}

func TestPollOnce_PreservesChainSeededForFromExistingStore(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	seededFor := time.Date(2026, 5, 16, 23, 30, 0, 0, time.UTC)
	abandonedAt := time.Date(2026, 5, 14, 3, 30, 0, 0, time.UTC)
	matchStore.Seed([]models.Match{{
		ID: "m-live", Status: "STATUS_IN_PROGRESS", State: "in",
		Kickoff: seededFor,
		// These three were set by /admin/refresh-matches and a prior poll.
		ChainSeededFor: seededFor,
		AbandonedAt:    abandonedAt,
		LastPollAt:     time.Date(2026, 5, 16, 23, 28, 0, 0, time.UTC),
	}})
	// Fresh ESPN data doesn't have any of the chain-tracking fields.
	fetcher := func() ([]models.Match, error) {
		return []models.Match{
			{ID: "m-live", Status: "STATUS_IN_PROGRESS", State: "in", Kickoff: seededFor},
		}, nil
	}

	if err := poll.PollOnce(context.Background(), matchStore, repository.NewMemoryResultStore(), fetcher); err != nil {
		t.Fatalf("PollOnce: %v", err)
	}
	stored, _ := matchStore.GetAll()
	if len(stored) != 1 {
		t.Fatalf("expected 1 stored match, got %d", len(stored))
	}
	if !stored[0].ChainSeededFor.Equal(seededFor) {
		t.Errorf("ChainSeededFor wiped: got %v, want %v", stored[0].ChainSeededFor, seededFor)
	}
	if !stored[0].AbandonedAt.Equal(abandonedAt) {
		t.Errorf("AbandonedAt wiped: got %v, want %v", stored[0].AbandonedAt, abandonedAt)
	}
	if stored[0].LastPollAt.IsZero() {
		t.Errorf("LastPollAt should be stamped, got zero")
	}
}

func TestMergeChainFields_HandlesMatchNotInExistingStore(t *testing.T) {
	// Newly-discovered match comes through fetcher — has no existing record.
	// MergeChainFields should leave it unchanged (no prev to copy from).
	store := repository.NewMemoryMatchStore()
	store.Seed([]models.Match{{ID: "m-old", LastPollAt: time.Now().Add(-1 * time.Minute)}})
	fresh := []models.Match{
		{ID: "m-old", Status: "STATUS_IN_PROGRESS"},
		{ID: "m-new", Status: "STATUS_SCHEDULED"},
	}
	out := poll.MergeChainFields(store, fresh)
	if len(out) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(out))
	}
	// m-old should have LastPollAt merged from existing store
	if out[0].ID != "m-old" || out[0].LastPollAt.IsZero() {
		t.Errorf("m-old: expected LastPollAt copied from existing, got zero")
	}
	// m-new should remain unchanged (no existing record)
	if out[1].ID != "m-new" || !out[1].LastPollAt.IsZero() {
		t.Errorf("m-new: expected LastPollAt zero (no existing), got %v", out[1].LastPollAt)
	}
}

func TestMergeChainFields_ReturnsFreshOnStoreReadError(t *testing.T) {
	// If the store's GetAll fails, MergeChainFields should soft-fail (return
	// the fresh input unchanged) so the caller can still proceed.
	store := &errGetAllMatchStore{}
	fresh := []models.Match{
		{ID: "m-1", Status: "STATUS_IN_PROGRESS"},
	}
	out := poll.MergeChainFields(store, fresh)
	if len(out) != 1 || out[0].ID != "m-1" {
		t.Errorf("expected fresh returned unchanged on store error, got %+v", out)
	}
}

type errGetAllMatchStore struct{}

func (e *errGetAllMatchStore) GetAll() ([]models.Match, error) {
	return nil, fmt.Errorf("store read failed")
}
func (e *errGetAllMatchStore) SaveAll(_ []models.Match) error { return nil }
func (e *errGetAllMatchStore) Reset()                         {}
