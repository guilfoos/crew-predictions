package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/guilfoos/crew-predictions/internal/models"
	"github.com/guilfoos/crew-predictions/internal/repository"
	"github.com/guilfoos/crew-predictions/internal/scoring"
)

// formatLastPollAt renders the LastPollAt time as RFC3339 for the API. Zero
// time is rendered as "" so omitempty drops the field for unpolled matches.
func formatLastPollAt(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

type SummaryFetcher func(matchID string) (models.MatchSummary, error)

type MatchDetailHandler struct {
	predictions    repository.PredictionStore
	results        repository.ResultStore
	matches        repository.MatchStore
	users          repository.UserStore
	targetTeam     string
	summaryFetcher SummaryFetcher
}

func NewMatchDetailHandler(predictions repository.PredictionStore, results repository.ResultStore, matches repository.MatchStore, users repository.UserStore, targetTeam string, fetcher SummaryFetcher) *MatchDetailHandler {
	return &MatchDetailHandler{predictions: predictions, results: results, matches: matches, users: users, targetTeam: targetTeam, summaryFetcher: fetcher}
}

type matchDetailPrediction struct {
	UserID            string `json:"userID"`
	Handle            string `json:"handle"`
	HomeGoals         int    `json:"homeGoals"`
	AwayGoals         int    `json:"awayGoals"`
	AcesRadioPoints   int    `json:"acesRadioPoints"`
	Upper90ClubPoints int    `json:"upper90ClubPoints"`
	GrouchyPoints     int    `json:"grouchyPoints"`
}

type matchDetailMatch struct {
	ID           string              `json:"id"`
	HomeTeam     string              `json:"homeTeam"`
	AwayTeam     string              `json:"awayTeam"`
	Kickoff      string              `json:"kickoff"`
	HomeScore    string              `json:"homeScore"`
	AwayScore    string              `json:"awayScore"`
	State        string              `json:"state"`
	Status       string              `json:"status"`
	DisplayClock string              `json:"displayClock,omitempty"`
	Venue        string              `json:"venue,omitempty"`
	HomeRecord   string              `json:"homeRecord,omitempty"`
	AwayRecord   string              `json:"awayRecord,omitempty"`
	HomeForm     string              `json:"homeForm,omitempty"`
	AwayForm     string              `json:"awayForm,omitempty"`
	HomeLogo     string              `json:"homeLogo,omitempty"`
	AwayLogo     string              `json:"awayLogo,omitempty"`
	Attendance   int                 `json:"attendance,omitempty"`
	Referee      string              `json:"referee,omitempty"`
	Events       []models.MatchEvent `json:"events,omitempty"`
	LastPollAt   string              `json:"lastPollAt,omitempty"`
}

func (h *MatchDetailHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	matchID := r.PathValue("matchId")

	allMatches, err := h.matches.GetAll()
	if err != nil {
		http.Error(w, "could not load matches", http.StatusInternalServerError)
		return
	}
	var found *matchDetailMatch
	for i, m := range allMatches {
		if m.ID == matchID {
			if h.summaryFetcher != nil && m.State == "post" && m.Attendance == 0 {
				if summary, err := h.summaryFetcher(matchID); err == nil && (summary.Attendance > 0 || len(summary.Events) > 0) {
					allMatches[i].Attendance = summary.Attendance
					allMatches[i].Events = summary.Events
					allMatches[i].HomeLogo = summary.HomeLogo
					allMatches[i].AwayLogo = summary.AwayLogo
					allMatches[i].Referee = summary.Referee
					_ = h.matches.SaveAll(allMatches)
					m = allMatches[i]
				}
			}
			found = &matchDetailMatch{
				ID:           m.ID,
				HomeTeam:     m.HomeTeam,
				AwayTeam:     m.AwayTeam,
				Kickoff:      m.Kickoff.Format("2006-01-02T15:04:05Z07:00"),
				HomeScore:    m.HomeScore,
				AwayScore:    m.AwayScore,
				State:        m.State,
				Status:       m.Status,
				DisplayClock: m.DisplayClock,
				Venue:        m.Venue,
				HomeRecord:   m.HomeRecord,
				AwayRecord:   m.AwayRecord,
				HomeForm:     m.HomeForm,
				AwayForm:     m.AwayForm,
				HomeLogo:     m.HomeLogo,
				AwayLogo:     m.AwayLogo,
				Attendance:   m.Attendance,
				Referee:      m.Referee,
				Events:       m.Events,
				LastPollAt:   formatLastPollAt(m.LastPollAt),
			}
			break
		}
	}
	if found == nil {
		http.Error(w, "match not found", http.StatusNotFound)
		return
	}

	preds, err := h.predictions.GetByMatch(ctx, matchID)
	if err != nil {
		http.Error(w, "could not load predictions", http.StatusInternalServerError)
		return
	}

	allUsers, _ := h.users.GetAll(ctx)
	handleByUserID := make(map[string]string, len(allUsers))
	for _, u := range allUsers {
		handleByUserID[u.UserID] = u.Handle
	}

	result, _ := h.results.GetResult(ctx, matchID)

	// For live matches with a current score but no final result yet, project points.
	isProjected := false
	if result == nil && found.State == "in" && found.HomeScore != "" && found.AwayScore != "" {
		homeGoals, errH := strconv.Atoi(found.HomeScore)
		awayGoals, errA := strconv.Atoi(found.AwayScore)
		if errH == nil && errA == nil {
			result = &repository.Result{
				MatchID:   matchID,
				HomeTeam:  found.HomeTeam,
				AwayTeam:  found.AwayTeam,
				HomeGoals: homeGoals,
				AwayGoals: awayGoals,
			}
			isProjected = true
		}
	}

	entries := make([]matchDetailPrediction, 0, len(preds))
	for _, p := range preds {
		handle := handleByUserID[p.UserID]
		entry := matchDetailPrediction{
			UserID:    p.UserID,
			Handle:    handle,
			HomeGoals: p.HomeGoals,
			AwayGoals: p.AwayGoals,
		}
		if result != nil {
			pred := scoring.Prediction{Home: p.HomeGoals, Away: p.AwayGoals}
			res := scoring.Result{Home: result.HomeGoals, Away: result.AwayGoals}
			targetIsHome := result.HomeTeam == h.targetTeam
			entry.AcesRadioPoints = scoring.AcesRadio(res, pred)
			entry.Upper90ClubPoints = scoring.Upper90Club(res, pred, targetIsHome)
			entry.GrouchyPoints = scoring.Grouchy(res, pred, targetIsHome)
		}
		entries = append(entries, entry)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"match":       found,
		"isProjected": isProjected,
		"scoringFormats": []map[string]string{
			{"key": "acesRadio", "label": "Aces Radio"},
			{"key": "upper90Club", "label": "Upper 90 Club"},
			{"key": "grouchy", "label": "Grouchy\u2122"},
		},
		"predictions": entries,
	}); err != nil {
		log.Printf("match_detail: encode response: %v", err)
	}
}
