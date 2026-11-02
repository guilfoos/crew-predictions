// Package bot provides the TwoOneBot automated predictor that submits a
// fixed 2-1 prediction (home or away depending on the venue) for every
// upcoming Crew match before kickoff.
package bot

import (
	"context"
	"log/slog"
	"time"

	"github.com/guilfoos/crew-predictions/internal/models"
	"github.com/guilfoos/crew-predictions/internal/repository"
)

const UserID = "bot:twooonebot"

type TwoOneBot struct {
	predictions repository.PredictionStore
	users       repository.UserStore
	targetTeam  string
}

func New(predictions repository.PredictionStore, users repository.UserStore, targetTeam string) *TwoOneBot {
	return &TwoOneBot{predictions: predictions, users: users, targetTeam: targetTeam}
}

func (b *TwoOneBot) Predict(ctx context.Context, matches []models.Match) {
	if err := b.users.Upsert(ctx, repository.User{
		UserID:   UserID,
		Handle:   "Upper 90 Club's TwoOneBot",
		Location: "From the Upper 90 Club",
		Provider: "bot",
	}); err != nil {
		slog.Error("twoonebot: failed to register user", "error", err)
	}

	for _, m := range matches {
		if m.Status != "STATUS_SCHEDULED" {
			continue
		}
		if !m.Kickoff.After(time.Now()) {
			continue
		}
		existing, err := b.predictions.GetByMatchAndUser(ctx, m.ID, UserID)
		if err != nil {
			slog.Error("twoonebot: failed to check existing prediction", "matchID", m.ID, "error", err)
			continue
		}
		if existing != nil {
			continue
		}
		homeGoals, awayGoals := 2, 1
		if m.AwayTeam == b.targetTeam {
			homeGoals, awayGoals = 1, 2
		}
		if err := b.predictions.Save(ctx, repository.Prediction{
			MatchID:   m.ID,
			UserID:    UserID,
			HomeGoals: homeGoals,
			AwayGoals: awayGoals,
		}); err != nil {
			slog.Error("twoonebot: failed to save prediction", "matchID", m.ID, "error", err)
		}
	}
}
