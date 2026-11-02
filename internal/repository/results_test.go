package repository_test

import (
	"context"
	"testing"

	"github.com/guilfoos/crew-predictions/internal/repository"
)

func TestMemoryResultStore_Reset_ClearsAllData(t *testing.T) {
	store := repository.NewMemoryResultStore()
	ctx := context.Background()
	_ = store.SaveResult(ctx, repository.Result{MatchID: "m1"})
	store.Reset()
	got, _ := store.GetResult(ctx, "m1")
	if got != nil {
		t.Errorf("expected nil after Reset, got %+v", got)
	}
}

func TestMemoryResultStore_ReturnsNilWhenNotFound(t *testing.T) {
	store := repository.NewMemoryResultStore()
	got, err := store.GetResult(context.Background(), "no-such-match")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for missing result, got %+v", got)
	}
}

func TestMemoryResultStore_SaveAndRetrieve(t *testing.T) {
	store := repository.NewMemoryResultStore()
	ctx := context.Background()

	result := repository.Result{MatchID: "match-1", HomeGoals: 2, AwayGoals: 0}
	if err := store.SaveResult(ctx, result); err != nil {
		t.Fatalf("unexpected error saving: %v", err)
	}

	got, err := store.GetResult(ctx, "match-1")
	if err != nil {
		t.Fatalf("unexpected error retrieving: %v", err)
	}
	if got == nil {
		t.Fatal("expected result, got nil")
	}
	if got.HomeGoals != 2 || got.AwayGoals != 0 {
		t.Errorf("expected 2-0, got %d-%d", got.HomeGoals, got.AwayGoals)
	}
}
