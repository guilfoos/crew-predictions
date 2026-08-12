package bot_test

import (
	"context"
	"testing"
	"time"

	"github.com/mcornell/crew-predictions/internal/bot"
	"github.com/mcornell/crew-predictions/internal/models"
	"github.com/mcornell/crew-predictions/internal/repository"
)

func upcoming(id, homeTeam, awayTeam string) models.Match {
	return models.Match{
		ID:       id,
		HomeTeam: homeTeam,
		AwayTeam: awayTeam,
		Status:   "STATUS_SCHEDULED",
		Kickoff:  time.Now().Add(24 * time.Hour),
	}
}

func past(id, homeTeam, awayTeam string) models.Match {
	return models.Match{
		ID:       id,
		HomeTeam: homeTeam,
		AwayTeam: awayTeam,
		Status:   "STATUS_SCHEDULED",
		Kickoff:  time.Now().Add(-1 * time.Hour),
	}
}

func TestTwoOneBot_PredictHomeMatch(t *testing.T) {
	ctx := context.Background()
	preds := repository.NewMemoryPredictionStore()
	users := repository.NewMemoryUserStore()
	b := bot.New(preds, users, "Columbus Crew")

	b.Predict(ctx, []models.Match{upcoming("m1", "Columbus Crew", "FC Dallas")})

	p, err := preds.GetByMatchAndUser(ctx, "m1", bot.UserID)
	if err != nil || p == nil {
		t.Fatal("expected prediction for home match, got nil")
	}
	if p.HomeGoals != 2 || p.AwayGoals != 1 {
		t.Errorf("expected 2-1, got %d-%d", p.HomeGoals, p.AwayGoals)
	}
}

func TestTwoOneBot_PredictAwayMatch(t *testing.T) {
	ctx := context.Background()
	preds := repository.NewMemoryPredictionStore()
	users := repository.NewMemoryUserStore()
	b := bot.New(preds, users, "Columbus Crew")

	b.Predict(ctx, []models.Match{upcoming("m2", "FC Dallas", "Columbus Crew")})

	p, _ := preds.GetByMatchAndUser(ctx, "m2", bot.UserID)
	if p == nil {
		t.Fatal("expected prediction for away match, got nil")
	}
	if p.HomeGoals != 1 || p.AwayGoals != 2 {
		t.Errorf("expected 1-2, got %d-%d", p.HomeGoals, p.AwayGoals)
	}
}

func TestTwoOneBot_SkipsMatchPastKickoff(t *testing.T) {
	ctx := context.Background()
	preds := repository.NewMemoryPredictionStore()
	users := repository.NewMemoryUserStore()
	b := bot.New(preds, users, "Columbus Crew")

	b.Predict(ctx, []models.Match{past("m3", "Columbus Crew", "FC Dallas")})

	p, _ := preds.GetByMatchAndUser(ctx, "m3", bot.UserID)
	if p != nil {
		t.Errorf("expected no prediction for past match, got %+v", p)
	}
}

func TestTwoOneBot_SkipsAlreadyPredicted(t *testing.T) {
	ctx := context.Background()
	preds := repository.NewMemoryPredictionStore()
	users := repository.NewMemoryUserStore()
	b := bot.New(preds, users, "Columbus Crew")

	// predict once
	b.Predict(ctx, []models.Match{upcoming("m4", "Columbus Crew", "FC Dallas")})
	// predict again — should not overwrite
	preds.Save(ctx, repository.Prediction{MatchID: "m4", UserID: bot.UserID, HomeGoals: 99, AwayGoals: 99})
	b.Predict(ctx, []models.Match{upcoming("m4", "Columbus Crew", "FC Dallas")})

	all, _ := preds.GetByMatch(ctx, "m4")
	if len(all) != 1 {
		t.Errorf("expected 1 prediction, got %d", len(all))
	}
}

func TestTwoOneBot_SkipsNonScheduledMatch(t *testing.T) {
	ctx := context.Background()
	preds := repository.NewMemoryPredictionStore()
	users := repository.NewMemoryUserStore()
	b := bot.New(preds, users, "Columbus Crew")

	m := upcoming("m5", "Columbus Crew", "FC Dallas")
	m.Status = "STATUS_FULL_TIME"
	b.Predict(ctx, []models.Match{m})

	p, _ := preds.GetByMatchAndUser(ctx, "m5", bot.UserID)
	if p != nil {
		t.Errorf("expected no prediction for completed match, got %+v", p)
	}
}

func TestTwoOneBot_SkipsMatchOnGetByMatchAndUserError(t *testing.T) {
	ctx := context.Background()
	preds := repository.NewGetByMatchAndUserErrorPredictionStore()
	b := bot.New(preds, repository.NewMemoryUserStore(), "Columbus Crew")

	b.Predict(ctx, []models.Match{upcoming("m-err", "Columbus Crew", "FC Dallas")})

	// GetByMatchAndUser errored, so no prediction should have been saved
	all, _ := preds.GetByMatch(ctx, "m-err")
	if len(all) != 0 {
		t.Errorf("expected no prediction saved when GetByMatchAndUser fails, got %d", len(all))
	}
}

func TestTwoOneBot_LogsErrorWhenSaveFails(t *testing.T) {
	// ErrorPredictionStore returns nil from GetByMatchAndUser (no existing
	// prediction, so the bot tries to save) and errors on Save. The bot must
	// log and proceed without panicking.
	ctx := context.Background()
	preds := repository.NewErrorPredictionStore()
	b := bot.New(preds, repository.NewMemoryUserStore(), "Columbus Crew")

	b.Predict(ctx, []models.Match{upcoming("m-save-err", "Columbus Crew", "FC Dallas")})
}

func TestTwoOneBot_LogsErrorOnUpsertFailure(t *testing.T) {
	ctx := context.Background()
	preds := repository.NewMemoryPredictionStore()
	b := bot.New(preds, repository.NewErrorUpsertUserStore(), "Columbus Crew")

	// Should not panic — error is logged
	b.Predict(ctx, nil)
}

func TestTwoOneBot_RegistersUserInUserStore(t *testing.T) {
	ctx := context.Background()
	preds := repository.NewMemoryPredictionStore()
	users := repository.NewMemoryUserStore()
	b := bot.New(preds, users, "Columbus Crew")

	b.Predict(ctx, nil)

	u, err := users.GetByID(ctx, bot.UserID)
	if err != nil || u == nil {
		t.Fatal("expected bot user in UserStore, got nil")
	}
	if u.Handle != "Upper 90 Club's TwoOneBot" {
		t.Errorf("expected handle %q, got %q", "Upper 90 Club's TwoOneBot", u.Handle)
	}
	if u.Location != "From the Upper 90 Club" {
		t.Errorf("expected location 'From the Upper 90 Club', got %q", u.Location)
	}
}
