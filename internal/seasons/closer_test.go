package seasons_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/guilfoos/crew-predictions/internal/repository"
	"github.com/guilfoos/crew-predictions/internal/seasons"
)

func TestCloseSeason_SavesSnapshotWithRankedEntries(t *testing.T) {
	ctx := context.Background()
	users := repository.NewMemoryUserStore()
	snaps := repository.NewMemorySeasonStore()

	users.Upsert(ctx, repository.User{UserID: "u1", Handle: "TopFan"})
	users.Upsert(ctx, repository.User{UserID: "u2", Handle: "SecondFan"})
	users.UpdateScores(ctx, "u1", 1, 15, 0, 0)
	users.UpdateScores(ctx, "u2", 1, 10, 0, 0)

	now := time.Date(2027, 2, 1, 0, 0, 0, 0, time.UTC)
	if err := seasons.CloseSeason(ctx, "2026", users, snaps, now); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	snap, err := snaps.GetByID(ctx, "2026")
	if err != nil || snap == nil {
		t.Fatalf("expected snapshot, got err=%v snap=%v", err, snap)
	}
	if snap.Name != "2026 Season" {
		t.Errorf("expected name '2026 Season', got %q", snap.Name)
	}
	if len(snap.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(snap.Entries))
	}
	if snap.Entries[0].Handle != "TopFan" {
		t.Errorf("expected first entry TopFan, got %q", snap.Entries[0].Handle)
	}
	if snap.Entries[0].Rank != 1 {
		t.Errorf("expected rank 1, got %d", snap.Entries[0].Rank)
	}
	if snap.Entries[1].Rank != 2 {
		t.Errorf("expected rank 2, got %d", snap.Entries[1].Rank)
	}
}

func TestCloseSeason_ResetsUserScores(t *testing.T) {
	ctx := context.Background()
	users := repository.NewMemoryUserStore()
	snaps := repository.NewMemorySeasonStore()

	users.Upsert(ctx, repository.User{UserID: "u1", Handle: "Fan"})
	users.UpdateScores(ctx, "u1", 1, 15, 0, 0)

	now := time.Date(2027, 2, 1, 0, 0, 0, 0, time.UTC)
	if err := seasons.CloseSeason(ctx, "2026", users, snaps, now); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	u, err := users.GetByID(ctx, "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.AcesRadioPoints != 0 {
		t.Errorf("expected 0 AcesRadioPoints after close, got %d", u.AcesRadioPoints)
	}
	if u.PredictionCount != 1 {
		t.Errorf("expected PredictionCount preserved at 1, got %d", u.PredictionCount)
	}
}

func TestCloseSeason_ReturnsErrorForUnknownSeasonID(t *testing.T) {
	ctx := context.Background()
	err := seasons.CloseSeason(ctx, "9999", repository.NewMemoryUserStore(), repository.NewMemorySeasonStore(), time.Now())
	if err == nil {
		t.Error("expected error for unknown season ID, got nil")
	}
}

func TestCloseSeason_ReturnsErrorWhenGetAllFails(t *testing.T) {
	err := seasons.CloseSeason(context.Background(), "2026", repository.NewErrorGetAllUserStore(), repository.NewMemorySeasonStore(), time.Now())
	if err == nil {
		t.Error("expected error when GetAll fails, got nil")
	}
}

func TestCloseSeason_ReturnsErrorWhenSaveFails(t *testing.T) {
	ctx := context.Background()
	users := repository.NewMemoryUserStore()
	users.Upsert(ctx, repository.User{UserID: "u1", Handle: "Fan"})
	users.UpdateScores(ctx, "u1", 1, 10, 0, 0)

	err := seasons.CloseSeason(ctx, "2026", users, repository.NewErrorSeasonStore(), time.Now())
	if err == nil {
		t.Error("expected error when season Save fails, got nil")
	}
}

func TestCloseSeason_ReturnsErrorWhenUpdateScoresFails(t *testing.T) {
	// ErrorUpsertWithUserStore: GetAll returns one user (so we reach the
	// reset loop), UpdateScores fails. The Upsert error is irrelevant here
	// because CloseSeason never calls Upsert.
	err := seasons.CloseSeason(
		context.Background(),
		"2026",
		repository.NewErrorUpsertWithUserStore(),
		repository.NewMemorySeasonStore(),
		time.Now(),
	)
	if err == nil {
		t.Fatal("expected error when UpdateScores fails, got nil")
	}
	if !strings.Contains(err.Error(), "reset scores") {
		t.Errorf("expected error to mention 'reset scores', got %q", err.Error())
	}
}

func TestCloseSeason_SetsClosedAt(t *testing.T) {
	ctx := context.Background()
	users := repository.NewMemoryUserStore()
	snaps := repository.NewMemorySeasonStore()

	users.Upsert(ctx, repository.User{UserID: "u1", Handle: "Fan"})
	users.UpdateScores(ctx, "u1", 0, 5, 0, 0)

	now := time.Date(2027, 3, 15, 12, 0, 0, 0, time.UTC)
	seasons.CloseSeason(ctx, "2026", users, snaps, now)

	snap, _ := snaps.GetByID(ctx, "2026")
	if snap == nil {
		t.Fatal("expected snapshot")
	}
	if !snap.ClosedAt.Equal(now) {
		t.Errorf("expected ClosedAt %v, got %v", now, snap.ClosedAt)
	}
}
