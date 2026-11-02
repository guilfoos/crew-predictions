//go:build integration

package repository_test

import (
	"context"
	"os"
	"testing"

	"github.com/guilfoos/crew-predictions/internal/repository"
)

func firestoreUserStoreOrSkip(t *testing.T) *repository.FirestoreUserStore {
	t.Helper()
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("FIRESTORE_EMULATOR_HOST not set; skipping Firestore integration tests")
	}
	store, err := repository.NewFirestoreUserStore(context.Background(), "crew-predictions-test")
	if err != nil {
		t.Fatalf("failed to create FirestoreUserStore: %v", err)
	}
	return store
}

func TestFirestoreUserStore_UpsertPreservesProviderWhenEmpty(t *testing.T) {
	store := firestoreUserStoreOrSkip(t)
	ctx := context.Background()

	_ = store.Upsert(ctx, repository.User{UserID: "fs-user-provider", Handle: "fan", Provider: "google.com"})
	_ = store.Upsert(ctx, repository.User{UserID: "fs-user-provider", Handle: "fan"}) // empty provider

	got, err := store.GetByID(ctx, "fs-user-provider")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil || got.Provider != "google.com" {
		t.Errorf("expected provider google.com preserved, got %+v", got)
	}
}

func TestFirestoreUserStore_UpsertPreservesLocationWhenEmpty(t *testing.T) {
	store := firestoreUserStoreOrSkip(t)
	ctx := context.Background()

	_ = store.Upsert(ctx, repository.User{UserID: "fs-user-location", Handle: "fan", Location: "Columbus, OH"})
	_ = store.Upsert(ctx, repository.User{UserID: "fs-user-location", Handle: "fan"}) // no location

	got, err := store.GetByID(ctx, "fs-user-location")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil || got.Location != "Columbus, OH" {
		t.Errorf("expected location Columbus, OH preserved, got %+v", got)
	}
}

func TestFirestoreUserStore_UpdateScoresRoundTrips(t *testing.T) {
	store := firestoreUserStoreOrSkip(t)
	ctx := context.Background()

	_ = store.Upsert(ctx, repository.User{UserID: "fs-user-scores", Handle: "fan"})
	if err := store.UpdateScores(ctx, "fs-user-scores", 3, 15, 6, 1); err != nil {
		t.Fatalf("UpdateScores failed: %v", err)
	}

	got, err := store.GetByID(ctx, "fs-user-scores")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.AcesRadioPoints != 15 {
		t.Errorf("expected AcesRadioPoints 15, got %d", got.AcesRadioPoints)
	}
	if got.Upper90Points != 6 {
		t.Errorf("expected Upper90Points 6, got %d", got.Upper90Points)
	}
	if got.GrouchyPoints != 1 {
		t.Errorf("expected GrouchyPoints 1, got %d", got.GrouchyPoints)
	}
	if got.PredictionCount != 3 {
		t.Errorf("expected PredictionCount 3, got %d", got.PredictionCount)
	}
}

func TestFirestoreUserStore_UpsertPreservesScoringFields(t *testing.T) {
	store := firestoreUserStoreOrSkip(t)
	ctx := context.Background()

	_ = store.Upsert(ctx, repository.User{UserID: "fs-user-scores-preserve", Handle: "fan"})
	_ = store.UpdateScores(ctx, "fs-user-scores-preserve", 3, 15, 6, 1)
	_ = store.Upsert(ctx, repository.User{UserID: "fs-user-scores-preserve", Handle: "fan-updated"}) // auth handler upsert

	got, err := store.GetByID(ctx, "fs-user-scores-preserve")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.AcesRadioPoints != 15 {
		t.Errorf("expected AcesRadioPoints 15 preserved after Upsert, got %d", got.AcesRadioPoints)
	}
}

func TestFirestoreUserStore_UpsertUpdatesHandle(t *testing.T) {
	store := firestoreUserStoreOrSkip(t)
	ctx := context.Background()

	_ = store.Upsert(ctx, repository.User{UserID: "fs-user-handle", Handle: "old", Provider: "google.com"})
	_ = store.Upsert(ctx, repository.User{UserID: "fs-user-handle", Handle: "new", Provider: "google.com"})

	got, err := store.GetByID(ctx, "fs-user-handle")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil || got.Handle != "new" {
		t.Errorf("expected handle new, got %+v", got)
	}
}
