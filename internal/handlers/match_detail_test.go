package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/guilfoos/crew-predictions/internal/handlers"
	"github.com/guilfoos/crew-predictions/internal/models"
	"github.com/guilfoos/crew-predictions/internal/repository"
)

// getMatchDetail builds a MatchDetailHandler with sensible defaults
// (empty prediction/user stores, "Columbus Crew" target team), invokes
// it for matchID, and returns the response recorder. Tests that need a
// non-default prediction store or user store should construct their own
// handler instead of using this helper.
func getMatchDetail(t *testing.T, store repository.MatchStore, fetcher handlers.SummaryFetcher, matchID string) *httptest.ResponseRecorder {
	t.Helper()
	h := handlers.NewMatchDetailHandler(
		repository.NewMemoryPredictionStore(),
		repository.NewMemoryResultStore(),
		store,
		repository.NewMemoryUserStore(),
		"Columbus Crew",
		fetcher,
	)
	req := httptest.NewRequest("GET", "/api/matches/"+matchID, http.NoBody)
	req.SetPathValue("matchId", matchID)
	w := httptest.NewRecorder()
	h.Get(w, req)
	return w
}

func TestMatchDetailHandler_ReturnsPredictionsWithScores(t *testing.T) {
	predStore := repository.NewMemoryPredictionStore()
	resultStore := repository.NewMemoryResultStore()
	matchStore := repository.NewMemoryMatchStore()
	ctx := context.Background()

	match := models.Match{
		ID:        "m-test",
		HomeTeam:  "Columbus Crew",
		AwayTeam:  "FC Dallas",
		Kickoff:   time.Now().Add(-24 * time.Hour),
		Status:    "STATUS_FULL_TIME",
		HomeScore: "2",
		AwayScore: "1",
	}
	matchStore.Seed([]models.Match{match})

	userStore := repository.NewMemoryUserStore()
	userStore.Upsert(ctx, repository.User{UserID: "google:u1", Handle: "fan1@bsky.mock"})
	userStore.Upsert(ctx, repository.User{UserID: "google:u2", Handle: "fan2@bsky.mock"})

	predStore.Save(ctx, repository.Prediction{MatchID: "m-test", UserID: "google:u1", HomeGoals: 2, AwayGoals: 1})
	predStore.Save(ctx, repository.Prediction{MatchID: "m-test", UserID: "google:u2", HomeGoals: 0, AwayGoals: 0})

	resultStore.SaveResult(ctx, repository.Result{
		MatchID:   "m-test",
		HomeTeam:  "Columbus Crew",
		AwayTeam:  "FC Dallas",
		HomeGoals: 2,
		AwayGoals: 1,
	})

	h := handlers.NewMatchDetailHandler(predStore, resultStore, matchStore, userStore, "Columbus Crew", nil)

	req := httptest.NewRequest("GET", "/api/matches/m-test", http.NoBody)
	req.SetPathValue("matchId", "m-test")
	w := httptest.NewRecorder()
	h.Get(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}

	predictions, ok := resp["predictions"].([]any)
	if !ok {
		t.Fatal("expected predictions array in response")
	}
	if len(predictions) != 2 {
		t.Errorf("expected 2 predictions, got %d", len(predictions))
	}

	// fan1 predicted exactly right (2-1) — should have more Aces Radio points
	var fan1Points, fan2Points float64
	for _, p := range predictions {
		entry := p.(map[string]any)
		if entry["handle"] == "fan1@bsky.mock" {
			fan1Points = entry["acesRadioPoints"].(float64)
		}
		if entry["handle"] == "fan2@bsky.mock" {
			fan2Points = entry["acesRadioPoints"].(float64)
		}
	}
	if fan1Points <= fan2Points {
		t.Errorf("fan1 (exact) should have more points than fan2, got %v vs %v", fan1Points, fan2Points)
	}
}

