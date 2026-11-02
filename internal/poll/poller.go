// Package poll runs the live-match polling loop that fetches updated scores
// and events from ESPN on a fixed cadence and writes finished results to the
// result store.
package poll

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/guilfoos/crew-predictions/internal/models"
	"github.com/guilfoos/crew-predictions/internal/repository"
)

var terminalStatuses = map[string]bool{
	"STATUS_FULL_TIME": true,
	"STATUS_FINAL_AET": true,
	"STATUS_FINAL_PEN": true,
}

// MergeChainFields copies LastPollAt / ChainSeededFor / AbandonedAt from the
// existing matches in `store` onto matches in `fresh`, matched by ID. Called
// before SaveAll on any path that overwrites the match store with fresh ESPN
// data (which doesn't carry those fields). Without this merge, every fetch
// would silently wipe the chain-tracking state — refresh would think every
// in-progress chain is dead and seed duplicate revival tasks, and the API
// `lastPollAt` field would always come back zero.
//
// On read error: returns `fresh` unchanged and logs. Worst case is a one-tick
// regression on those fields; the next merge cycle restores them.
func MergeChainFields(store repository.MatchStore, fresh []models.Match) []models.Match {
	existing, err := store.GetAll()
	if err != nil {
		slog.Error("MergeChainFields: existing store read failed; chain fields may briefly reset", "error", err)
		return fresh
	}
	byID := make(map[string]models.Match, len(existing))
	for _, m := range existing {
		byID[m.ID] = m
	}
	for i, m := range fresh {
		if prev, ok := byID[m.ID]; ok {
			fresh[i].LastPollAt = prev.LastPollAt
			fresh[i].ChainSeededFor = prev.ChainSeededFor
			fresh[i].AbandonedAt = prev.AbandonedAt
		}
	}
	return fresh
}

func PollOnce(ctx context.Context, matchStore repository.MatchStore, resultStore repository.ResultStore, fetcher func() ([]models.Match, error)) error {
	matches, err := fetcher()
	if err != nil {
		return err
	}

	matches = MergeChainFields(matchStore, matches)
	now := time.Now().UTC()
	for i := range matches {
		matches[i].LastPollAt = now
	}

	if err := matchStore.SaveAll(matches); err != nil {
		return err
	}

	for _, m := range matches {
		if !terminalStatuses[m.Status] {
			continue
		}
		if err := saveResult(ctx, resultStore, m); err != nil {
			return err
		}
	}
	return nil
}

func saveResult(ctx context.Context, resultStore repository.ResultStore, m models.Match) error {
	home, err := strconv.Atoi(m.HomeScore)
	if err != nil {
		return nil
	}
	away, err := strconv.Atoi(m.AwayScore)
	if err != nil {
		return nil
	}
	return resultStore.SaveResult(ctx, repository.Result{
		MatchID:   m.ID,
		HomeTeam:  m.HomeTeam,
		AwayTeam:  m.AwayTeam,
		HomeGoals: home,
		AwayGoals: away,
	})
}
