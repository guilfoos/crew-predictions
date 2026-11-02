package poll

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/guilfoos/crew-predictions/internal/models"
	"github.com/guilfoos/crew-predictions/internal/repository"
)

type MatchPoller struct {
	matchStore     repository.MatchStore
	resultStore    repository.ResultStore
	fetcher        func() ([]models.Match, error)
	summaryFetcher func(matchID string) (models.MatchSummary, error)
	onResultSaved  func(ctx context.Context)

	mu        sync.Mutex
	scheduled map[string]bool
	active    map[string]bool
	timerFunc func(time.Duration, func())
}

// SetSummaryFetcher registers a function used to retrieve event/attendance
// data for active matches. When set, Tick will call it once per active match
// per tick and persist the returned events back to the match store.
func (p *MatchPoller) SetSummaryFetcher(fn func(matchID string) (models.MatchSummary, error)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.summaryFetcher = fn
}

// SetOnResultSaved registers a callback invoked after each result is saved during Tick.
func (p *MatchPoller) SetOnResultSaved(fn func(ctx context.Context)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onResultSaved = fn
}

func NewMatchPoller(
	matchStore repository.MatchStore,
	resultStore repository.ResultStore,
	fetcher func() ([]models.Match, error),
	timerFunc func(time.Duration, func()),
) *MatchPoller {
	return &MatchPoller{
		matchStore:  matchStore,
		resultStore: resultStore,
		fetcher:     fetcher,
		timerFunc:   timerFunc,
		scheduled:   make(map[string]bool),
		active:      make(map[string]bool),
	}
}

// SetTimerFunc replaces the timer function; used in tests to switch from
// immediate firing to delay-capturing between Schedule calls.
func (p *MatchPoller) SetTimerFunc(f func(time.Duration, func())) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.timerFunc = f
}

type scheduledMatch struct {
	matchID string
	delay   time.Duration
}

func (p *MatchPoller) Schedule(matches []models.Match) {
	p.mu.Lock()
	var toSchedule []scheduledMatch
	for _, m := range matches {
		if p.scheduled[m.ID] || terminalStatuses[m.Status] {
			continue
		}
		p.scheduled[m.ID] = true
		delay := time.Until(m.Kickoff)
		if delay < 0 {
			delay = 0
		}
		toSchedule = append(toSchedule, scheduledMatch{m.ID, delay})
	}
	p.mu.Unlock()

	for _, s := range toSchedule {
		matchID := s.matchID
		delay := s.delay
		slog.Info("poller: match scheduled", "matchID", matchID, "delaySeconds", int(delay.Seconds()))
		p.timerFunc(delay, func() {
			p.mu.Lock()
			p.active[matchID] = true
			p.mu.Unlock()
			slog.Info("poller: kickoff reached, entering active polling", "matchID", matchID)
		})
	}
}

// Reset reconciles poller state to a fresh match list. Soft semantics:
// in-progress matches that re-appear in the new list keep their active
// status (no race window where polling drops them) and are NOT re-scheduled
// (the timer would re-fire immediately, duplicating polls). Matches that
// are absent from the new list, or present with terminal status, drop from
// active/scheduled. Net effect during refresh-while-in-progress: an
// undisturbed continuation for live matches, plus seeding for any newly
// scheduled pre matches.
func (p *MatchPoller) Reset(matches []models.Match) {
	keep := make(map[string]bool, len(matches))
	for _, m := range matches {
		if !terminalStatuses[m.Status] {
			keep[m.ID] = true
		}
	}

	p.mu.Lock()
	for id := range p.active {
		if !keep[id] {
			delete(p.active, id)
		}
	}
	for id := range p.scheduled {
		if !keep[id] {
			delete(p.scheduled, id)
		}
	}
	p.mu.Unlock()

	p.Schedule(matches)
}

