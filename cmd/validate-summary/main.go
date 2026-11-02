//go:build manual

// Validates the ESPN /summary parser against every completed Columbus Crew
// match this season. Run manually (the `manual` build tag excludes this file
// from default builds, tests, and coverage):
//
//	go run -tags manual ./cmd/validate-summary
//
// For each completed match, prints attendance, total events, displayable
// events, and the distinct event types observed. Useful to spot matches with
// missing attendance or new event types our parser hasn't seen.
package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/guilfoos/crew-predictions/internal/espn"
)

var nonDisplayable = map[string]bool{
	"kickoff":          true,
	"halftime":         true,
	"start-2nd-half":   true,
	"end-regular-time": true,
}

func main() {
	matches, err := espn.FetchCrewMatches()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch matches: %v\n", err)
		os.Exit(1)
	}

	completed := matches[:0]
	for _, m := range matches {
		if m.State == "post" {
			completed = append(completed, m)
		}
	}
	if len(completed) == 0 {
		fmt.Println("no completed matches found")
		return
	}

	fmt.Printf("checking %d completed matches\n\n", len(completed))
	fmt.Printf("%-8s  %-32s  %-9s  %-3s  %-12s  %s\n",
		"ID", "matchup", "att", "evt", "displayable", "types")
	fmt.Println(strings.Repeat("-", 120))

	missingAttendance := 0
	parseErrors := 0
	for _, m := range completed {
		summary, err := espn.FetchSummary(m.ID)
		matchup := fmt.Sprintf("%s vs %s", m.HomeTeam, m.AwayTeam)
		if len(matchup) > 32 {
			matchup = matchup[:32]
		}
		if err != nil {
			fmt.Printf("%-8s  %-32s  ERROR: %v\n", m.ID, matchup, err)
			parseErrors++
			continue
		}
		if summary.Attendance == 0 {
			missingAttendance++
		}
		typeSet := map[string]bool{}
		displayable := 0
		for _, e := range summary.Events {
			typeSet[e.TypeID] = true
			if !nonDisplayable[e.TypeID] {
				displayable++
			}
		}
		types := make([]string, 0, len(typeSet))
		for t := range typeSet {
			types = append(types, t)
		}
		sort.Strings(types)
		fmt.Printf("%-8s  %-32s  %-9d  %-3d  %-12d  %v\n",
			m.ID, matchup, summary.Attendance, len(summary.Events), displayable, types)
	}

	fmt.Println()
	fmt.Printf("summary: %d matches checked, %d missing attendance, %d parse errors\n",
		len(completed), missingAttendance, parseErrors)
}
