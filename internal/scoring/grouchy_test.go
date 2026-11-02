package scoring_test

import (
	"testing"

	"github.com/guilfoos/crew-predictions/internal/scoring"
)

// Grouchy awards 1 point when the predicted Columbus margin lands in the
// same outcome bucket as the actual margin (Win by 2+ / Win by 1 / Draw /
// Lose by 1 / Lose by 2+); otherwise 0.
func TestGrouchy(t *testing.T) {
	cases := []struct {
		desc     string
		result   scoring.Result
		pred     scoring.Prediction
		crewHome bool
		want     int
	}{
		// Columbus is home in these cases
		{"home: exact win by 2+", scoring.Result{Home: 2, Away: 0}, scoring.Prediction{Home: 3, Away: 0}, true, 1},
		{"home: predicted win by 1, actual win by 2 (different category)",
			scoring.Result{Home: 2, Away: 0}, scoring.Prediction{Home: 1, Away: 0}, true, 0},
		{"home: actual win by 3 still in win-by-2+ bucket as predicted win by 4",
			scoring.Result{Home: 3, Away: 0}, scoring.Prediction{Home: 4, Away: 0}, true, 1},
		{"home: draw",
			scoring.Result{Home: 1, Away: 1}, scoring.Prediction{Home: 2, Away: 2}, true, 1},
		{"home: lose by 1",
			scoring.Result{Home: 0, Away: 1}, scoring.Prediction{Home: 1, Away: 2}, true, 1},
		{"home: lose by 2+ (boundary)",
			scoring.Result{Home: 0, Away: 2}, scoring.Prediction{Home: 0, Away: 5}, true, 1},
		{"home: exact score in win-by-1 bucket",
			scoring.Result{Home: 2, Away: 1}, scoring.Prediction{Home: 2, Away: 1}, true, 1},

		// Columbus is away — crew margin = away - home
		{"away: actual lose-by-2+ matches predicted lose-by-2+",
			scoring.Result{Home: 2, Away: 0}, scoring.Prediction{Home: 3, Away: 1}, false, 1},
		{"away: predicted win by 2+, actual lose by 1 (different category)",
			scoring.Result{Home: 1, Away: 0}, scoring.Prediction{Home: 0, Away: 2}, false, 0},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			if got := scoring.Grouchy(c.result, c.pred, c.crewHome); got != c.want {
				t.Errorf("Grouchy(%v, %v, crewHome=%v) = %d, want %d",
					c.result, c.pred, c.crewHome, got, c.want)
			}
		})
	}
}
