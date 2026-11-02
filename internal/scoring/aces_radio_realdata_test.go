package scoring_test

import (
	"testing"

	"github.com/guilfoos/crew-predictions/internal/scoring"
)

// Real match data from the 2026 Columbus Crew season.
// Match 1: Portland (H) vs Columbus (A) — final 3-2 Portland
// Match 2: Kansas City (H) vs Columbus (A) — final 2-2 draw

var match1Result = scoring.Result{Home: 3, Away: 2}
var match2Result = scoring.Result{Home: 2, Away: 2}

func TestAcesRadio_RealData_Darby(t *testing.T) {
	pts := scoring.AcesRadio(match1Result, scoring.Prediction{Home: 3, Away: 1}) // Portland 3-1
	pts += scoring.AcesRadio(match2Result, scoring.Prediction{Home: 1, Away: 2}) // KC 1-2 Columbus
	if pts != 10 {
		t.Errorf("Darby: expected 10 points, got %d", pts)
	}
}

func TestAcesRadio_RealData_Alex(t *testing.T) {
	pts := scoring.AcesRadio(match1Result, scoring.Prediction{Home: 3, Away: 0}) // Portland 3-0
	pts += scoring.AcesRadio(match2Result, scoring.Prediction{Home: 0, Away: 2}) // KC 0-2 Columbus
	if pts != 10 {
		t.Errorf("Alex: expected 10 points, got %d", pts)
	}
}

func TestAcesRadio_RealData_Zidar(t *testing.T) {
	// Predicted Columbus wins 3-2 (away team wins), actual Portland 3-2 — flipped same scoreline
	pts := scoring.AcesRadio(match1Result, scoring.Prediction{Home: 2, Away: 3})
	pts += scoring.AcesRadio(match2Result, scoring.Prediction{Home: 1, Away: 3}) // KC 1-3 Columbus
	if pts != -15 {
		t.Errorf("Zidar: expected -15 points, got %d", pts)
	}
}

func TestAcesRadio_RealData_Lumbus(t *testing.T) {
	pts := scoring.AcesRadio(match1Result, scoring.Prediction{Home: 2, Away: 4}) // Columbus wins 4-2 — wrong winner
	pts += scoring.AcesRadio(match2Result, scoring.Prediction{Home: 1, Away: 3}) // KC 1-3 Columbus
	if pts != 0 {
		t.Errorf("Lumbus: expected 0 points, got %d", pts)
	}
}

func TestAcesRadio_RealData_Morgan(t *testing.T) {
	pts := scoring.AcesRadio(match1Result, scoring.Prediction{Home: 2, Away: 0}) // Portland 2-0 — correct winner
	pts += scoring.AcesRadio(match2Result, scoring.Prediction{Home: 0, Away: 3}) // KC 0-3 Columbus
	if pts != 10 {
		t.Errorf("Morgan: expected 10 points, got %d", pts)
	}
}
