package repository_test

import (
	"context"
	"sync"
	"testing"

	"github.com/guilfoos/crew-predictions/internal/repository"
)

func TestMemoryPredictionStore_Reset_ClearsAllData(t *testing.T) {
	store := repository.NewMemoryPredictionStore()
	ctx := context.Background()
	_ = store.Save(ctx, repository.Prediction{MatchID: "m1", UserID: "u1"})
	store.Reset()
	all, _ := store.GetAll(ctx)
	if len(all) != 0 {
		t.Errorf("expected 0 predictions after Reset, got %d", len(all))
	}
}

func TestMemoryPredictionStore_ReturnsNilWhenNotFound(t *testing.T) {
	store := repository.NewMemoryPredictionStore()
	got, err := store.GetByMatchAndUser(context.Background(), "no-match", "nobody")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for missing prediction, got %+v", got)
	}
}

func TestMemoryPredictionStore_GetAllReturnsAllSaved(t *testing.T) {
	store := repository.NewMemoryPredictionStore()
	ctx := context.Background()

	store.Save(ctx, repository.Prediction{MatchID: "m1", UserID: "google:user1", HomeGoals: 1, AwayGoals: 0})
	store.Save(ctx, repository.Prediction{MatchID: "m2", UserID: "google:user2", HomeGoals: 2, AwayGoals: 1})

	all, err := store.GetAll(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2 predictions, got %d", len(all))
	}
}

func TestMemoryPredictionStore_ConcurrentSave(t *testing.T) {
	store := repository.NewMemoryPredictionStore()
	ctx := context.Background()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			store.Save(ctx, repository.Prediction{MatchID: "m", UserID: "u", HomeGoals: i})
		}(i)
	}
	wg.Wait()
}

func TestMemoryPredictionStore_GetByMatch_ReturnsOnlyMatchPredictions(t *testing.T) {
	store := repository.NewMemoryPredictionStore()
	ctx := context.Background()

	store.Save(ctx, repository.Prediction{MatchID: "m1", UserID: "google:u1", HomeGoals: 2, AwayGoals: 1})
	store.Save(ctx, repository.Prediction{MatchID: "m1", UserID: "google:u2", HomeGoals: 0, AwayGoals: 0})
	store.Save(ctx, repository.Prediction{MatchID: "m2", UserID: "google:u3", HomeGoals: 1, AwayGoals: 1})

	got, err := store.GetByMatch(ctx, "m1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 predictions for m1, got %d", len(got))
	}
	for _, p := range got {
		if p.MatchID != "m1" {
			t.Errorf("expected all predictions to be for m1, got %q", p.MatchID)
		}
	}
}

func TestMemoryPredictionStore_GetByMatch_ReturnsEmptyForNoMatch(t *testing.T) {
	store := repository.NewMemoryPredictionStore()
	ctx := context.Background()

	got, err := store.GetByMatch(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 predictions, got %d", len(got))
	}
}

func TestMemoryPredictionStore_SaveAndRetrieve(t *testing.T) {
	store := repository.NewMemoryPredictionStore()
	ctx := context.Background()

	pred := repository.Prediction{
		MatchID:   "match1",
		UserID:    "google:abc123",
		HomeGoals: 3,
		AwayGoals: 1,
	}

	if err := store.Save(ctx, pred); err != nil {
		t.Fatalf("unexpected error saving: %v", err)
	}

	got, err := store.GetByMatchAndUser(ctx, "match1", "google:abc123")
	if err != nil {
		t.Fatalf("unexpected error retrieving: %v", err)
	}
	if got.HomeGoals != 3 || got.AwayGoals != 1 {
		t.Errorf("expected 3-1, got %d-%d", got.HomeGoals, got.AwayGoals)
	}
}
