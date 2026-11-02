package scoring_test

import (
	"testing"

	"github.com/guilfoos/crew-predictions/internal/scoring"
)

// Real match data from the 2026 Columbus Crew season.
// Match 1: Portland (H) vs Columbus (A) — final 3-2 Portland
// Match 2: Kansas City (H) vs Columbus (A) — final 2-2 draw
// Columbus is the away team in both matches (columbusIsHome=false).

var u90match1 = scoring.Result{Home: 3, Away: 2}
var u90match2 = scoring.Result{Home: 2, Away: 2}

func TestUpper90Club_RealData_TwoOneBot(t *testing.T) {
	pts := scoring.Upper90Club(u90match1, scoring.Prediction{Home: 1, Away: 2}, false) // Portland 1-2 Columbus: wrong winner, correct Columbus goals → +1
	pts += scoring.Upper90Club(u90match2, scoring.Prediction{Home: 1, Away: 2}, false) // KC 1-2 Columbus: wrong winner, correct Columbus goals → +1
	if pts != 2 {
		t.Errorf("TwoOneBot: expected 2 points, got %d", pts)
	}
}

func TestUpper90Club_RealData_Haws(t *testing.T) {
	pts := scoring.Upper90Club(u90match1, scoring.Prediction{Home: 0, Away: 2}, false) // Portland 0-2 Columbus: wrong winner, correct Columbus goals → +1
	pts += scoring.Upper90Club(u90match2, scoring.Prediction{Home: 1, Away: 2}, false) // KC 1-2 Columbus: wrong winner, correct Columbus goals → +1
	if pts != 2 {
		t.Errorf("Haws: expected 2 points, got %d", pts)
	}
}

func TestUpper90Club_RealData_Mars(t *testing.T) {
	pts := scoring.Upper90Club(u90match1, scoring.Prediction{Home: 1, Away: 2}, false) // Portland 1-2 Columbus: wrong winner, correct Columbus goals → +1
	pts += scoring.Upper90Club(u90match2, scoring.Prediction{Home: 1, Away: 3}, false) // KC 1-3 Columbus: wrong winner, wrong Columbus goals → 0
	if pts != 1 {
		t.Errorf("Mars: expected 1 point, got %d", pts)
	}
}

func TestUpper90Club_RealData_Ben(t *testing.T) {
	pts := scoring.Upper90Club(u90match1, scoring.Prediction{Home: 2, Away: 2}, false) // Portland 2-2 Columbus: wrong outcome (draw), correct Columbus goals → +1
	pts += scoring.Upper90Club(u90match2, scoring.Prediction{Home: 2, Away: 3}, false) // KC 2-3 Columbus: wrong outcome, wrong Columbus goals, correct KC goals (2=2) → +1
	if pts != 2 {
		t.Errorf("Ben: expected 2 points, got %d", pts)
	}
}

func TestUpper90Club_RealData_Tre(t *testing.T) {
	pts := scoring.Upper90Club(u90match1, scoring.Prediction{Home: 1, Away: 2}, false) // Portland 1-2 Columbus: wrong winner, correct Columbus goals → +1
	pts += scoring.Upper90Club(u90match2, scoring.Prediction{Home: 0, Away: 4}, false) // KC 0-4 Columbus: wrong winner, wrong Columbus goals → 0
	if pts != 1 {
		t.Errorf("Tre: expected 1 point, got %d", pts)
	}
}

func TestUpper90Club_RealData_Mort(t *testing.T) {
	pts := scoring.Upper90Club(u90match1, scoring.Prediction{Home: 1, Away: 1}, false) // Portland 1-1 Columbus: wrong winner (draw), wrong Columbus goals → 0
	pts += scoring.Upper90Club(u90match2, scoring.Prediction{Home: 0, Away: 3}, false) // KC 0-3 Columbus: wrong winner, wrong Columbus goals → 0
	if pts != 0 {
		t.Errorf("Mort: expected 0 points, got %d", pts)
	}
}