// Tick fetches current match data, updates the store, and records results for
// any active matches that have reached a terminal status. It is a no-op when
// no matches are active.
func (p *MatchPoller) Tick(ctx context.Context) {
	p.mu.Lock()
	if len(p.active) == 0 {
		p.mu.Unlock()
		return
	}
	p.mu.Unlock()

	p.mu.Lock()
	activeCount := len(p.active)
	p.mu.Unlock()
	slog.Info("poller: tick", "activeMatches", activeCount)

	matches, err := p.fetcher()
	if err != nil {
		slog.Error("poller: ESPN fetch failed", "error", err)
		return
	}

	// Preserve chain-tracking fields (LastPollAt, ChainSeededFor, AbandonedAt)
	// from the existing store — fresh ESPN data carries none of them. Without
	// this, every Tick would wipe what the chain (/admin/poll-scores) and the
	// refresh handler set.
	matches = MergeChainFields(p.matchStore, matches)
	// Tick IS polling — stamp LastPollAt now so the in-process path also
	// counts as recent activity for the refresh's chain-liveness check.
	now := time.Now().UTC()
	for i := range matches {
		matches[i].LastPollAt = now
	}

	byID := make(map[string]models.Match, len(matches))
	for _, m := range matches {
		byID[m.ID] = m
	}

	p.mu.Lock()
	summaryFetcher := p.summaryFetcher
	activeIDs := make([]string, 0, len(p.active))
	for id := range p.active {
		activeIDs = append(activeIDs, id)
	}
	p.mu.Unlock()

	// Enrich active matches with /summary data (events, attendance, refs, logos)
	// before persisting. One ESPN call per active match per tick — typically
	// one match at a time for Crew predictions.
	if summaryFetcher != nil {
		for _, id := range activeIDs {
			m, ok := byID[id]
			if !ok {
				continue
			}
			summary, err := summaryFetcher(id)
			if err != nil {
				slog.Warn("poller: summary fetch failed", "matchID", id, "error", err)
				continue
			}
			if summary.Attendance > 0 {
				m.Attendance = summary.Attendance
			}
			if summary.HomeLogo != "" {
				m.HomeLogo = summary.HomeLogo
			}
			if summary.AwayLogo != "" {
				m.AwayLogo = summary.AwayLogo
			}
			if summary.Referee != "" {
				m.Referee = summary.Referee
			}
			if len(summary.Events) > 0 {
				m.Events = summary.Events
			}
			byID[id] = m
		}
		// Rebuild matches slice with enriched copies.
		for i, m := range matches {
			if enriched, ok := byID[m.ID]; ok {
				matches[i] = enriched
			}
		}
	}

	if err := p.matchStore.SaveAll(matches); err != nil {
		slog.Error("poller: store update failed", "error", err)
		return
	}

	p.mu.Lock()
	var toSave []models.Match
	for matchID := range p.active {
		m, ok := byID[matchID]
		if !ok {
			continue
		}
		if terminalStatuses[m.Status] {
			toSave = append(toSave, m)
			delete(p.active, matchID)
			delete(p.scheduled, matchID)
		} else {
			slog.Info("poller: match still in progress", "matchID", matchID, "status", m.Status, "homeScore", m.HomeScore, "awayScore", m.AwayScore)
		}
	}
	p.mu.Unlock()

	savedAny := false
	for _, m := range toSave {
		slog.Info("poller: match finished, saving result", "matchID", m.ID, "status", m.Status, "homeScore", m.HomeScore, "awayScore", m.AwayScore)
		if err := saveResult(ctx, p.resultStore, m); err != nil {
			slog.Error("poller: save result failed", "matchID", m.ID, "error", err)
		} else {
			savedAny = true
		}
	}
	p.mu.Lock()
	fn := p.onResultSaved
	p.mu.Unlock()
	if savedAny && fn != nil {
		fn(ctx)
	}
}

// Run starts the polling loop. Call in a goroutine; cancel ctx to stop.
func (p *MatchPoller) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			p.Tick(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// Backfill saves results for any terminal-status matches in the provided list
// without re-fetching from ESPN. Called after a daily refresh to catch matches
// that finished while no active poller was running for them.
func (p *MatchPoller) Backfill(ctx context.Context, matches []models.Match) {
	for _, m := range matches {
		if !terminalStatuses[m.Status] {
			continue
		}
		if err := saveResult(ctx, p.resultStore, m); err != nil {
			slog.Error("poller: backfill result failed", "matchID", m.ID, "error", err)
		}
	}
}

// ActiveCount returns the number of matches currently being polled (for testing).
func (p *MatchPoller) ActiveCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.active)
}
