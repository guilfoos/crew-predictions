package scoring_test

import (
	"testing"

	"github.com/guilfoos/crew-predictions/internal/scoring"
)

// Upper 90 Club scoring:
//
//	+3 exact score
//	+2 correct outcome AND (correct Columbus goals OR correct opponent goals)
//	+1 correct outcome only OR correct Columbus goals only
//	 0 otherwise
func TestUpper90Club(t *testing.T) {
	cases := []struct {
		desc     string
		result   scoring.Result
		pred     scoring.Prediction
		crewHome bool
		want     int
	}{
		// Columbus is away (crew goals = away score)
		{"away: exact score → +3",
			scoring.Result{Home: 2, Away: 0}, scoring.Prediction{Home: 2, Away: 0}, false, 3},
		{"away: correct outcome + correct opponent goals (Portland 3-2 vs pick 3-0) → +2",
			scoring.Result{Home: 3, Away: 2}, scoring.Prediction{Home: 3, Away: 0}, false, 2},
		{"away: correct winner only, wrong Crew goals → +1",
			scoring.Result{Home: 2, Away: 0}, scoring.Prediction{Home: 3, Away: 1}, false, 1},
		{"away: wrong winner but correct Crew goals → +1",
			scoring.Result{Home: 2, Away: 1}, scoring.Prediction{Home: 0, Away: 1}, false, 1},
		{"away: correct winner + correct Crew goals (4≠3 not exact) → +2",
			scoring.Result{Home: 3, Away: 2}, scoring.Prediction{Home: 4, Away: 2}, false, 2},
		{"away: wrong winner + wrong Crew goals → 0",
			scoring.Result{Home: 2, Away: 0}, scoring.Prediction{Home: 0, Away: 1}, false, 0},

		// User-provided Portland 3-2 Crew matrix (Columbus is away)
		{"away: Portland 3-2 picked Portland 1-0 (correct outcome only)",
			scoring.Result{Home: 3, Away: 2}, scoring.Prediction{Home: 1, Away: 0}, false, 1},
		{"away: Portland 3-2 picked Portland 1-2 (correct Crew goals only)",
			scoring.Result{Home: 3, Away: 2}, scoring.Prediction{Home: 1, Away: 2}, false, 1},
		{"away: Portland 3-2 picked Portland 4-2 (correct outcome + Crew goals)",
			scoring.Result{Home: 3, Away: 2}, scoring.Prediction{Home: 4, Away: 2}, false, 2},
		{"away: Portland 3-2 picked Portland 3-2 (exact)",
			scoring.Result{Home: 3, Away: 2}, scoring.Prediction{Home: 3, Away: 2}, false, 3},

		// Columbus is home (crew goals = home score)
		{"home: wrong winner but correct Crew goals → +1",
			scoring.Result{Home: 2, Away: 3}, scoring.Prediction{Home: 2, Away: 0}, true, 1},
		{"home: exact score → +3",
			scoring.Result{Home: 2, Away: 1}, scoring.Prediction{Home: 2, Away: 1}, true, 3},
		{"home: correct outcome + correct opponent goals (3-1 vs pick 4-1) → +2",
			scoring.Result{Home: 3, Away: 1}, scoring.Prediction{Home: 4, Away: 1}, true, 2},
		{"home: correct winner only → +1",
			scoring.Result{Home: 3, Away: 1}, scoring.Prediction{Home: 2, Away: 0}, true, 1},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			if got := scoring.Upper90Club(c.result, c.pred, c.crewHome); got != c.want {
				t.Errorf("Upper90Club(%v, %v, crewHome=%v) = %d, want %d",
					c.result, c.pred, c.crewHome, got, c.want)
			}
		})
	}
}
