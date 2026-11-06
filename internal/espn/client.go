// Package espn fetches Columbus Crew match schedules and summaries from the
// public ESPN site API. It exposes typed match records and event timelines
// that the rest of the service consumes.
package espn

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/guilfoos/crew-predictions/internal/models"
)

// scoreField handles ESPN's polymorphic score value: either an integer (upcoming)
// or an object with displayValue (completed matches).
type scoreField struct {
	Display string
}

func (s *scoreField) UnmarshalJSON(data []byte) error {
	// Completed matches send {"displayValue":"2",...}
	var obj struct {
		DisplayValue string `json:"displayValue"`
	}
	if err := json.Unmarshal(data, &obj); err == nil && obj.DisplayValue != "" {
		s.Display = obj.DisplayValue
		return nil
	}
	// Live matches send a plain string "2"
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		s.Display = str
	}
	return nil
}

// leagueSlugs are the ESPN league identifiers to check for Columbus Crew matches.
// Friendlies are excluded by omission.
var leagueSlugs = []string{
	"usa.1",                // MLS
	"usa.open",             // US Open Cup
	"concacaf.leagues.cup", // Leagues Cup
	"concacaf.champions",   // CONCACAF Champions Cup
}

const teamID = "183"
const espnBase = "https://site.api.espn.com/apis/site/v2/sports/soccer"

type espnResponse struct {
	Events []struct {
		ID           string `json:"id"`
		Date         string `json:"date"`
		Competitions []struct {
			Competitors []struct {
				HomeAway string     `json:"homeAway"`
				Score    scoreField `json:"score"`
				Team     struct {
					DisplayName string `json:"displayName"`
				} `json:"team"`
				Records []struct {
					Type    string `json:"type"`
					Summary string `json:"summary"`
				} `json:"records"`
				Form string `json:"form"`
			} `json:"competitors"`
			Venue struct {
				FullName string `json:"fullName"`
			} `json:"venue"`
			Status struct {
				DisplayClock string `json:"displayClock"`
				State        string `json:"state"`
				Type         struct {
					Name string `json:"name"`
				} `json:"type"`
			} `json:"status"`
		} `json:"competitions"`
	} `json:"events"`
}

type matchRecord struct {
	id           string
	kickoff      time.Time
	home         string
	away         string
	homeScore    string
	awayScore    string
	status       string
	state        string
	displayClock string
	venue        string
	homeRecord   string
	awayRecord   string
	homeForm     string
	awayForm     string
}

func deriveState(espnState, statusName string) string {
	if espnState != "" {
		return espnState
	}
	switch statusName {
	case "STATUS_FIRST_HALF", "STATUS_SECOND_HALF", "STATUS_HALFTIME",
		"STATUS_IN_PROGRESS", "STATUS_END_PERIOD", "STATUS_OVERTIME",
		"STATUS_EXTRA_TIME", "STATUS_SHOOTOUT":
		return "in"
	case "STATUS_FULL_TIME", "STATUS_FINAL", "STATUS_FT",
		"STATUS_FULL_PEN", "STATUS_FINAL_PEN", "STATUS_FINAL_AET", "STATUS_ABANDONED":
		return "post"
	default:
		return "pre"
	}
}

func parseKickoff(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02T15:04Z07:00", s)
}

func scheduleURL(base, league string) string {
	return fmt.Sprintf("%s/%s/teams/%s/schedule", base, league, teamID)
}

func upcomingURL(base, league string, from time.Time) string {
	return fmt.Sprintf(
		"%s/%s/scoreboard?dates=%s&limit=500",
		base, league,
		from.Format("20060102"),
	)
}

func dedupeByID(records []matchRecord) []matchRecord {
	seen := map[string]bool{}
	out := make([]matchRecord, 0, len(records))
	for _, r := range records {
		if !seen[r.id] {
			seen[r.id] = true
			out = append(out, r)
		}
	}
	return out
}

func parseEvents(data espnResponse) []matchRecord {
	var records []matchRecord
	for _, event := range data.Events {
		if len(event.Competitions) == 0 {
			continue
		}
		comp := event.Competitions[0]
		var home, away, homeScore, awayScore, homeRecord, awayRecord, homeForm, awayForm string
		for _, c := range comp.Competitors {
			record := ""
			for _, r := range c.Records {
				if r.Type == "total" {
					record = r.Summary
					break
				}
			}
			if c.HomeAway == "home" {
				home = c.Team.DisplayName
				homeScore = c.Score.Display
				homeRecord = record
				homeForm = c.Form
			} else {
				away = c.Team.DisplayName
				awayScore = c.Score.Display
				awayRecord = record
				awayForm = c.Form
			}
		}
		kickoff, _ := parseKickoff(event.Date)
		records = append(records, matchRecord{
			id:           event.ID,
			kickoff:      kickoff,
			home:         home,
			away:         away,
			homeScore:    homeScore,
			awayScore:    awayScore,
			status:       comp.Status.Type.Name,
			state:        deriveState(comp.Status.State, comp.Status.Type.Name),
			displayClock: comp.Status.DisplayClock,
			venue:        comp.Venue.FullName,
			homeRecord:   homeRecord,
			awayRecord:   awayRecord,
			homeForm:     homeForm,
			awayForm:     awayForm,
		})
	}
	return records
}

