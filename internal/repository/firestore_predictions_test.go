//go:build integration

package repository_test

import (
	"context"
	"os"
	"testing"

	"github.com/guilfoos/crew-predictions/internal/repository"
)

func firestoreOrSkip(t *testing.T) *repository.FirestorePredictionStore {
	t.Helper()
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("FIRESTORE_EMULATOR_HOST not set; skipping Firestore integration tests")
	}
	store, err := repository.NewFirestorePredictionStore(context.Background(), "crew-predictions-test")
	if err != nil {
		t.Fatalf("failed to create FirestorePredictionStore: %v", err)
	}
	return store
}

func TestFirestorePredictionStore_SaveAndRetrieve(t *testing.T) {
	store := firestoreOrSkip(t)
	ctx := context.Background()

	pred := repository.Prediction{
		MatchID:   "firestore-match-1",
		UserID:    "google:nordecke123",
		HomeGoals: 2,
		AwayGoals: 0,
	}

	if err := store.Save(ctx, pred); err != nil {
		t.Fatalf("unexpected error saving: %v", err)
	}

	got, err := store.GetByMatchAndUser(ctx, "firestore-match-1", "google:nordecke123")
	if err != nil {
		t.Fatalf("unexpected error retrieving: %v", err)
	}
	if got == nil {
		t.Fatal("expected prediction, got nil")
	}
	if got.HomeGoals != 2 || got.AwayGoals != 0 {
		t.Errorf("expected 2-0, got %d-%d", got.HomeGoals, got.AwayGoals)
	}
}

func TestFirestorePredictionStore_GetAllReturnsSaved(t *testing.T) {
	store := firestoreOrSkip(t)
	ctx := context.Background()

	store.Save(ctx, repository.Prediction{MatchID: "getall-m1", UserID: "google:user-ga1", HomeGoals: 1, AwayGoals: 0})
	store.Save(ctx, repository.Prediction{MatchID: "getall-m2", UserID: "google:user-ga2", HomeGoals: 2, AwayGoals: 1})

	all, err := store.GetAll(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(all) < 2 {
		t.Errorf("expected at least 2 predictions, got %d", len(all))
	}
}

func TestFirestorePredictionStore_ReturnsNilWhenNotFound(t *testing.T) {
	store := firestoreOrSkip(t)
	got, err := store.GetByMatchAndUser(context.Background(), "no-such-match", "google:nobody")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for missing prediction, got %+v", got)
	}
}