func TestMatchDetailHandler_IncludesGrouchyPoints(t *testing.T) {
	predStore := repository.NewMemoryPredictionStore()
	resultStore := repository.NewMemoryResultStore()
	matchStore := repository.NewMemoryMatchStore()
	ctx := context.Background()

	matchStore.Seed([]models.Match{{
		ID: "m-grouchy", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas",
		Kickoff: time.Now().Add(-24 * time.Hour), Status: "STATUS_FULL_TIME",
		HomeScore: "2", AwayScore: "0",
	}})
	// Columbus home, predicted 3-0 (Win by 2+), actual 2-0 (Win by 2+) → 1 pt
	predStore.Save(ctx, repository.Prediction{MatchID: "m-grouchy", UserID: "u1", HomeGoals: 3, AwayGoals: 0})
	resultStore.SaveResult(ctx, repository.Result{
		MatchID: "m-grouchy", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas", HomeGoals: 2, AwayGoals: 0,
	})

	h := handlers.NewMatchDetailHandler(predStore, resultStore, matchStore, repository.NewMemoryUserStore(), "Columbus Crew", nil)
	req := httptest.NewRequest("GET", "/api/matches/m-grouchy", http.NoBody)
	req.SetPathValue("matchId", "m-grouchy")
	w := httptest.NewRecorder()
	h.Get(w, req)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	predictions := resp["predictions"].([]any)
	grouchy := predictions[0].(map[string]any)["grouchyPoints"].(float64)
	if grouchy != 1 {
		t.Errorf("expected grouchyPoints=1, got %v", grouchy)
	}
}

func TestMatchDetailHandler_ScoringFormatsIncludesGrouchy(t *testing.T) {
	predStore := repository.NewMemoryPredictionStore()
	resultStore := repository.NewMemoryResultStore()
	matchStore := repository.NewMemoryMatchStore()
	matchStore.Seed([]models.Match{{
		ID: "m-fmt", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas",
		Kickoff: time.Now().Add(-24 * time.Hour), Status: "STATUS_FULL_TIME",
		HomeScore: "1", AwayScore: "0",
	}})

	h := handlers.NewMatchDetailHandler(predStore, resultStore, matchStore, repository.NewMemoryUserStore(), "Columbus Crew", nil)
	req := httptest.NewRequest("GET", "/api/matches/m-fmt", http.NoBody)
	req.SetPathValue("matchId", "m-fmt")
	w := httptest.NewRecorder()
	h.Get(w, req)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	formats := resp["scoringFormats"].([]any)
	keys := make([]string, 0, len(formats))
	for _, f := range formats {
		keys = append(keys, f.(map[string]any)["key"].(string))
	}
	found := false
	for _, k := range keys {
		if k == "grouchy" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected grouchy in scoringFormats, got %v", keys)
	}
}

func TestMatchDetailHandler_UsesCurrentHandleFromUserStore(t *testing.T) {
	predStore := repository.NewMemoryPredictionStore()
	resultStore := repository.NewMemoryResultStore()
	matchStore := repository.NewMemoryMatchStore()
	userStore := repository.NewMemoryUserStore()
	ctx := context.Background()

	matchStore.Seed([]models.Match{{
		ID: "m-handle", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas",
		Kickoff: time.Now().Add(-24 * time.Hour), Status: "STATUS_FULL_TIME",
		HomeScore: "1", AwayScore: "0",
	}})
	predStore.Save(ctx, repository.Prediction{MatchID: "m-handle", UserID: "google:u1", HomeGoals: 1, AwayGoals: 0})
	userStore.Upsert(ctx, repository.User{UserID: "google:u1", Handle: "BlackAndGold"})

	h := handlers.NewMatchDetailHandler(predStore, resultStore, matchStore, userStore, "Columbus Crew", nil)

	req := httptest.NewRequest("GET", "/api/matches/m-handle", http.NoBody)
	req.SetPathValue("matchId", "m-handle")
	w := httptest.NewRecorder()
	h.Get(w, req)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	predictions := resp["predictions"].([]any)
	handle := predictions[0].(map[string]any)["handle"].(string)
	if handle != "BlackAndGold" {
		t.Errorf("expected handle from UserStore %q, got %q", "BlackAndGold", handle)
	}
}

