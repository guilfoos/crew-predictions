package poll_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/mcornell/crew-predictions/internal/models"
	"github.com/mcornell/crew-predictions/internal/poll"
	"github.com/mcornell/crew-predictions/internal/repository"
)

// immediateTimer fires the callback synchronously, ignoring the delay.
func immediateTimer(_ time.Duration, f func()) { f() }

// capturingTimer records delays without firing callbacks.
func capturingTimer(delays *[]time.Duration) func(time.Duration, func()) {
	return func(d time.Duration, _ func()) {
		*delays = append(*delays, d)
	}
}

func newPoller(matchStore repository.MatchStore, resultStore repository.ResultStore, matches []models.Match, timer func(time.Duration, func())) *poll.MatchPoller {
	fetcher := func() ([]models.Match, error) { return matches, nil }
	return poll.NewMatchPoller(matchStore, resultStore, fetcher, timer)
}

func TestMatchPoller_Tick_FetchesSummaryAndPersistsEventsForLiveMatch(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	matchStore.Seed([]models.Match{{
		ID: "m-live", HomeTeam: "Crew", AwayTeam: "FC Dallas",
		Kickoff: time.Now().Add(-30 * time.Minute), State: "in", Status: "STATUS_IN_PROGRESS",
	}})

	scoreboardMatch := models.Match{
		ID: "m-live", HomeTeam: "Crew", AwayTeam: "FC Dallas",
		Kickoff: time.Now().Add(-30 * time.Minute), State: "in", Status: "STATUS_IN_PROGRESS",
		HomeScore: "1", AwayScore: "0",
	}
	scoreboardFetcher := func() ([]models.Match, error) { return []models.Match{scoreboardMatch}, nil }
	summaryCalls := 0
	summaryFetcher := func(id string) (models.MatchSummary, error) {
		summaryCalls++
		return models.MatchSummary{
			Events: []models.MatchEvent{
				{Clock: "23'", TypeID: "goal", Team: "Crew", Players: []string{"Hugo Picard"}},
			},
		}, nil
	}

	p := poll.NewMatchPoller(matchStore, repository.NewMemoryResultStore(), scoreboardFetcher, immediateTimer)
	p.SetSummaryFetcher(summaryFetcher)
	p.Schedule([]models.Match{scoreboardMatch}) // activates the match
	p.Tick(context.Background())

	if summaryCalls != 1 {
		t.Errorf("expected summaryFetcher called exactly once, got %d", summaryCalls)
	}
	stored, _ := matchStore.GetAll()
	if len(stored) == 0 || len(stored[0].Events) != 1 || stored[0].Events[0].Players[0] != "Hugo Picard" {
		t.Errorf("expected events persisted to store; got %+v", stored)
	}
}

func TestMatchPoller_Tick_ContinuesWhenSummaryFetcherFails(t *testing.T) {
	// A failing summaryFetcher should not block the scoreboard data from
	// being persisted. Match score still updates; events stay empty.
	matchStore := repository.NewMemoryMatchStore()
	scoreboardMatch := models.Match{
		ID: "m-sumerr", HomeTeam: "Crew", AwayTeam: "FC Dallas",
		Kickoff: time.Now().Add(-30 * time.Minute), State: "in", Status: "STATUS_IN_PROGRESS",
		HomeScore: "2", AwayScore: "1",
	}
	matchStore.Seed([]models.Match{scoreboardMatch})
	scoreboardFetcher := func() ([]models.Match, error) { return []models.Match{scoreboardMatch}, nil }
	summaryFetcher := func(_ string) (models.MatchSummary, error) {
		return models.MatchSummary{}, fmt.Errorf("simulated summary failure")
	}

	p := poll.NewMatchPoller(matchStore, repository.NewMemoryResultStore(), scoreboardFetcher, immediateTimer)
	p.SetSummaryFetcher(summaryFetcher)
	p.Schedule([]models.Match{scoreboardMatch})
	p.Tick(context.Background())

	stored, _ := matchStore.GetAll()
	if len(stored) != 1 {
		t.Fatalf("expected 1 match in store, got %d", len(stored))
	}
	if stored[0].HomeScore != "2" || stored[0].AwayScore != "1" {
		t.Errorf("expected scoreboard data persisted (2-1), got %s-%s", stored[0].HomeScore, stored[0].AwayScore)
	}
	if len(stored[0].Events) != 0 {
		t.Errorf("expected no events when summary fetch failed, got %d", len(stored[0].Events))
	}
}

