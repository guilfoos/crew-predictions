package seasons

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/guilfoos/crew-predictions/internal/repository"
)

// CloseSeason snapshots the current standings for seasonID, ranks entries by
// AcesRadioPoints descending, saves the snapshot, then resets all user scores.
func CloseSeason(ctx context.Context, seasonID string, users repository.UserStore, snaps repository.SeasonStore, now time.Time) error {
	def, ok := SeasonByID(seasonID)
	if !ok {
		return fmt.Errorf("close season: unknown season ID %q", seasonID)
	}

	allUsers, err := users.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("close season: get users: %w", err)
	}

	sort.Slice(allUsers, func(i, j int) bool {
		return allUsers[i].AcesRadioPoints > allUsers[j].AcesRadioPoints
	})

	entries := make([]repository.SeasonEntry, len(allUsers))
	for i, u := range allUsers {
		entries[i] = repository.SeasonEntry{
			UserID:          u.UserID,
			Handle:          u.Handle,
			AcesRadioPoints: u.AcesRadioPoints,
			Upper90Points:   u.Upper90Points,
			GrouchyPoints:   u.GrouchyPoints,
			PredictionCount: u.PredictionCount,
			Rank:            i + 1,
		}
	}

	snap := repository.SeasonSnapshot{
		ID:       seasonID,
		Name:     def.Name,
		ClosedAt: now,
		Entries:  entries,
	}
	if err := snaps.Save(ctx, snap); err != nil {
		return fmt.Errorf("close season: save snapshot: %w", err)
	}

	for _, u := range allUsers {
		// Keep predictionCount so users remain visible on the current leaderboard at 0 points.
		if err := users.UpdateScores(ctx, u.UserID, u.PredictionCount, 0, 0, 0); err != nil {
			return fmt.Errorf("close season: reset scores for %s: %w", u.UserID, err)
		}
	}

	return nil
}