func TestMatchDetailHandler_ShowsEmptyHandleWhenUserNotInStore(t *testing.T) {
	predStore := repository.NewMemoryPredictionStore()
	resultStore := repository.NewMemoryResultStore()
	matchStore := repository.NewMemoryMatchStore()
	userStore := repository.NewMemoryUserStore()
	ctx := context.Background()

	matchStore.Seed([]models.Match{{
		ID: "m-no-user", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas",
		Kickoff: time.Now().Add(-24 * time.Hour), Status: "STATUS_FULL_TIME",
		HomeScore: "1", AwayScore: "0",
	}})
	predStore.Save(ctx, repository.Prediction{MatchID: "m-no-user", UserID: "google:orphan", HomeGoals: 1, AwayGoals: 0})

	h := handlers.NewMatchDetailHandler(predStore, resultStore, matchStore, userStore, "Columbus Crew", nil)
	req := httptest.NewRequest("GET", "/api/matches/m-no-user", http.NoBody)
	req.SetPathValue("matchId", "m-no-user")
	w := httptest.NewRecorder()
	h.Get(w, req)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	predictions := resp["predictions"].([]any)
	handle := predictions[0].(map[string]any)["handle"].(string)
	if handle != "" {
		t.Errorf("expected empty handle when user not in UserStore, got %q", handle)
	}
}

func TestMatchDetailHandler_LiveMatchProjectsScores(t *testing.T) {
	predStore := repository.NewMemoryPredictionStore()
	resultStore := repository.NewMemoryResultStore()
	matchStore := repository.NewMemoryMatchStore()
	ctx := context.Background()

	matchStore.Seed([]models.Match{{
		ID: "m-live", HomeTeam: "Columbus Crew", AwayTeam: "Philadelphia Union",
		Kickoff: time.Now().Add(-45 * time.Minute), Status: "STATUS_IN_PROGRESS",
		State: "in", HomeScore: "2", AwayScore: "0",
	}})
	userStore := repository.NewMemoryUserStore()
	userStore.Upsert(ctx, repository.User{UserID: "u1", Handle: "exact@bsky.mock"})
	userStore.Upsert(ctx, repository.User{UserID: "u2", Handle: "wrong@bsky.mock"})

	predStore.Save(ctx, repository.Prediction{MatchID: "m-live", UserID: "u1", HomeGoals: 2, AwayGoals: 0})
	predStore.Save(ctx, repository.Prediction{MatchID: "m-live", UserID: "u2", HomeGoals: 1, AwayGoals: 1})

	h := handlers.NewMatchDetailHandler(predStore, resultStore, matchStore, userStore, "Columbus Crew", nil)
	req := httptest.NewRequest("GET", "/api/matches/m-live", http.NoBody)
	req.SetPathValue("matchId", "m-live")
	w := httptest.NewRecorder()
	h.Get(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)

	match := resp["match"].(map[string]any)
	if match["state"] != "in" {
		t.Errorf("expected state=in, got %v", match["state"])
	}
	if resp["isProjected"] != true {
		t.Errorf("expected isProjected=true for live match, got %v", resp["isProjected"])
	}

	predictions := resp["predictions"].([]any)
	var exactPts, wrongPts float64
	for _, p := range predictions {
		entry := p.(map[string]any)
		if entry["handle"] == "exact@bsky.mock" {
			exactPts = entry["acesRadioPoints"].(float64)
		}
		if entry["handle"] == "wrong@bsky.mock" {
			wrongPts = entry["acesRadioPoints"].(float64)
		}
	}
	if exactPts <= wrongPts {
		t.Errorf("exact predictor should outscore wrong predictor, got %v vs %v", exactPts, wrongPts)
	}
}