func TestMatchPoller_Tick_SkipsSummaryForActiveIDsAbsentFromScoreboard(t *testing.T) {
	// If an active match isn't in the scoreboard fetch result (rare, but
	// possible during ESPN reshuffles), the !ok guard should skip the
	// summary call entirely rather than enriching a zero-value match.
	matchStore := repository.NewMemoryMatchStore()
	staleMatch := models.Match{
		ID: "m-stale", HomeTeam: "Crew", AwayTeam: "FC Dallas",
		Kickoff: time.Now().Add(-30 * time.Minute), State: "in", Status: "STATUS_IN_PROGRESS",
	}
	matchStore.Seed([]models.Match{staleMatch})
	// Scoreboard returns empty — m-stale is no longer in ESPN's response.
	scoreboardFetcher := func() ([]models.Match, error) { return []models.Match{}, nil }
	summaryCalls := 0
	summaryFetcher := func(_ string) (models.MatchSummary, error) {
		summaryCalls++
		return models.MatchSummary{}, nil
	}

	p := poll.NewMatchPoller(matchStore, repository.NewMemoryResultStore(), scoreboardFetcher, immediateTimer)
	p.SetSummaryFetcher(summaryFetcher)
	p.Schedule([]models.Match{staleMatch})
	p.Tick(context.Background())

	if summaryCalls != 0 {
		t.Errorf("expected summaryFetcher NOT called for IDs absent from scoreboard, was called %d times", summaryCalls)
	}
}

func TestMatchPoller_Tick_DoesNotCallSummaryWhenNoneConfigured(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	scoreboardMatch := models.Match{
		ID: "m-no-sum", HomeTeam: "Crew", AwayTeam: "FC Dallas",
		Kickoff: time.Now().Add(-30 * time.Minute), State: "in", Status: "STATUS_IN_PROGRESS",
	}
	matchStore.Seed([]models.Match{scoreboardMatch})
	fetcher := func() ([]models.Match, error) { return []models.Match{scoreboardMatch}, nil }
	p := poll.NewMatchPoller(matchStore, repository.NewMemoryResultStore(), fetcher, immediateTimer)
	p.Schedule([]models.Match{scoreboardMatch})
	// Should not panic with nil summaryFetcher.
	p.Tick(context.Background())
}

