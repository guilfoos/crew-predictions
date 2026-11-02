package repository_test

import (
	"context"
	"testing"

	"github.com/guilfoos/crew-predictions/internal/repository"
)

func TestMemoryConfigStore_GetActiveSeason_ReturnsDefault(t *testing.T) {
	c := repository.NewMemoryConfigStore("2026")
	got := c.GetActiveSeason(context.Background())
	if got != "2026" {
		t.Errorf("expected default season 2026, got %q", got)
	}
}

func TestMemoryConfigStore_SetActiveSeason_ChangesValue(t *testing.T) {
	c := repository.NewMemoryConfigStore("2026")
	if err := c.SetActiveSeason(context.Background(), "2027-sprint"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := c.GetActiveSeason(context.Background())
	if got != "2027-sprint" {
		t.Errorf("expected 2027-sprint, got %q", got)
	}
}

func TestMemoryConfigStore_Reset_RestoresDefault(t *testing.T) {
	c := repository.NewMemoryConfigStore("2026")
	c.SetActiveSeason(context.Background(), "2027-sprint")
	c.Reset()
	got := c.GetActiveSeason(context.Background())
	if got != "2026" {
		t.Errorf("expected default 2026 after Reset, got %q", got)
	}
}