func TestMatchDetailHandler_LiveMatchWithNoScoreDoesNotProject(t *testing.T) {
	predStore := repository.NewMemoryPredictionStore()
	resultStore := repository.NewMemoryResultStore()
	matchStore := repository.NewMemoryMatchStore()
	ctx := context.Background()

	matchStore.Seed([]models.Match{{
		ID: "m-live-noscore", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas",
		Kickoff: time.Now().Add(-5 * time.Minute), Status: "STATUS_IN_PROGRESS",
		State: "in", HomeScore: "", AwayScore: "",
	}})
	predStore.Save(ctx, repository.Prediction{MatchID: "m-live-noscore", UserID: "u1", HomeGoals: 1, AwayGoals: 0})

	h := handlers.NewMatchDetailHandler(predStore, resultStore, matchStore, repository.NewMemoryUserStore(), "Columbus Crew", nil)
	req := httptest.NewRequest("GET", "/api/matches/m-live-noscore", http.NoBody)
	req.SetPathValue("matchId", "m-live-noscore")
	w := httptest.NewRecorder()
	h.Get(w, req)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["isProjected"] == true {
		t.Error("expected isProjected=false when no live score available")
	}
}

func TestMatchDetailHandler_Returns404ForUnknownMatch(t *testing.T) {
	predStore := repository.NewMemoryPredictionStore()
	resultStore := repository.NewMemoryResultStore()
	matchStore := repository.NewMemoryMatchStore()
	userStore := repository.NewMemoryUserStore()

	h := handlers.NewMatchDetailHandler(predStore, resultStore, matchStore, userStore, "Columbus Crew", nil)

	req := httptest.NewRequest("GET", "/api/matches/no-such-match", http.NoBody)
	req.SetPathValue("matchId", "no-such-match")
	w := httptest.NewRecorder()
	h.Get(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestMatchDetailHandler_LiveMatchWithNonNumericScoreDoesNotProject(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	matchStore.Seed([]models.Match{{
		ID: "m-live-nan", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas",
		Kickoff: time.Now().Add(-5 * time.Minute), Status: "STATUS_IN_PROGRESS",
		State: "in", HomeScore: "TBD", AwayScore: "TBD",
	}})
	h := handlers.NewMatchDetailHandler(
		repository.NewMemoryPredictionStore(),
		repository.NewMemoryResultStore(),
		matchStore,
		repository.NewMemoryUserStore(),
		"Columbus Crew",
		nil,
	)
	req := httptest.NewRequest("GET", "/api/matches/m-live-nan", http.NoBody)
	req.SetPathValue("matchId", "m-live-nan")
	w := httptest.NewRecorder()
	h.Get(w, req)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["isProjected"] == true {
		t.Error("expected isProjected=false when live score is non-numeric")
	}
}

func TestMatchDetailHandler_ReturnsEmptyPredictionsWhenNone(t *testing.T) {
	predStore := repository.NewMemoryPredictionStore()
	resultStore := repository.NewMemoryResultStore()
	matchStore := repository.NewMemoryMatchStore()

	matchStore.Seed([]models.Match{{
		ID:        "m-empty",
		HomeTeam:  "Columbus Crew",
		AwayTeam:  "FC Dallas",
		Kickoff:   time.Now().Add(-24 * time.Hour),
		Status:    "STATUS_FULL_TIME",
		HomeScore: "1",
		AwayScore: "0",
	}})

	h := handlers.NewMatchDetailHandler(predStore, resultStore, matchStore, repository.NewMemoryUserStore(), "Columbus Crew", nil)

	req := httptest.NewRequest("GET", "/api/matches/m-empty", http.NoBody)
	req.SetPathValue("matchId", "m-empty")
	w := httptest.NewRecorder()
	h.Get(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	predictions := resp["predictions"].([]any)
	if len(predictions) != 0 {
		t.Errorf("expected 0 predictions, got %d", len(predictions))
	}
}

func TestMatchDetailHandler_Returns500WhenMatchStoreFails(t *testing.T) {
	h := handlers.NewMatchDetailHandler(
		repository.NewMemoryPredictionStore(),
		repository.NewMemoryResultStore(),
		repository.NewErrorGetAllMatchStore(),
		repository.NewMemoryUserStore(),
		"Columbus Crew",
		nil,
	)
	req := httptest.NewRequest("GET", "/api/matches/any", http.NoBody)
	req.SetPathValue("matchId", "any")
	w := httptest.NewRecorder()
	h.Get(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestMatchDetailHandler_Returns500WhenPredictionStoreFails(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	matchStore.Seed([]models.Match{{
		ID: "m-pred-err", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas",
		Kickoff: time.Now(), Status: "STATUS_FULL_TIME",
	}})
	h := handlers.NewMatchDetailHandler(
		repository.NewErrorGetByMatchPredictionStore(),
		repository.NewMemoryResultStore(),
		matchStore,
		repository.NewMemoryUserStore(),
		"Columbus Crew",
		nil,
	)
	req := httptest.NewRequest("GET", "/api/matches/m-pred-err", http.NoBody)
	req.SetPathValue("matchId", "m-pred-err")
	w := httptest.NewRecorder()
	h.Get(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestMatchDetailHandler_IncludesVenue(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	matchStore.Seed([]models.Match{{
		ID:        "m-venue",
		HomeTeam:  "Columbus Crew",
		AwayTeam:  "FC Dallas",
		Kickoff:   time.Now().Add(-24 * time.Hour),
		Status:    "STATUS_FULL_TIME",
		HomeScore: "2",
		AwayScore: "1",
		Venue:     "ScottsMiracle-Gro Field",
	}})

	w := getMatchDetail(t, matchStore, nil, "m-venue")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp struct {
		Match struct {
			Venue string `json:"venue"`
		} `json:"match"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.Match.Venue != "ScottsMiracle-Gro Field" {
		t.Errorf("expected venue 'ScottsMiracle-Gro Field', got %q", resp.Match.Venue)
	}
}

func TestMatchDetailHandler_LazyFetchesAttendanceForPostMatch(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	matchStore.Seed([]models.Match{{
		ID: "m-post", HomeTeam: "Columbus Crew", AwayTeam: "Philadelphia Union",
		Kickoff: time.Now().Add(-3 * time.Hour), Status: "STATUS_FULL_TIME",
		State: "post", HomeScore: "2", AwayScore: "0", Attendance: 0,
	}})

	fetcher := func(_ string) (models.MatchSummary, error) {
		return models.MatchSummary{Attendance: 19903}, nil
	}

	w := getMatchDetail(t, matchStore, fetcher, "m-post")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Match struct {
			Attendance int `json:"attendance"`
		} `json:"match"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.Match.Attendance != 19903 {
		t.Errorf("expected attendance 19903, got %d", resp.Match.Attendance)
	}
}

func TestMatchDetailHandler_WritesAttendanceBackToStore(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	matchStore.Seed([]models.Match{{
		ID: "m-wb", HomeTeam: "Columbus Crew", AwayTeam: "Philadelphia Union",
		Kickoff: time.Now().Add(-3 * time.Hour), Status: "STATUS_FULL_TIME",
		State: "post", HomeScore: "2", AwayScore: "0", Attendance: 0,
	}})

	fetcher := func(_ string) (models.MatchSummary, error) {
		return models.MatchSummary{Attendance: 5000}, nil
	}

	getMatchDetail(t, matchStore, fetcher, "m-wb")

	matches, _ := matchStore.GetAll()
	var found models.Match
	for _, m := range matches {
		if m.ID == "m-wb" {
			found = m
		}
	}
	if found.Attendance != 5000 {
		t.Errorf("expected attendance written back as 5000, got %d", found.Attendance)
	}
}

func TestMatchDetailHandler_SkipsLazyFetchWhenAttendanceAlreadySet(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	matchStore.Seed([]models.Match{{
		ID: "m-cached", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas",
		Kickoff: time.Now().Add(-3 * time.Hour), Status: "STATUS_FULL_TIME",
		State: "post", HomeScore: "1", AwayScore: "0", Attendance: 9999,
	}})

	callCount := 0
	fetcher := func(_ string) (models.MatchSummary, error) {
		callCount++
		return models.MatchSummary{Attendance: 0}, nil
	}

	getMatchDetail(t, matchStore, fetcher, "m-cached")

	if callCount != 0 {
		t.Errorf("expected fetcher not called when attendance already set, called %d times", callCount)
	}
}

func TestMatchDetailHandler_IncludesEventsInResponse(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	matchStore.Seed([]models.Match{{
		ID: "m-events", HomeTeam: "Columbus Crew", AwayTeam: "Philadelphia Union",
		Kickoff: time.Now().Add(-3 * time.Hour), Status: "STATUS_FULL_TIME",
		State: "post", HomeScore: "2", AwayScore: "0", Attendance: 0,
	}})

	fetcher := func(_ string) (models.MatchSummary, error) {
		return models.MatchSummary{
			Attendance: 19903,
			Events: []models.MatchEvent{
				{Clock: "4'", TypeID: "goal", Team: "Columbus Crew", Players: []string{"Max Arfsten"}},
				{Clock: "90'+4'", TypeID: "red-card", Team: "Philadelphia Union", Players: []string{"Japhet Sery"}},
			},
		}, nil
	}

	w := getMatchDetail(t, matchStore, fetcher, "m-events")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Match struct {
			Events []struct {
				Clock   string   `json:"clock"`
				TypeID  string   `json:"typeID"`
				Team    string   `json:"team"`
				Players []string `json:"players"`
			} `json:"events"`
		} `json:"match"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(resp.Match.Events) != 2 {
		t.Errorf("expected 2 events, got %d", len(resp.Match.Events))
	}
	if resp.Match.Events[0].TypeID != "goal" {
		t.Errorf("expected first event typeID=goal, got %q", resp.Match.Events[0].TypeID)
	}
}

func TestMatchDetailHandler_WritesEventsBackToStore(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	matchStore.Seed([]models.Match{{
		ID: "m-ev-wb", HomeTeam: "Columbus Crew", AwayTeam: "Philadelphia Union",
		Kickoff: time.Now().Add(-3 * time.Hour), Status: "STATUS_FULL_TIME",
		State: "post", HomeScore: "2", AwayScore: "0", Attendance: 0,
	}})

	fetcher := func(_ string) (models.MatchSummary, error) {
		return models.MatchSummary{
			Attendance: 5000,
			Events:     []models.MatchEvent{{Clock: "10'", TypeID: "goal", Team: "Columbus Crew", Players: []string{"Player A"}}},
		}, nil
	}

	getMatchDetail(t, matchStore, fetcher, "m-ev-wb")

	matches, _ := matchStore.GetAll()
	var found models.Match
	for _, m := range matches {
		if m.ID == "m-ev-wb" {
			found = m
		}
	}
	if len(found.Events) != 1 {
		t.Errorf("expected events written back to store, got %d events", len(found.Events))
	}
	if found.Events[0].TypeID != "goal" {
		t.Errorf("expected written-back event typeID=goal, got %q", found.Events[0].TypeID)
	}
}

func TestMatchDetailHandler_IncludesRefereeAndLogosInResponse(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	matchStore.Seed([]models.Match{{
		ID: "m-rl", HomeTeam: "Columbus Crew", AwayTeam: "Toronto FC",
		Kickoff: time.Now().Add(-3 * time.Hour), Status: "STATUS_FULL_TIME",
		State: "post", HomeScore: "2", AwayScore: "0", Attendance: 0,
	}})

	fetcher := func(_ string) (models.MatchSummary, error) {
		return models.MatchSummary{
			Attendance: 15384,
			HomeLogo:   "https://a.espncdn.com/i/teamlogos/soccer/500/183.png",
			AwayLogo:   "https://a.espncdn.com/i/teamlogos/soccer/500/7318.png",
			Referee:    "Pierre-Luc Lauziere",
		}, nil
	}

	w := getMatchDetail(t, matchStore, fetcher, "m-rl")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Match struct {
			HomeLogo string `json:"homeLogo"`
			AwayLogo string `json:"awayLogo"`
			Referee  string `json:"referee"`
		} `json:"match"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.Match.HomeLogo != "https://a.espncdn.com/i/teamlogos/soccer/500/183.png" {
		t.Errorf("HomeLogo: got %q", resp.Match.HomeLogo)
	}
	if resp.Match.AwayLogo != "https://a.espncdn.com/i/teamlogos/soccer/500/7318.png" {
		t.Errorf("AwayLogo: got %q", resp.Match.AwayLogo)
	}
	if resp.Match.Referee != "Pierre-Luc Lauziere" {
		t.Errorf("Referee: got %q", resp.Match.Referee)
	}

	// Also verify the values were persisted back to the store.
	matches, _ := matchStore.GetAll()
	var stored models.Match
	for _, m := range matches {
		if m.ID == "m-rl" {
			stored = m
		}
	}
	if stored.HomeLogo == "" || stored.AwayLogo == "" || stored.Referee == "" {
		t.Errorf("expected refs+logos written back to store, got home=%q away=%q ref=%q",
			stored.HomeLogo, stored.AwayLogo, stored.Referee)
	}
}

func TestMatchDetailHandler_IncludesRecordsAndForm(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	matchStore.SaveAll([]models.Match{{
		ID:         "m-rf",
		HomeTeam:   "Columbus Crew",
		AwayTeam:   "FC Dallas",
		Kickoff:    time.Now(),
		Status:     "STATUS_SCHEDULED",
		HomeRecord: "5-3-2",
		AwayRecord: "4-4-2",
		HomeForm:   "WWWLL",
		AwayForm:   "LWDWL",
	}})

	w := getMatchDetail(t, matchStore, nil, "m-rf")

	var resp struct {
		Match struct {
			HomeRecord string `json:"homeRecord"`
			AwayRecord string `json:"awayRecord"`
			HomeForm   string `json:"homeForm"`
			AwayForm   string `json:"awayForm"`
		} `json:"match"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.Match.HomeRecord != "5-3-2" {
		t.Errorf("HomeRecord: got %q, want %q", resp.Match.HomeRecord, "5-3-2")
	}
	if resp.Match.AwayRecord != "4-4-2" {
		t.Errorf("AwayRecord: got %q, want %q", resp.Match.AwayRecord, "4-4-2")
	}
	if resp.Match.HomeForm != "WWWLL" {
		t.Errorf("HomeForm: got %q, want %q", resp.Match.HomeForm, "WWWLL")
	}
	if resp.Match.AwayForm != "LWDWL" {
		t.Errorf("AwayForm: got %q, want %q", resp.Match.AwayForm, "LWDWL")
	}
}

func TestMatchDetailHandler_IncludesLastPollAt(t *testing.T) {
	predStore := repository.NewMemoryPredictionStore()
	resultStore := repository.NewMemoryResultStore()
	matchStore := repository.NewMemoryMatchStore()
	userStore := repository.NewMemoryUserStore()

	polledAt := time.Date(2026, 5, 13, 23, 50, 0, 0, time.UTC)
	matchStore.Seed([]models.Match{{
		ID: "m-poll", HomeTeam: "Columbus Crew", AwayTeam: "LAFC",
		Kickoff: polledAt.Add(-20 * time.Minute), Status: "STATUS_IN_PROGRESS", State: "in",
		LastPollAt: polledAt,
	}})

	h := handlers.NewMatchDetailHandler(predStore, resultStore, matchStore, userStore, "Columbus Crew", nil)
	req := httptest.NewRequest("GET", "/api/matches/m-poll", http.NoBody)
	req.SetPathValue("matchId", "m-poll")
	w := httptest.NewRecorder()
	h.Get(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	matchObj, ok := resp["match"].(map[string]any)
	if !ok {
		t.Fatalf("expected response.match to be an object, got: %T", resp["match"])
	}
	got, ok := matchObj["lastPollAt"].(string)
	if !ok || got == "" {
		t.Fatalf("expected response.match.lastPollAt to be a non-empty string, got: %v", matchObj["lastPollAt"])
	}
	parsed, err := time.Parse(time.RFC3339, got)
	if err != nil {
		t.Fatalf("expected lastPollAt to parse as RFC3339, got %q (err: %v)", got, err)
	}
	if !parsed.Equal(polledAt) {
		t.Errorf("expected lastPollAt to round-trip as %v, got %v", polledAt, parsed)
	}
}
