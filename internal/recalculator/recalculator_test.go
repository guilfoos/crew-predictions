package recalculator_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/guilfoos/crew-predictions/internal/recalculator"
	"github.com/guilfoos/crew-predictions/internal/repository"
)

type errorGetResultStore struct{}

func (e *errorGetResultStore) SaveResult(_ context.Context, _ repository.Result) error { return nil }
func (e *errorGetResultStore) GetResult(_ context.Context, _ string) (*repository.Result, error) {
	return nil, fmt.Errorf("simulated GetResult failure")
}

func TestRecalculate_SetsAllScoringSystemPoints(t *testing.T) {
	ctx := context.Background()
	users := repository.NewMemoryUserStore()
	preds := repository.NewMemoryPredictionStore()
	results := repository.NewMemoryResultStore()

	users.Upsert(ctx, repository.User{UserID: "u1", Handle: "CrewFan"})
	// Columbus home, exact 2-1 → AcesRadio=15, Upper90=3 (correct outcome + both scores), Grouchy=1 (win by 1)
	preds.Save(ctx, repository.Prediction{MatchID: "m1", UserID: "u1", HomeGoals: 2, AwayGoals: 1})
	results.SaveResult(ctx, repository.Result{MatchID: "m1", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas", HomeGoals: 2, AwayGoals: 1})

	recalculator.Recalculate(ctx, preds, results, users, nil, recalculator.SeasonWindow{}, "Columbus Crew")

	u, _ := users.GetByID(ctx, "u1")
	if u.AcesRadioPoints != 15 {
		t.Errorf("expected 15 AcesRadio points, got %d", u.AcesRadioPoints)
	}
	if u.Upper90Points != 3 {
		t.Errorf("expected 3 Upper90 points, got %d", u.Upper90Points)
	}
	if u.GrouchyPoints != 1 {
		t.Errorf("expected 1 Grouchy point, got %d", u.GrouchyPoints)
	}
}

func TestRecalculate_ZerosPointsForUserWithNoPredictions(t *testing.T) {
	ctx := context.Background()
	users := repository.NewMemoryUserStore()
	// User exists in store but has made no predictions
	users.Upsert(ctx, repository.User{UserID: "u1", Handle: "Lurker", AcesRadioPoints: 99})

	recalculator.Recalculate(ctx, repository.NewMemoryPredictionStore(), repository.NewMemoryResultStore(), users, nil, recalculator.SeasonWindow{}, "Columbus Crew")

	u, _ := users.GetByID(ctx, "u1")
	if u.AcesRadioPoints != 0 {
		t.Errorf("expected 0 points after recalculate with no predictions, got %d", u.AcesRadioPoints)
	}
}

func TestRecalculate_SetsPredictionCount(t *testing.T) {
	ctx := context.Background()
	users := repository.NewMemoryUserStore()
	preds := repository.NewMemoryPredictionStore()

	users.Upsert(ctx, repository.User{UserID: "u1", Handle: "CrewFan"})
	preds.Save(ctx, repository.Prediction{MatchID: "m1", UserID: "u1", HomeGoals: 2, AwayGoals: 1})
	preds.Save(ctx, repository.Prediction{MatchID: "m2", UserID: "u1", HomeGoals: 1, AwayGoals: 0})

	recalculator.Recalculate(ctx, preds, repository.NewMemoryResultStore(), users, nil, recalculator.SeasonWindow{}, "Columbus Crew")

	u, _ := users.GetByID(ctx, "u1")
	if u.PredictionCount != 2 {
		t.Errorf("expected PredictionCount 2, got %d", u.PredictionCount)
	}
}

// countingResultStore wraps MemoryResultStore and counts GetResult calls so
// we can verify the result-cache deduplication actually saves lookups.
type countingResultStore struct {
	*repository.MemoryResultStore
	calls int
}

func (c *countingResultStore) GetResult(ctx context.Context, matchID string) (*repository.Result, error) {
	c.calls++
	return c.MemoryResultStore.GetResult(ctx, matchID)
}

func TestRecalculate_DeduplicatesResultLookupsAcrossUsers(t *testing.T) {
	// Two users, both with predictions for the SAME match. The result cache
	// in Recalculate should fetch the result exactly once (not once per user).
	ctx := context.Background()
	users := repository.NewMemoryUserStore()
	preds := repository.NewMemoryPredictionStore()
	results := &countingResultStore{MemoryResultStore: repository.NewMemoryResultStore()}

	users.Upsert(ctx, repository.User{UserID: "u1", Handle: "FanA"})
	users.Upsert(ctx, repository.User{UserID: "u2", Handle: "FanB"})
	preds.Save(ctx, repository.Prediction{MatchID: "m1", UserID: "u1", HomeGoals: 2, AwayGoals: 1})
	preds.Save(ctx, repository.Prediction{MatchID: "m1", UserID: "u2", HomeGoals: 0, AwayGoals: 0})
	results.SaveResult(ctx, repository.Result{MatchID: "m1", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas", HomeGoals: 2, AwayGoals: 1})

	// Reset call counter — SaveResult above also touches the store
	// indirectly but doesn't use GetResult.
	results.calls = 0

	if err := recalculator.Recalculate(ctx, preds, results, users, nil, recalculator.SeasonWindow{}, "Columbus Crew"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results.calls != 1 {
		t.Errorf("expected GetResult called exactly once via the cache, was called %d times", results.calls)
	}

	// Sanity check both users got scored from the same result.
	u1, _ := users.GetByID(ctx, "u1")
	u2, _ := users.GetByID(ctx, "u2")
	if u1.AcesRadioPoints != 15 {
		t.Errorf("u1 (exact 2-1) expected 15 AcesRadio points, got %d", u1.AcesRadioPoints)
	}
	if u2.AcesRadioPoints != 0 {
		t.Errorf("u2 (wrong 0-0) expected 0 AcesRadio points, got %d", u2.AcesRadioPoints)
	}
}

func TestRecalculate_ReturnsErrorWhenPredictionStoreFails(t *testing.T) {
	ctx := context.Background()
	err := recalculator.Recalculate(ctx, repository.NewErrorGetAllPredictionStore(), repository.NewMemoryResultStore(), repository.NewMemoryUserStore(), nil, recalculator.SeasonWindow{}, "Columbus Crew")
	if err == nil {
		t.Error("expected error when prediction store fails, got nil")
	}
}

func TestRecalculate_ReturnsErrorWhenUserStoreFails(t *testing.T) {
	ctx := context.Background()
	err := recalculator.Recalculate(ctx, repository.NewMemoryPredictionStore(), repository.NewMemoryResultStore(), repository.NewErrorGetAllUserStore(), nil, recalculator.SeasonWindow{}, "Columbus Crew")
	if err == nil {
		t.Error("expected error when user store GetAll fails, got nil")
	}
}

func TestRecalculate_ReturnsErrorWhenUpdateScoresFails(t *testing.T) {
	ctx := context.Background()
	// ErrorUpsertWithUserStore returns one user from GetAll but fails on UpdateScores.
	err := recalculator.Recalculate(ctx, repository.NewMemoryPredictionStore(), repository.NewMemoryResultStore(), repository.NewErrorUpsertWithUserStore(), nil, recalculator.SeasonWindow{}, "Columbus Crew")
	if err == nil {
		t.Error("expected error when UpdateScores fails, got nil")
	}
}

func TestRecalculate_CreatesUserStoreEntryForPredictionOnlyUser(t *testing.T) {
	ctx := context.Background()
	users := repository.NewMemoryUserStore()
	preds := repository.NewMemoryPredictionStore()
	results := repository.NewMemoryResultStore()

	// User has a prediction but was never upserted to UserStore (e.g. seeded via admin endpoint)
	preds.Save(ctx, repository.Prediction{MatchID: "m1", UserID: "google:BlackAndGold", HomeGoals: 2, AwayGoals: 1})
	results.SaveResult(ctx, repository.Result{MatchID: "m1", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas", HomeGoals: 2, AwayGoals: 1})

	if err := recalculator.Recalculate(ctx, preds, results, users, nil, recalculator.SeasonWindow{}, "Columbus Crew"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	u, err := users.GetByID(ctx, "google:BlackAndGold")
	if err != nil {
		t.Fatalf("expected user to be created in UserStore, got error: %v", err)
	}
	if u.PredictionCount != 1 {
		t.Errorf("expected PredictionCount 1, got %d", u.PredictionCount)
	}
	if u.AcesRadioPoints != 15 {
		t.Errorf("expected 15 AcesRadio points, got %d", u.AcesRadioPoints)
	}
}

func TestRecalculate_ReturnsErrorWhenGetResultFails(t *testing.T) {
	ctx := context.Background()
	users := repository.NewMemoryUserStore()
	users.Upsert(ctx, repository.User{UserID: "u1"})
	preds := repository.NewMemoryPredictionStore()
	preds.Save(ctx, repository.Prediction{MatchID: "m1", UserID: "u1", HomeGoals: 1, AwayGoals: 0})

	err := recalculator.Recalculate(ctx, preds, &errorGetResultStore{}, users, nil, recalculator.SeasonWindow{}, "Columbus Crew")
	if err == nil {
		t.Error("expected error when result store GetResult fails, got nil")
	}
}

func TestRecalculate_ReturnsErrorWhenPhantomUserUpsertFails(t *testing.T) {
	ctx := context.Background()
	preds := repository.NewMemoryPredictionStore()
	preds.Save(ctx, repository.Prediction{MatchID: "m1", UserID: "google:ghost", HomeGoals: 1, AwayGoals: 0})

	// ErrorUpsertUserStore: GetAll returns empty (no error) → phantom user detected; Upsert returns error
	err := recalculator.Recalculate(ctx, preds, repository.NewMemoryResultStore(), repository.NewErrorUpsertUserStore(), nil, recalculator.SeasonWindow{}, "Columbus Crew")
	if err == nil {
		t.Error("expected error when phantom user upsert fails, got nil")
	}
}

func TestRecalculate_SeasonWindow_ExcludesPredictionsOutsideWindow(t *testing.T) {
	ctx := context.Background()
	users := repository.NewMemoryUserStore()
	preds := repository.NewMemoryPredictionStore()
	results := repository.NewMemoryResultStore()

	users.Upsert(ctx, repository.User{UserID: "u1", Handle: "CrewFan"})
	preds.Save(ctx, repository.Prediction{MatchID: "m1", UserID: "u1", HomeGoals: 2, AwayGoals: 1})
	results.SaveResult(ctx, repository.Result{MatchID: "m1", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas", HomeGoals: 2, AwayGoals: 1})

	// Match kickoff is in 2026; window is 2027 Sprint — should score 0.
	kickoff2026 := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	kickoffFor := func(matchID string) (time.Time, bool) {
		if matchID == "m1" {
			return kickoff2026, true
		}
		return time.Time{}, false
	}
	window := recalculator.SeasonWindow{
		Start: time.Date(2027, 1, 10, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2027, 6, 20, 0, 0, 0, 0, time.UTC),
	}

	if err := recalculator.Recalculate(ctx, preds, results, users, kickoffFor, window, "Columbus Crew"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	u, _ := users.GetByID(ctx, "u1")
	if u.AcesRadioPoints != 0 {
		t.Errorf("expected 0 points for out-of-window prediction, got %d", u.AcesRadioPoints)
	}
}

func TestRecalculate_SeasonWindow_IncludesPredictionsInsideWindow(t *testing.T) {
	ctx := context.Background()
	users := repository.NewMemoryUserStore()
	preds := repository.NewMemoryPredictionStore()
	results := repository.NewMemoryResultStore()

	users.Upsert(ctx, repository.User{UserID: "u1", Handle: "CrewFan"})
	preds.Save(ctx, repository.Prediction{MatchID: "m1", UserID: "u1", HomeGoals: 2, AwayGoals: 1})
	results.SaveResult(ctx, repository.Result{MatchID: "m1", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas", HomeGoals: 2, AwayGoals: 1})

	kickoff2026 := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	kickoffFor := func(matchID string) (time.Time, bool) { return kickoff2026, true }
	window := recalculator.SeasonWindow{
		Start: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2027, 1, 10, 0, 0, 0, 0, time.UTC),
	}

	recalculator.Recalculate(ctx, preds, results, users, kickoffFor, window, "Columbus Crew")

	u, _ := users.GetByID(ctx, "u1")
	if u.AcesRadioPoints != 15 {
		t.Errorf("expected 15 points for in-window prediction, got %d", u.AcesRadioPoints)
	}
}

func TestRecalculate_NilKickoffFor_ScoresAllPredictions(t *testing.T) {
	ctx := context.Background()
	users := repository.NewMemoryUserStore()
	preds := repository.NewMemoryPredictionStore()
	results := repository.NewMemoryResultStore()

	users.Upsert(ctx, repository.User{UserID: "u1", Handle: "CrewFan"})
	preds.Save(ctx, repository.Prediction{MatchID: "m1", UserID: "u1", HomeGoals: 2, AwayGoals: 1})
	results.SaveResult(ctx, repository.Result{MatchID: "m1", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas", HomeGoals: 2, AwayGoals: 1})

	// nil kickoffFor and zero window = no filtering
	recalculator.Recalculate(ctx, preds, results, users, nil, recalculator.SeasonWindow{}, "Columbus Crew")

	u, _ := users.GetByID(ctx, "u1")
	if u.AcesRadioPoints != 15 {
		t.Errorf("expected 15 points with nil kickoffFor, got %d", u.AcesRadioPoints)
	}
}

func TestRecalculate_SetsAcesRadioPoints(t *testing.T) {
	ctx := context.Background()
	users := repository.NewMemoryUserStore()
	preds := repository.NewMemoryPredictionStore()
	results := repository.NewMemoryResultStore()

	users.Upsert(ctx, repository.User{UserID: "u1", Handle: "CrewFan"})
	preds.Save(ctx, repository.Prediction{MatchID: "m1", UserID: "u1", HomeGoals: 2, AwayGoals: 1})
	results.SaveResult(ctx, repository.Result{MatchID: "m1", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas", HomeGoals: 2, AwayGoals: 1})

	if err := recalculator.Recalculate(ctx, preds, results, users, nil, recalculator.SeasonWindow{}, "Columbus Crew"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	u, _ := users.GetByID(ctx, "u1")
	if u.AcesRadioPoints != 15 {
		t.Errorf("expected 15 AcesRadio points (exact score), got %d", u.AcesRadioPoints)
	}
}
