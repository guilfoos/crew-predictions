package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/guilfoos/crew-predictions/internal/bot"
	"github.com/guilfoos/crew-predictions/internal/models"
	"github.com/guilfoos/crew-predictions/internal/poll"
	"github.com/guilfoos/crew-predictions/internal/repository"
)

// next4amET returns the next 4:00am ET instant strictly after now. It's used
// to schedule the daily ESPN refresh: we want exactly one refresh per day,
// happening overnight when traffic is lowest.
func next4amET(now time.Time, loc *time.Location) time.Time {
	t := now.In(loc)
	candidate := time.Date(t.Year(), t.Month(), t.Day(), 4, 0, 0, 0, loc)
	if !candidate.After(now) {
		candidate = candidate.Add(24 * time.Hour)
	}
	return candidate
}

// startDailyRefresh kicks off an immediate refresh and then loops until the
// returned channel is closed, refreshing again at each subsequent 4am ET.
// A refresh: pulls matches from ESPN, persists them, backfills any results
// for matches that finished while no poller was watching, recomputes the
// leaderboard, asks the 2-1 bot to fill in upcoming predictions, and resets
// the live poller so it watches the new schedule.
func startDailyRefresh(store repository.MatchStore, fetcher func() ([]models.Match, error), poller *poll.MatchPoller, twoOneBot *bot.TwoOneBot, recalcFn func(context.Context), etLoc *time.Location) chan struct{} {
	stop := make(chan struct{})
	go func() {
		refresh := func() {
			matches, err := fetcher()
			if err != nil {
				slog.Error("daily match refresh: ESPN fetch failed", "error", err)
				return
			}
			if err := store.SaveAll(matches); err != nil {
				slog.Error("daily match refresh: store save failed", "error", err)
				return
			}
			poller.Backfill(context.Background(), matches)
			recalcFn(context.Background())
			twoOneBot.Predict(context.Background(), matches)
			slog.Info("daily match refresh complete", "matchCount", len(matches))
			poller.Reset(matches)
		}
		refresh()
		for {
			delay := time.Until(next4amET(time.Now(), etLoc))
			timer := time.NewTimer(delay)
			select {
			case <-timer.C:
				refresh()
			case <-stop:
				timer.Stop()
				return
			}
		}
	}()
	return stop
}