func fetchAndParse(url string) ([]matchRecord, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, nil // league may not have data; skip silently
	}
	var data espnResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	return parseEvents(data), nil
}

func isCrewMatch(r matchRecord) bool {
	return r.home == "Columbus Crew" || r.away == "Columbus Crew"
}

func fetchCrewMatchesFrom(base string) ([]models.Match, error) {
	now := time.Now().UTC()
	var all []matchRecord

	for _, league := range leagueSlugs {
		past, err := fetchAndParse(scheduleURL(base, league))
		if err != nil {
			return nil, fmt.Errorf("espn schedule fetch (%s): %w", league, err)
		}
		for _, r := range past {
			if isCrewMatch(r) {
				all = append(all, r)
			}
		}

		// ESPN changed their API to no longer accept date ranges, so we have to
		// loop on our end to poll each day
		// Start 2 days back so delayed/rescheduled matches aren't dropped when
		// ESPN still indexes them under their original kickoff date.
		start := now.AddDate(0, 0, -2)
		end := now.AddDate(0, 0, 8)

		for d := start; d.Before(end); d = d.AddDate(0, 0, 1) {
			upcoming, err := fetchAndParse(upcomingURL(base, league, d))
			if err != nil {
				return nil, fmt.Errorf("espn scoreboard fetch (%s): %w", league, err)
			}
			for _, r := range upcoming {
				if isCrewMatch(r) {
					all = append(all, r)
				}
			}
		}
	}

	all = dedupeByID(all)
	sort.Slice(all, func(i, j int) bool {
		return all[i].kickoff.Before(all[j].kickoff)
	})

	matches := make([]models.Match, len(all))
	for i, r := range all {
		matches[i] = models.Match{
			ID:           r.id,
			HomeTeam:     r.home,
			AwayTeam:     r.away,
			Kickoff:      r.kickoff,
			Status:       r.status,
			HomeScore:    r.homeScore,
			AwayScore:    r.awayScore,
			State:        r.state,
			DisplayClock: r.displayClock,
			Venue:        r.venue,
			HomeRecord:   r.homeRecord,
			AwayRecord:   r.awayRecord,
			HomeForm:     r.homeForm,
			AwayForm:     r.awayForm,
		}
	}
	return matches, nil
}

type espnSummaryResponse struct {
	GameInfo struct {
		Attendance int `json:"attendance"`
		Officials  []struct {
			DisplayName string `json:"displayName"`
			Order       int    `json:"order"`
		} `json:"officials"`
	} `json:"gameInfo"`
	Boxscore struct {
		Teams []struct {
			HomeAway string `json:"homeAway"`
			Team     struct {
				Logo string `json:"logo"`
			} `json:"team"`
		} `json:"teams"`
	} `json:"boxscore"`
	KeyEvents []struct {
		Clock struct {
			DisplayValue string `json:"displayValue"`
		} `json:"clock"`
		Type struct {
			Type string `json:"type"`
		} `json:"type"`
		Team struct {
			DisplayName string `json:"displayName"`
		} `json:"team"`
		Participants []struct {
			Athlete struct {
				DisplayName string `json:"displayName"`
			} `json:"athlete"`
		} `json:"participants"`
	} `json:"keyEvents"`
}

func summaryURL(base, matchID string) string {
	return fmt.Sprintf("%s/usa.1/summary?event=%s", base, matchID)
}

func fetchSummaryFrom(base, matchID string) (models.MatchSummary, error) {
	resp, err := http.Get(summaryURL(base, matchID))
	if err != nil {
		return models.MatchSummary{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return models.MatchSummary{}, nil
	}
	var data espnSummaryResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return models.MatchSummary{}, err
	}
	return parseSummary(data), nil
}

// parseSummary extracts a MatchSummary from a decoded ESPN /summary response.
// Shared between the live HTTP fetcher and the test-mode fixture fetcher so
// the two paths stay consistent.
func parseSummary(data espnSummaryResponse) models.MatchSummary {
	events := make([]models.MatchEvent, 0, len(data.KeyEvents))
	for _, ke := range data.KeyEvents {
		players := make([]string, 0, len(ke.Participants))
		for _, p := range ke.Participants {
			players = append(players, p.Athlete.DisplayName)
		}
		events = append(events, models.MatchEvent{
			Clock:   ke.Clock.DisplayValue,
			TypeID:  ke.Type.Type,
			Team:    ke.Team.DisplayName,
			Players: players,
		})
	}
	var homeLogo, awayLogo string
	for _, t := range data.Boxscore.Teams {
		switch t.HomeAway {
		case "home":
			homeLogo = t.Team.Logo
		case "away":
			awayLogo = t.Team.Logo
		}
	}
	var referee string
	if len(data.GameInfo.Officials) > 0 {
		referee = data.GameInfo.Officials[0].DisplayName
	}
	return models.MatchSummary{
		Attendance: data.GameInfo.Attendance,
		HomeLogo:   homeLogo,
		AwayLogo:   awayLogo,
		Referee:    referee,
		Events:     events,
	}
}

func FetchSummary(matchID string) (models.MatchSummary, error) {
	return fetchSummaryFrom(espnBase, matchID)
}

func FetchCrewMatches() ([]models.Match, error) {
	return fetchCrewMatchesFrom(espnBase)
}
