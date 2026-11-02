//go:build integration

package repository_test

import (
	"context"
	"os"
	"testing"

	"github.com/guilfoos/crew-predictions/internal/repository"
)

func firestoreResultStoreOrSkip(t *testing.T) *repository.FirestoreResultStore {
	t.Helper()
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("FIRESTORE_EMULATOR_HOST not set; skipping Firestore integration tests")
	}
	store, err := repository.NewFirestoreResultStore(context.Background(), "crew-predictions-test")
	if err != nil {
		t.Fatalf("failed to create FirestoreResultStore: %v", err)
	}
	return store
}

func TestFirestoreResultStore_SaveAndRetrieve(t *testing.T) {
	store := firestoreResultStoreOrSkip(t)
	ctx := context.Background()

	r := repository.Result{MatchID: "fs-result-1", HomeTeam: "Columbus Crew", AwayTeam: "LA Galaxy", HomeGoals: 2, AwayGoals: 1}
	if err := store.SaveResult(ctx, r); err != nil {
		t.Fatalf("unexpected error saving: %v", err)
	}

	got, err := store.GetResult(ctx, "fs-result-1")
	if err != nil {
		t.Fatalf("unexpected error retrieving: %v", err)
	}
	if got == nil {
		t.Fatal("expected result, got nil")
	}
	if got.HomeGoals != 2 || got.AwayGoals != 1 {
		t.Errorf("expected 2-1, got %d-%d", got.HomeGoals, got.AwayGoals)
	}
}

func TestFirestoreResultStore_ReturnsNilWhenNotFound(t *testing.T) {
	store := firestoreResultStoreOrSkip(t)
	got, err := store.GetResult(context.Background(), "no-such-match")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for missing result, got %+v", got)
	}
}

func TestFirestoreResultStore_OverwritesExisting(t *testing.T) {
	store := firestoreResultStoreOrSkip(t)
	ctx := context.Background()

	_ = store.SaveResult(ctx, repository.Result{MatchID: "fs-result-overwrite", HomeGoals: 1, AwayGoals: 0})
	_ = store.SaveResult(ctx, repository.Result{MatchID: "fs-result-overwrite", HomeGoals: 3, AwayGoals: 2})

	got, err := store.GetResult(ctx, "fs-result-overwrite")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.HomeGoals != 3 || got.AwayGoals != 2 {
		t.Errorf("expected 3-2 after overwrite, got %d-%d", got.HomeGoals, got.AwayGoals)
	}
}
