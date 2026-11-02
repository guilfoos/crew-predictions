package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/guilfoos/crew-predictions/internal/models"
	"github.com/guilfoos/crew-predictions/internal/poll"
	"github.com/guilfoos/crew-predictions/internal/repository"
	"github.com/guilfoos/crew-predictions/internal/tasks"
)

// pollChainInterval is how far in the future the next chain task is scheduled
// when a match remains in a non-terminal state. Kept short so the live clock
// stays current.
const pollChainInterval = 2 * time.Minute

// pollChainMaxAge bounds how long a chain may run for one match. Longest
// legitimate soccer match (90 + ET 30 + PKs + buffer) is well under 4h; if
// we're still polling past this threshold, ESPN data has likely gone weird
// and the chain should bail out so the next 4am/12pm/6pm refresh can
// re-evaluate. The bail-out writes Match.AbandonedAt for diagnostic visibility.
const pollChainMaxAge = 4 * time.Hour

// terminalStatuses mirrors internal/poll.terminalStatuses plus the "won't
// resume today" states. Chain ends on any of these; the next 4am/12pm/6pm
// refresh picks up rescheduled kickoffs and seeds a fresh chain if needed.
var terminalStatuses = map[string]bool{
	"STATUS_FULL_TIME": true,
	"STATUS_FINAL_AET": true,
	"STATUS_FINAL_PEN": true,
	"STATUS_POSTPONED": true,
	"STATUS_CANCELED":  true,
	"STATUS_CANCELLED": true, // ESPN inconsistency between US/UK spelling
	"STATUS_ABANDONED": true,
	"STATUS_FORFEIT":   true,
}

type PollScoresHandler struct {
	matchStore  repository.MatchStore
	resultStore repository.ResultStore
	fetcher     func() ([]models.Match, error)
	recalcFn    func(context.Context)
	enqueuer    tasks.Enqueuer // nil = no chain continuation (legacy callers, tests)
}

func NewPollScoresHandler(matchStore repository.MatchStore, resultStore repository.ResultStore, fetcher func() ([]models.Match, error), recalcFn func(context.Context)) *PollScoresHandler {
	return &PollScoresHandler{matchStore: matchStore, resultStore: resultStore, fetcher: fetcher, recalcFn: recalcFn}
}

// WithEnqueuer attaches a chain-task enqueuer. When set, Poll consults the
// just-polled match for the matchID query param and schedules a follow-up
// task ~2 min out if the match is still in a non-terminal state. Without an
// enqueuer the handler retains its legacy "poll all matches, no chain"
// behavior (suitable for ad-hoc admin triggers and tests).
func (h *PollScoresHandler) WithEnqueuer(e tasks.Enqueuer) *PollScoresHandler {
	h.enqueuer = e
	return h
}

func (h *PollScoresHandler) Poll(w http.ResponseWriter, r *http.Request) {
	if err := poll.PollOnce(r.Context(), h.matchStore, h.resultStore, h.fetcher); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.recalcFn(r.Context())

	if matchID := r.URL.Query().Get("matchID"); matchID != "" && h.enqueuer != nil {
		h.maybeEnqueueNext(r.Context(), matchID)
	}

	w.WriteHeader(http.StatusNoContent)
}

// maybeEnqueueNext consults the freshly-polled store for the named match and
// schedules a follow-up task at now + pollChainInterval if the match is
// still in a non-terminal state. A missing match, a terminal status, or
// a pollChainMaxAge-exceeded match ends the chain (no enqueue). The
// max-age path additionally marks the match Abandoned for diagnostic
// visibility.
func (h *PollScoresHandler) maybeEnqueueNext(ctx context.Context, matchID string) {
	matches, err := h.matchStore.GetAll()
	if err != nil {
		slog.Error("poll_scores: chain continuation read failed", "matchID", matchID, "error", err)
		return
	}
	now := time.Now().UTC()
	for i, m := range matches {
		if m.ID != matchID {
			continue
		}
		if terminalStatuses[m.Status] {
			return
		}
		if !m.Kickoff.IsZero() && now.Sub(m.Kickoff) > pollChainMaxAge {
			matches[i].AbandonedAt = now
			if err := h.matchStore.SaveAll(matches); err != nil {
				slog.Error("poll_scores: AbandonedAt persist failed", "matchID", matchID, "error", err)
			}
			slog.Warn("poll_scores: chain safety bailout", "matchID", matchID, "kickoff", m.Kickoff, "ageSeconds", int(now.Sub(m.Kickoff).Seconds()))
			return
		}
		runAt := now.Add(pollChainInterval)
		if err := h.enqueuer.EnqueuePoll(ctx, matchID, runAt); err != nil {
			slog.Error("poll_scores: chain enqueue failed", "matchID", matchID, "error", err)
		}
		return
	}
}
