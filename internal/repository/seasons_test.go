package repository_test

import (
	"context"
	"testing"

	"github.com/guilfoos/crew-predictions/internal/repository"
)

func TestMemorySeasonStore_SaveAndGetByID(t *testing.T) {
	s := repository.NewMemorySeasonStore()
	ctx := context.Background()

	snap := repository.SeasonSnapshot{
		ID:   "2026",
		Name: "2026 Season",
		Entries: []repository.SeasonEntry{
			{Handle: "HistoryFan", AcesRadioPoints: 15, Upper90Points: 3, GrouchyPoints: 1, PredictionCount: 5, Rank: 1},
		},
	}
	if err := s.Save(ctx, snap); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := s.GetByID(ctx, "2026")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got == nil {
		t.Fatal("expected snapshot, got nil")
	}
	if got.ID != "2026" || len(got.Entries) != 1 || got.Entries[0].Handle != "HistoryFan" || got.Entries[0].AcesRadioPoints != 15 {
		t.Errorf("unexpected snapshot: %+v", got)
	}
}

func TestMemorySeasonStore_GetByID_NotFound(t *testing.T) {
	s := repository.NewMemorySeasonStore()
	got, err := s.GetByID(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for missing season, got %+v", got)
	}
}

func TestMemorySeasonStore_Exists(t *testing.T) {
	s := repository.NewMemorySeasonStore()
	ctx := context.Background()

	if s.Exists(ctx, "2026") {
		t.Error("expected Exists=false before Save")
	}
	s.Save(ctx, repository.SeasonSnapshot{ID: "2026"})
	if !s.Exists(ctx, "2026") {
		t.Error("expected Exists=true after Save")
	}
}

func TestMemorySeasonStore_ListAll(t *testing.T) {
	s := repository.NewMemorySeasonStore()
	ctx := context.Background()
	s.Save(ctx, repository.SeasonSnapshot{ID: "2026"})
	s.Save(ctx, repository.SeasonSnapshot{ID: "2027-sprint"})

	all, err := s.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2 seasons, got %d", len(all))
	}
}

func TestMemorySeasonStore_Reset(t *testing.T) {
	s := repository.NewMemorySeasonStore()
	ctx := context.Background()
	s.Save(ctx, repository.SeasonSnapshot{ID: "2026"})
	s.Reset()

	all, _ := s.ListAll(ctx)
	if len(all) != 0 {
		t.Errorf("expected empty after Reset, got %d", len(all))
	}
}