func TestMatchPoller_ScheduleSetsPositiveDelayForFutureKickoff(t *testing.T) {
	var delays []time.Duration
	p := newPoller(repository.NewMemoryMatchStore(), repository.NewMemoryResultStore(), nil, capturingTimer(&delays))

	future := time.Now().Add(2 * time.Hour)
	p.Schedule([]models.Match{{ID: "m1", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas", Kickoff: future, Status: "STATUS_SCHEDULED"}})

	if len(delays) != 1 {
		t.Fatalf("expected 1 timer, got %d", len(delays))
	}
	if delays[0] <= 0 {
		t.Errorf("expected positive delay for future kickoff, got %v", delays[0])
	}
}

func TestMatchPoller_ScheduleUsesZeroDelayForPastKickoff(t *testing.T) {
	var delays []time.Duration
	p := newPoller(repository.NewMemoryMatchStore(), repository.NewMemoryResultStore(), nil, capturingTimer(&delays))

	past := time.Now().Add(-1 * time.Hour)
	p.Schedule([]models.Match{{ID: "m1", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas", Kickoff: past, Status: "STATUS_SCHEDULED", State: "in"}})

	if len(delays) != 1 {
		t.Fatalf("expected 1 timer, got %d", len(delays))
	}
	if delays[0] != 0 {
		t.Errorf("expected zero delay for past kickoff, got %v", delays[0])
	}
}

func TestMatchPoller_ScheduleSkipsTerminalMatch(t *testing.T) {
	var delays []time.Duration
	p := newPoller(repository.NewMemoryMatchStore(), repository.NewMemoryResultStore(), nil, capturingTimer(&delays))

	p.Schedule([]models.Match{{ID: "m1", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas", Kickoff: time.Now().Add(-1 * time.Hour), Status: "STATUS_FULL_TIME", State: "post"}})

	if len(delays) != 0 {
		t.Errorf("expected no timer for terminal match, got %d", len(delays))
	}
}

func TestMatchPoller_ScheduleSkipsAlreadyScheduled(t *testing.T) {
	var delays []time.Duration
	p := newPoller(repository.NewMemoryMatchStore(), repository.NewMemoryResultStore(), nil, capturingTimer(&delays))

	m := models.Match{ID: "m1", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas", Kickoff: time.Now().Add(1 * time.Hour), Status: "STATUS_SCHEDULED"}
	p.Schedule([]models.Match{m})
	p.Schedule([]models.Match{m})

	if len(delays) != 1 {
		t.Errorf("expected 1 timer after two Schedule calls for same match, got %d", len(delays))
	}
}

func TestMatchPoller_Tick_EnrichesAttendanceLogosAndRefereeFromSummary(t *testing.T) {
	// Covers the per-field summary enrichment branches in Tick: Attendance,
	// HomeLogo, AwayLogo, Referee. Existing tests only populate Events.
	matchStore := repository.NewMemoryMatchStore()
	scoreboardMatch := models.Match{
		ID: "m-rich", HomeTeam: "Crew", AwayTeam: "FC Dallas",
		Kickoff: time.Now().Add(-30 * time.Minute), State: "in", Status: "STATUS_IN_PROGRESS",
	}
	matchStore.Seed([]models.Match{scoreboardMatch})
	scoreboardFetcher := func() ([]models.Match, error) { return []models.Match{scoreboardMatch}, nil }
	summaryFetcher := func(_ string) (models.MatchSummary, error) {
		return models.MatchSummary{
			Attendance: 21657,
			HomeLogo:   "https://espncdn.com/i/teamlogos/soccer/500/9723.png",
			AwayLogo:   "https://espncdn.com/i/teamlogos/soccer/500/152.png",
			Referee:    "Ted Unkel",
		}, nil
	}

	p := poll.NewMatchPoller(matchStore, repository.NewMemoryResultStore(), scoreboardFetcher, immediateTimer)
	p.SetSummaryFetcher(summaryFetcher)
	p.Schedule([]models.Match{scoreboardMatch})
	p.Tick(context.Background())

	stored, _ := matchStore.GetAll()
	if len(stored) != 1 {
		t.Fatalf("expected 1 match in store, got %d", len(stored))
	}
	got := stored[0]
	if got.Attendance != 21657 {
		t.Errorf("Attendance: got %d, want 21657", got.Attendance)
	}
	if got.HomeLogo == "" || got.AwayLogo == "" {
		t.Errorf("logos not enriched: home=%q away=%q", got.HomeLogo, got.AwayLogo)
	}
	if got.Referee != "Ted Unkel" {
		t.Errorf("Referee: got %q, want %q", got.Referee, "Ted Unkel")
	}
}

func TestMatchPoller_Tick_NoopWhenNoActiveMatches(t *testing.T) {
	calls := 0
	matchStore := repository.NewMemoryMatchStore()
	resultStore := repository.NewMemoryResultStore()
	fetcher := func() ([]models.Match, error) { calls++; return nil, nil }
	p := poll.NewMatchPoller(matchStore, resultStore, fetcher, capturingTimer(new([]time.Duration)))

	p.Tick(context.Background())

	if calls != 0 {
		t.Errorf("expected no fetcher calls with no active matches, got %d", calls)
	}
}

func TestMatchPoller_Tick_SavesResultAndDeactivatesTerminalMatch(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	resultStore := repository.NewMemoryResultStore()

	// Fetcher simulates ESPN returning a now-terminal match
	terminalMatch := models.Match{
		ID: "m-done", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas",
		Status: "STATUS_FULL_TIME", State: "post", HomeScore: "2", AwayScore: "1",
		Kickoff: time.Now().Add(-2 * time.Hour),
	}
	p := newPoller(matchStore, resultStore, []models.Match{terminalMatch}, immediateTimer)

	// Schedule with pre-match state (as it would have been at kickoff)
	preMatch := models.Match{
		ID: "m-done", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas",
		Status: "STATUS_SCHEDULED", State: "pre",
		Kickoff: time.Now().Add(-2 * time.Hour),
	}
	p.Schedule([]models.Match{preMatch})

	p.Tick(context.Background())

	result, err := resultStore.GetResult(context.Background(), "m-done")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil || result.HomeGoals != 2 || result.AwayGoals != 1 {
		t.Errorf("expected result 2-1, got %+v", result)
	}
	if p.ActiveCount() != 0 {
		t.Errorf("expected match removed from active after terminal, got %d active", p.ActiveCount())
	}
}

func TestMatchPoller_Tick_KeepsActiveForLiveMatch(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	resultStore := repository.NewMemoryResultStore()

	liveMatch := models.Match{
		ID: "m-live", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas",
		Status: "STATUS_IN_PROGRESS", State: "in", HomeScore: "1", AwayScore: "0",
		Kickoff: time.Now().Add(-1 * time.Hour),
	}
	p := newPoller(matchStore, resultStore, []models.Match{liveMatch}, immediateTimer)
	p.Schedule([]models.Match{liveMatch})

	p.Tick(context.Background())

	if p.ActiveCount() != 1 {
		t.Errorf("expected match to remain active for live match, got %d active", p.ActiveCount())
	}
}

func TestMatchPoller_Tick_KeepsActiveForUnknownStatus(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	resultStore := repository.NewMemoryResultStore()

	unknownMatch := models.Match{
		ID: "m-weird", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas",
		Status: "STATUS_POSTPONED", State: "pre",
		Kickoff: time.Now().Add(-1 * time.Hour),
	}
	p := newPoller(matchStore, resultStore, []models.Match{unknownMatch}, immediateTimer)
	p.Schedule([]models.Match{unknownMatch})

	p.Tick(context.Background())

	if p.ActiveCount() != 1 {
		t.Errorf("expected match to remain active for unknown status (4am reset will clean up), got %d active", p.ActiveCount())
	}
}

func TestMatchPoller_Tick_NoopWhenFetcherFails(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	resultStore := repository.NewMemoryResultStore()

	callCount := 0
	fetcher := func() ([]models.Match, error) {
		callCount++
		return nil, fmt.Errorf("ESPN down")
	}
	p := poll.NewMatchPoller(matchStore, resultStore, fetcher, immediateTimer)

	pastMatch := models.Match{
		ID: "m1", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas",
		Status: "STATUS_SCHEDULED", State: "in",
		Kickoff: time.Now().Add(-1 * time.Hour),
	}
	p.Schedule([]models.Match{pastMatch})
	p.Tick(context.Background())

	result, _ := resultStore.GetResult(context.Background(), "m1")
	if result != nil {
		t.Errorf("expected no result saved when fetcher fails, got %+v", result)
	}
}

func TestMatchPoller_Run_StopsOnContextCancel(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	resultStore := repository.NewMemoryResultStore()
	fetcher := func() ([]models.Match, error) { return nil, nil }
	p := poll.NewMatchPoller(matchStore, resultStore, fetcher, func(time.Duration, func()) {})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		p.Run(ctx, 10*time.Millisecond)
		close(done)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Error("Run did not stop after context cancel within 1 second")
	}
}

func TestMatchPoller_Backfill_SavesResultForTerminalMatch(t *testing.T) {
	resultStore := repository.NewMemoryResultStore()
	p := newPoller(repository.NewMemoryMatchStore(), resultStore, nil, capturingTimer(new([]time.Duration)))

	p.Backfill(context.Background(), []models.Match{
		{ID: "m-done", HomeTeam: "Columbus Crew", AwayTeam: "LA Galaxy",
			Status: "STATUS_FULL_TIME", HomeScore: "2", AwayScore: "1"},
	})

	result, err := resultStore.GetResult(context.Background(), "m-done")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil || result.HomeGoals != 2 || result.AwayGoals != 1 {
		t.Errorf("expected result 2-1, got %+v", result)
	}
}

func TestMatchPoller_Backfill_SkipsNonTerminalMatch(t *testing.T) {
	resultStore := repository.NewMemoryResultStore()
	p := newPoller(repository.NewMemoryMatchStore(), resultStore, nil, capturingTimer(new([]time.Duration)))

	p.Backfill(context.Background(), []models.Match{
		{ID: "m-live", HomeTeam: "Columbus Crew", AwayTeam: "LA Galaxy",
			Status: "STATUS_IN_PROGRESS", State: "in", HomeScore: "1", AwayScore: "0"},
	})

	result, _ := resultStore.GetResult(context.Background(), "m-live")
	if result != nil {
		t.Errorf("expected no result for non-terminal match, got %+v", result)
	}
}

func TestMatchPoller_Backfill_LogsErrorWhenSaveResultFails(t *testing.T) {
	p := newPoller(repository.NewMemoryMatchStore(), repository.NewErrorResultStore(), nil, capturingTimer(new([]time.Duration)))

	// Should not panic — error is logged
	p.Backfill(context.Background(), []models.Match{
		{ID: "m1", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas", Status: "STATUS_FULL_TIME", HomeScore: "2", AwayScore: "1"},
	})
}

func TestMatchPoller_Tick_LogsErrorWhenSaveAllFails(t *testing.T) {
	resultStore := repository.NewMemoryResultStore()
	fetcher := func() ([]models.Match, error) {
		return []models.Match{{ID: "m1", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas", Status: "STATUS_IN_PROGRESS", HomeScore: "1", AwayScore: "0"}}, nil
	}
	p := poll.NewMatchPoller(&errMatchStore{}, resultStore, fetcher, immediateTimer)

	pastMatch := models.Match{ID: "m1", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas", Status: "STATUS_SCHEDULED", Kickoff: time.Now().Add(-1 * time.Hour)}
	p.Schedule([]models.Match{pastMatch})

	// Should not panic — error is logged
	p.Tick(context.Background())
}

func TestMatchPoller_Tick_LogsErrorWhenSaveResultFails(t *testing.T) {
	fetcher := func() ([]models.Match, error) {
		return []models.Match{{ID: "m1", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas", Status: "STATUS_FULL_TIME", HomeScore: "2", AwayScore: "1", Kickoff: time.Now().Add(-2 * time.Hour)}}, nil
	}
	p := poll.NewMatchPoller(repository.NewMemoryMatchStore(), repository.NewErrorResultStore(), fetcher, immediateTimer)

	preMatch := models.Match{ID: "m1", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas", Status: "STATUS_SCHEDULED", Kickoff: time.Now().Add(-2 * time.Hour)}
	p.Schedule([]models.Match{preMatch})

	// Should not panic — error is logged
	p.Tick(context.Background())
}

func TestMatchPoller_Tick_CallsOnResultSavedWhenMatchFinishes(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	resultStore := repository.NewMemoryResultStore()

	terminalMatch := models.Match{
		ID: "m-done", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas",
		Status: "STATUS_FULL_TIME", State: "post", HomeScore: "2", AwayScore: "1",
		Kickoff: time.Now().Add(-2 * time.Hour),
	}
	p := newPoller(matchStore, resultStore, []models.Match{terminalMatch}, immediateTimer)
	p.Schedule([]models.Match{{ID: "m-done", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas", Status: "STATUS_SCHEDULED", Kickoff: time.Now().Add(-2 * time.Hour)}})

	called := 0
	p.SetOnResultSaved(func(_ context.Context) { called++ })

	p.Tick(context.Background())

	if called != 1 {
		t.Errorf("expected onResultSaved called once, got %d", called)
	}
}

func TestMatchPoller_Tick_DoesNotCallOnResultSavedWhenMatchStillLive(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	resultStore := repository.NewMemoryResultStore()

	liveMatch := models.Match{
		ID: "m-live", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas",
		Status: "STATUS_IN_PROGRESS", State: "in", HomeScore: "1", AwayScore: "0",
		Kickoff: time.Now().Add(-1 * time.Hour),
	}
	p := newPoller(matchStore, resultStore, []models.Match{liveMatch}, immediateTimer)
	p.Schedule([]models.Match{liveMatch})

	called := 0
	p.SetOnResultSaved(func(_ context.Context) { called++ })

	p.Tick(context.Background())

	if called != 0 {
		t.Errorf("expected onResultSaved not called for live match, got %d", called)
	}
}

func TestMatchPoller_Reset_ClearsActiveAndReschedules(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	resultStore := repository.NewMemoryResultStore()

	liveMatch := models.Match{
		ID: "m-live", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas",
		Status: "STATUS_IN_PROGRESS", State: "in",
		Kickoff: time.Now().Add(-1 * time.Hour),
	}

	var delays []time.Duration
	p := newPoller(matchStore, resultStore, []models.Match{liveMatch}, immediateTimer)
	p.Schedule([]models.Match{liveMatch})

	if p.ActiveCount() != 1 {
		t.Fatalf("expected 1 active match before reset, got %d", p.ActiveCount())
	}

	// After reset, old active cleared; new schedule applied with capturing timer
	newMatch := models.Match{
		ID: "m-new", HomeTeam: "Columbus Crew", AwayTeam: "Portland Timbers",
		Status: "STATUS_SCHEDULED", State: "pre",
		Kickoff: time.Now().Add(3 * time.Hour),
	}
	p.SetTimerFunc(capturingTimer(&delays))
	p.Reset([]models.Match{newMatch})

	if p.ActiveCount() != 0 {
		t.Errorf("expected active cleared after reset, got %d", p.ActiveCount())
	}
	if len(delays) != 1 {
		t.Errorf("expected 1 new timer after reset, got %d", len(delays))
	}
}

func TestMatchPoller_Run_ExitsOnContextCancel(t *testing.T) {
	p := newPoller(repository.NewMemoryMatchStore(), repository.NewMemoryResultStore(), nil, capturingTimer(new([]time.Duration)))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		p.Run(ctx, 10*time.Millisecond)
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Error("Run did not exit after context cancellation")
	}
}

func TestMatchPoller_Reset_KeepsInProgressMatchActive(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	resultStore := repository.NewMemoryResultStore()

	live := models.Match{
		ID: "m-live", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas",
		Status: "STATUS_IN_PROGRESS", State: "in",
		Kickoff: time.Now().Add(-1 * time.Hour),
	}
	p := newPoller(matchStore, resultStore, []models.Match{live}, immediateTimer)
	p.Schedule([]models.Match{live})

	if p.ActiveCount() != 1 {
		t.Fatalf("precondition: expected 1 active match, got %d", p.ActiveCount())
	}

	// Refresh sees the same in-progress match. Switch to capturingTimer
	// so we can distinguish "stayed in active" from "re-fired immediately".
	delays := new([]time.Duration)
	p.SetTimerFunc(capturingTimer(delays))
	p.Reset([]models.Match{live})

	if p.ActiveCount() != 1 {
		t.Errorf("soft Reset: in-progress match should stay active, got ActiveCount=%d", p.ActiveCount())
	}
	if len(*delays) != 0 {
		t.Errorf("soft Reset: should not re-schedule in-progress match, got %d new timer(s)", len(*delays))
	}
}

func TestMatchPoller_Reset_DropsMatchNotInNewList(t *testing.T) {
	// If a previously-active match is no longer in the refresh result
	// (rare — e.g. ESPN dropped it), Reset should drop it from active.
	matchStore := repository.NewMemoryMatchStore()
	resultStore := repository.NewMemoryResultStore()

	live := models.Match{
		ID: "m-gone", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas",
		Status: "STATUS_IN_PROGRESS", State: "in",
		Kickoff: time.Now().Add(-1 * time.Hour),
	}
	p := newPoller(matchStore, resultStore, []models.Match{live}, immediateTimer)
	p.Schedule([]models.Match{live})

	if p.ActiveCount() != 1 {
		t.Fatalf("precondition: expected 1 active match, got %d", p.ActiveCount())
	}

	// Reset with empty list — m-gone should drop.
	p.SetTimerFunc(capturingTimer(new([]time.Duration)))
	p.Reset([]models.Match{})

	if p.ActiveCount() != 0 {
		t.Errorf("Reset with empty list: expected ActiveCount=0, got %d", p.ActiveCount())
	}
}

func TestMatchPoller_Reset_DropsTerminalMatch(t *testing.T) {
	// If a previously-in-progress match has reached terminal status in the
	// refresh result, Reset should drop it from active.
	matchStore := repository.NewMemoryMatchStore()
	resultStore := repository.NewMemoryResultStore()

	live := models.Match{
		ID: "m-ended", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas",
		Status: "STATUS_IN_PROGRESS", State: "in",
		Kickoff: time.Now().Add(-2 * time.Hour),
	}
	p := newPoller(matchStore, resultStore, []models.Match{live}, immediateTimer)
	p.Schedule([]models.Match{live})

	if p.ActiveCount() != 1 {
		t.Fatalf("precondition: expected 1 active match, got %d", p.ActiveCount())
	}

	// Reset with the same match but now finished — should drop from active.
	ended := live
	ended.Status = "STATUS_FULL_TIME"
	ended.State = "post"
	p.SetTimerFunc(capturingTimer(new([]time.Duration)))
	p.Reset([]models.Match{ended})

	if p.ActiveCount() != 0 {
		t.Errorf("terminal match in Reset: expected ActiveCount=0, got %d", p.ActiveCount())
	}
}

func TestMatchPoller_Tick_StampsLastPollAt(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	resultStore := repository.NewMemoryResultStore()

	live := models.Match{
		ID: "m-live", HomeTeam: "Columbus Crew", AwayTeam: "LAFC",
		Status: "STATUS_IN_PROGRESS", State: "in",
		Kickoff: time.Now().Add(-30 * time.Minute),
	}
	p := newPoller(matchStore, resultStore, []models.Match{live}, immediateTimer)
	p.Schedule([]models.Match{live})
	if p.ActiveCount() != 1 {
		t.Fatalf("precondition: expected 1 active, got %d", p.ActiveCount())
	}

	before := time.Now().UTC()
	p.Tick(context.Background())
	after := time.Now().UTC()

	stored, _ := matchStore.GetAll()
	if len(stored) != 1 {
		t.Fatalf("expected 1 stored match, got %d", len(stored))
	}
	if stored[0].LastPollAt.IsZero() {
		t.Error("Tick should stamp LastPollAt; got zero")
	}
	if stored[0].LastPollAt.Before(before) || stored[0].LastPollAt.After(after) {
		t.Errorf("Tick LastPollAt %v outside [%v, %v]", stored[0].LastPollAt, before, after)
	}
}

func TestMatchPoller_Tick_PreservesChainSeededForAndAbandonedAt(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	resultStore := repository.NewMemoryResultStore()

	seededFor := time.Date(2026, 5, 16, 23, 30, 0, 0, time.UTC)
	abandonedAt := time.Date(2026, 5, 14, 3, 30, 0, 0, time.UTC)
	// Seed with chain-tracking fields already set (as if /admin/refresh-matches
	// and a prior chain tick had run).
	matchStore.Seed([]models.Match{{
		ID: "m-live", HomeTeam: "Columbus Crew", AwayTeam: "LAFC",
		Status: "STATUS_IN_PROGRESS", State: "in",
		Kickoff:        seededFor,
		ChainSeededFor: seededFor,
		AbandonedAt:    abandonedAt,
	}})
	// Fetcher returns fresh ESPN data without the chain fields.
	live := models.Match{
		ID: "m-live", HomeTeam: "Columbus Crew", AwayTeam: "LAFC",
		Status: "STATUS_IN_PROGRESS", State: "in",
		Kickoff: seededFor,
	}
	p := newPoller(matchStore, resultStore, []models.Match{live}, immediateTimer)
	p.Schedule([]models.Match{live})

	p.Tick(context.Background())

	stored, _ := matchStore.GetAll()
	if !stored[0].ChainSeededFor.Equal(seededFor) {
		t.Errorf("Tick wiped ChainSeededFor: got %v, want %v", stored[0].ChainSeededFor, seededFor)
	}
	if !stored[0].AbandonedAt.Equal(abandonedAt) {
		t.Errorf("Tick wiped AbandonedAt: got %v, want %v", stored[0].AbandonedAt, abandonedAt)
	}
}
