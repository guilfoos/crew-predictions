// Package recalculator rolls every prediction against its match's final
// result and writes the per-user score totals back onto the user records.
package recalculator

import (
	"context"
	"fmt"
	"time"

	"github.com/guilfoos/crew-predictions/internal/repository"
	"github.com/guilfoos/crew-predictions/internal/scoring"
)

type SeasonWindow struct {
	Start time.Time
	End   time.Time
}

func (w SeasonWindow) IsZero() bool {
	return w.Start.IsZero() && w.End.IsZero()
}

func (w SeasonWindow) Contains(t time.Time) bool {
	return !t.Before(w.Start) && t.Before(w.End)
}

func Recalculate(ctx context.Context, predictions repository.PredictionStore, results repository.ResultStore, users repository.UserStore, kickoffFor func(matchID string) (time.Time, bool), window SeasonWindow, targetTeam string) error {
	allPredictions, err := predictions.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("recalculate: get predictions: %w", err)
	}

	allUsers, err := users.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("recalculate: get users: %w", err)
	}

	// Index predictions by userID.
	predsByUser := make(map[string][]repository.Prediction, len(allUsers))
	for _, p := range allPredictions {
		predsByUser[p.UserID] = append(predsByUser[p.UserID], p)
	}

	// Cache results by matchID to avoid redundant lookups.
	resultCache := map[string]*repository.Result{}
	for _, p := range allPredictions {
		if _, seen := resultCache[p.MatchID]; seen {
			continue
		}
		r, err := results.GetResult(ctx, p.MatchID)
		if err != nil {
			return fmt.Errorf("recalculate: get result %s: %w", p.MatchID, err)
		}
		resultCache[p.MatchID] = r
	}

	// Build a set of known userIDs for fast lookup.
	knownUsers := make(map[string]repository.User, len(allUsers))
	for _, u := range allUsers {
		knownUsers[u.UserID] = u
	}

	// Also include users who have predictions but no UserStore entry yet.
	// Upsert their profile so handle is available for leaderboard/match detail display.
	for userID := range predsByUser {
		if _, exists := knownUsers[userID]; !exists {
			u := repository.User{UserID: userID}
			knownUsers[userID] = u
			if err := users.Upsert(ctx, u); err != nil {
				return fmt.Errorf("recalculate: upsert new user %s: %w", userID, err)
			}
		}
	}

	for _, u := range knownUsers {
		var acesTotal, u90Total, grouchyTotal, predCount int
		for _, p := range predsByUser[u.UserID] {
			if kickoffFor != nil && !window.IsZero() {
				kickoff, ok := kickoffFor(p.MatchID)
				if !ok || !window.Contains(kickoff) {
					continue
				}
			}
			predCount++
			r := resultCache[p.MatchID]
			if r == nil {
				continue
			}
			pred := scoring.Prediction{Home: p.HomeGoals, Away: p.AwayGoals}
			res := scoring.Result{Home: r.HomeGoals, Away: r.AwayGoals}
			targetIsHome := r.HomeTeam == targetTeam
			acesTotal += scoring.AcesRadio(res, pred)
			u90Total += scoring.Upper90Club(res, pred, targetIsHome)
			grouchyTotal += scoring.Grouchy(res, pred, targetIsHome)
		}
		if err := users.UpdateScores(ctx, u.UserID, predCount, acesTotal, u90Total, grouchyTotal); err != nil {
			return fmt.Errorf("recalculate: update scores %s: %w", u.UserID, err)
		}
	}

	return nil
}
