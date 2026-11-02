package espn

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/guilfoos/crew-predictions/internal/models"
)

// FixtureFetcher returns a SummaryFetcher backed by canned ESPN /summary
// responses on disk. Used in TEST_MODE so the e2e suite never depends on
// the live ESPN endpoint — tests stay deterministic even when ESPN is slow
// or down. Fixtures are stored as `summary_<matchID>.json` files in the
// given directory; missing files return an empty MatchSummary (mimicking
// ESPN's 404 path) so seeded fake match IDs don't cause failures.
func FixtureFetcher(dir string) func(matchID string) (models.MatchSummary, error) {
	return func(matchID string) (models.MatchSummary, error) {
		path := filepath.Join(dir, "summary_"+matchID+".json")
		raw, err := os.ReadFile(path)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return models.MatchSummary{}, nil
			}
			return models.MatchSummary{}, err
		}
		var data espnSummaryResponse
		if err := json.Unmarshal(raw, &data); err != nil {
			return models.MatchSummary{}, err
		}
		return parseSummary(data), nil
	}
}
