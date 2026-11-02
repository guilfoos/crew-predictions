package repository_test

import (
	"testing"
	"time"

	"github.com/guilfoos/crew-predictions/internal/models"
	"github.com/guilfoos/crew-predictions/internal/repository"
)

func TestMemoryMatchStore_ReturnsSeededMatches(t *testing.T) {
	store := repository.NewMemoryMatchStore()
	kickoff := time.Now().Add(24 * time.Hour)
	store.Seed([]models.Match{
		{ID: "m1", HomeTeam: "Columbus Crew", AwayTeam: "LA Galaxy", Kickoff: kickoff, Status: "STATUS_SCHEDULED"},
	})

	matches, err := store.GetAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matches) != 1 || matches[0].ID != "m1" {
		t.Errorf("expected match m1, got %+v", matches)
	}
}

func TestMemoryMatchStore_SaveAll_ReplacesMatches(t *testing.T) {
	store := repository.NewMemoryMatchStore()
	store.Seed([]models.Match{{ID: "old", Kickoff: time.Now()}})

	err := store.SaveAll([]models.Match{{ID: "new", Kickoff: time.Now()}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	matches, _ := store.GetAll()
	if len(matches) != 1 || matches[0].ID != "new" {
		t.Errorf("expected only new match, got %+v", matches)
	}
}

func TestMemoryMatchStore_Reset_ClearsMatches(t *testing.T) {
	store := repository.NewMemoryMatchStore()
	store.Seed([]models.Match{{ID: "m1", Kickoff: time.Now()}})
	store.Reset()

	matches, err := store.GetAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matches) != 0 {
		t.Errorf("expected 0 matches after Reset, got %d", len(matches))
	}
}
