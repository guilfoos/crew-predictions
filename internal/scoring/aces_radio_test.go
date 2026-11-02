package scoring_test

import (
	"testing"

	"github.com/guilfoos/crew-predictions/internal/scoring"
)

func TestAcesRadio(t *testing.T) {
	cases := []struct {
		desc   string
		result scoring.Result
		pred   scoring.Prediction
		want   int
	}{
		{"exact score", scoring.Result{Home: 2, Away: 0}, scoring.Prediction{Home: 2, Away: 0}, 15},
		{"correct winner", scoring.Result{Home: 2, Away: 0}, scoring.Prediction{Home: 3, Away: 1}, 10},
		{"flipped same scoreline (Crew 3-2 Portland vs actual Portland 3-2 Crew)",
			scoring.Result{Home: 3, Away: 2}, scoring.Prediction{Home: 2, Away: 3}, -15},
		{"wrong winner, wrong score", scoring.Result{Home: 2, Away: 0}, scoring.Prediction{Home: 0, Away: 1}, 0},
		{"correct draw prediction", scoring.Result{Home: 1, Away: 1}, scoring.Prediction{Home: 2, Away: 2}, 10},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			if got := scoring.AcesRadio(c.result, c.pred); got != c.want {
				t.Errorf("AcesRadio(%v, %v) = %d, want %d", c.result, c.pred, got, c.want)
			}
		})
	}
}
