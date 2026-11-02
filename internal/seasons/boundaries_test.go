package seasons_test

import (
	"testing"
	"time"

	"github.com/guilfoos/crew-predictions/internal/seasons"
)

func TestAllSeasons(t *testing.T) {
	all := seasons.AllSeasons()
	if len(all) < 4 {
		t.Fatalf("expected at least 4 seasons, got %d", len(all))
	}
	// Verify ordering: each start == previous end
	for i := 1; i < len(all); i++ {
		if !all[i].Start.Equal(all[i-1].End) {
			t.Errorf("season %s start %v != previous season %s end %v", all[i].ID, all[i].Start, all[i-1].ID, all[i-1].End)
		}
	}
}

func TestSeasonByID(t *testing.T) {
	s, ok := seasons.SeasonByID("2026")
	if !ok {
		t.Fatal("expected 2026 to exist")
	}
	if s.Name != "2026 Season" {
		t.Errorf("expected name '2026 Season', got %q", s.Name)
	}

	s, ok = seasons.SeasonByID("2027-sprint")
	if !ok {
		t.Fatal("expected 2027-sprint to exist")
	}
	if s.Name != "2027 Sprint Season" {
		t.Errorf("expected name '2027 Sprint Season', got %q", s.Name)
	}

	_, ok = seasons.SeasonByID("nonexistent")
	if ok {
		t.Error("expected nonexistent season to not be found")
	}
}

func TestSeasonForDate(t *testing.T) {
	tests := []struct {
		date   time.Time
		wantID string
	}{
		{time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), "2026"},
		{time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC), "2026"},
		{time.Date(2027, 1, 10, 0, 0, 0, 0, time.UTC), "2027-sprint"},
		{time.Date(2027, 3, 15, 0, 0, 0, 0, time.UTC), "2027-sprint"},
		{time.Date(2027, 6, 20, 0, 0, 0, 0, time.UTC), "2027-28"},
		{time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC), "2027-28"},
		{time.Date(2028, 6, 20, 0, 0, 0, 0, time.UTC), "2028-29"},
	}
	for _, tt := range tests {
		s := seasons.SeasonForDate(tt.date)
		if s.ID != tt.wantID {
			t.Errorf("date %v: expected season %q, got %q", tt.date, tt.wantID, s.ID)
		}
	}
}

func TestSeasonForDate_BeforeAllSeasonsFallsBackToFirst(t *testing.T) {
	before2026 := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	s := seasons.SeasonForDate(before2026)
	if s.ID != "2026" {
		t.Errorf("expected fallback to first season 2026, got %q", s.ID)
	}
}

func TestMaybeCloseSeason(t *testing.T) {
	noExists := func(string) bool { return false }
	alreadyExists := func(string) bool { return true }

	// Day before Jan 10 boundary — no close
	jan9 := time.Date(2027, 1, 9, 4, 0, 0, 0, time.UTC)
	_, _, should := seasons.MaybeCloseSeason(jan9, "2026", noExists)
	if should {
		t.Error("should not close on Jan 9")
	}

	// Jan 10 — should close 2026, open 2027-sprint
	jan10 := time.Date(2027, 1, 10, 4, 0, 0, 0, time.UTC)
	closeID, openID, should := seasons.MaybeCloseSeason(jan10, "2026", noExists)
	if !should {
		t.Error("should close on Jan 10")
	}
	if closeID != "2026" {
		t.Errorf("expected closeID 2026, got %q", closeID)
	}
	if openID != "2027-sprint" {
		t.Errorf("expected openID 2027-sprint, got %q", openID)
	}

	// Already archived — idempotent
	_, _, should = seasons.MaybeCloseSeason(jan10, "2026", alreadyExists)
	if should {
		t.Error("should not close if season already archived")
	}

	// Jun 20 2027 — should close 2027-sprint, open 2027-28
	jun20 := time.Date(2027, 6, 20, 4, 0, 0, 0, time.UTC)
	closeID, openID, should = seasons.MaybeCloseSeason(jun20, "2027-sprint", noExists)
	if !should {
		t.Error("should close on Jun 20 2027")
	}
	if closeID != "2027-sprint" || openID != "2027-28" {
		t.Errorf("expected 2027-sprint→2027-28, got %q→%q", closeID, openID)
	}

	// Unknown season ID — should not close
	_, _, should = seasons.MaybeCloseSeason(jan10, "nonexistent-season", noExists)
	if should {
		t.Error("should not close for unknown season ID")
	}
}
