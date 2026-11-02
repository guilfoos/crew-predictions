package repository_test

import (
	"context"
	"testing"

	"github.com/guilfoos/crew-predictions/internal/repository"
)

func TestMemoryUserStore_UpsertAndGetByID(t *testing.T) {
	s := repository.NewMemoryUserStore()
	ctx := context.Background()

	u := repository.User{UserID: "firebase:abc", Handle: "crewfan", Provider: "google"}
	if err := s.Upsert(ctx, u); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := s.GetByID(ctx, "firebase:abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil || got.Handle != "crewfan" {
		t.Errorf("expected handle crewfan, got %+v", got)
	}
}

func TestMemoryUserStore_UpsertOverwritesHandle(t *testing.T) {
	s := repository.NewMemoryUserStore()
	ctx := context.Background()

	s.Upsert(ctx, repository.User{UserID: "firebase:abc", Handle: "old"})
	s.Upsert(ctx, repository.User{UserID: "firebase:abc", Handle: "new"})

	got, _ := s.GetByID(ctx, "firebase:abc")
	if got == nil || got.Handle != "new" {
		t.Errorf("expected handle new, got %+v", got)
	}
}

func TestMemoryUserStore_GetAll(t *testing.T) {
	s := repository.NewMemoryUserStore()
	ctx := context.Background()

	s.Upsert(ctx, repository.User{UserID: "u1", Handle: "fan1"})
	s.Upsert(ctx, repository.User{UserID: "u2", Handle: "fan2"})

	all, err := s.GetAll(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2 users, got %d", len(all))
	}
}

func TestMemoryUserStore_UpsertPreservesLocationWhenEmpty(t *testing.T) {
	s := repository.NewMemoryUserStore()
	ctx := context.Background()

	s.Upsert(ctx, repository.User{UserID: "u1", Handle: "fan", Location: "Columbus, OH"})
	s.Upsert(ctx, repository.User{UserID: "u1", Handle: "fan"}) // no location

	got, _ := s.GetByID(ctx, "u1")
	if got == nil || got.Location != "Columbus, OH" {
		t.Errorf("expected location Columbus, OH preserved, got %+v", got)
	}
}

func TestMemoryUserStore_UpsertPreservesProviderWhenEmpty(t *testing.T) {
	s := repository.NewMemoryUserStore()
	ctx := context.Background()

	s.Upsert(ctx, repository.User{UserID: "u1", Handle: "fan", Provider: "google.com"})
	s.Upsert(ctx, repository.User{UserID: "u1", Handle: "fan"}) // no provider

	got, _ := s.GetByID(ctx, "u1")
	if got == nil || got.Provider != "google.com" {
		t.Errorf("expected provider google.com preserved, got %+v", got)
	}
}

func TestMemoryUserStore_UpsertPreservesScoringFields(t *testing.T) {
	s := repository.NewMemoryUserStore()
	ctx := context.Background()

	s.Upsert(ctx, repository.User{UserID: "u1", Handle: "fan"})
	s.UpdateScores(ctx, "u1", 2, 15, 3, 1)                      // recalculator sets scores
	s.Upsert(ctx, repository.User{UserID: "u1", Handle: "fan"}) // auth handler upsert — no scoring fields

	got, _ := s.GetByID(ctx, "u1")
	if got == nil || got.AcesRadioPoints != 15 {
		t.Errorf("expected AcesRadioPoints 15 preserved after Upsert, got %+v", got)
	}
	if got.PredictionCount != 2 {
		t.Errorf("expected PredictionCount 2 preserved after Upsert, got %d", got.PredictionCount)
	}
}

func TestMemoryUserStore_UpdateScoresSetsFields(t *testing.T) {
	s := repository.NewMemoryUserStore()
	ctx := context.Background()

	s.Upsert(ctx, repository.User{UserID: "u1", Handle: "fan"})
	if err := s.UpdateScores(ctx, "u1", 3, 15, 6, 2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := s.GetByID(ctx, "u1")
	if got == nil || got.AcesRadioPoints != 15 {
		t.Errorf("expected AcesRadioPoints 15, got %+v", got)
	}
	if got.Upper90Points != 6 {
		t.Errorf("expected Upper90Points 6, got %d", got.Upper90Points)
	}
	if got.GrouchyPoints != 2 {
		t.Errorf("expected GrouchyPoints 2, got %d", got.GrouchyPoints)
	}
	if got.PredictionCount != 3 {
		t.Errorf("expected PredictionCount 3, got %d", got.PredictionCount)
	}
}

func TestMemoryUserStore_UpdateScoresCreatesEntryForNewUser(t *testing.T) {
	s := repository.NewMemoryUserStore()
	ctx := context.Background()

	// User not yet in store (prediction-only user)
	if err := s.UpdateScores(ctx, "google:NewFan", 1, 10, 0, 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := s.GetByID(ctx, "google:NewFan")
	if got == nil || got.AcesRadioPoints != 10 {
		t.Errorf("expected AcesRadioPoints 10 for new user, got %+v", got)
	}
}

func TestMemoryUserStore_ResetClearsAllUsers(t *testing.T) {
	s := repository.NewMemoryUserStore()
	ctx := context.Background()

	s.Upsert(ctx, repository.User{UserID: "u1", Handle: "fan"})
	s.Reset()

	all, _ := s.GetAll(ctx)
	if len(all) != 0 {
		t.Errorf("expected 0 users after Reset, got %d", len(all))
	}
}

func TestMemoryUserStore_GetByID_NotFound(t *testing.T) {
	s := repository.NewMemoryUserStore()
	got, err := repository.NewMemoryUserStore().GetByID(context.Background(), "nope")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for unknown user, got %+v", got)
	}
	_ = s
}
