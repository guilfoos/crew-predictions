package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/guilfoos/crew-predictions/internal/models"
	"github.com/guilfoos/crew-predictions/internal/repository"
	"github.com/guilfoos/crew-predictions/internal/tasks"
)

// refreshSeedWindow bounds how far in advance the refresh seeds chain tasks.
// Tasks are scheduled at kickoff-5min; with three daily refreshes (4am/12pm/
// 6pm ET, every 6h apart) this 8h window guarantees every match is covered
// by at least one refresh before kickoff.
const refreshSeedWindow = 8 * time.Hour

// refreshChainLivenessThreshold is how recent LastPollAt must be for the
// refresh to consider an in-progress match's chain "alive" and skip the
// revival enqueue. Slightly larger than pollChainInterval to allow normal
// poll jitter without false revivals.
const refreshChainLivenessThreshold = 5 * time.Minute

type RefreshMatchesHandler struct {
	store     repository.MatchStore
	fetcher   func() ([]models.Match, error)
	onRefresh func([]models.Match)
	enqueuer  tasks.Enqueuer
}

func NewRefreshMatchesHandler(store repository.MatchStore, fetcher func() ([]models.Match, error), onRefresh func([]models.Match)) *RefreshMatchesHandler {
	return &RefreshMatchesHandler{store: store, fetcher: fetcher, onRefresh: onRefresh}
}

// WithEnqueuer attaches a Cloud Tasks enqueuer. When set, Refresh applies
// the state-rule decision table (docs/match-polling-architecture.md) to
// seed chain tasks for upcoming matches and revive dead chains for
// in-progress matches. Without an enqueuer the refresh behaves like the
// legacy "fetch + save + onRefresh callback" path.
func (h *RefreshMatchesHandler) WithEnqueuer(e tasks.Enqueuer) *RefreshMatchesHandler {
	h.enqueuer = e
	return h
}

func (h *RefreshMatchesHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	fresh, err := h.fetcher()
	if err != nil {
		http.Error(w, "couldn't fetch matches", http.StatusInternalServerError)
		return
	}

	// Merge chain-tracking fields from the existing store into the fresh
	// ESPN data. ESPN doesn't know about LastPollAt / ChainSeededFor /
	// AbandonedAt — without this, every refresh would wipe them.
	merged := h.mergeChainFields(fresh)

	// Apply the state-rule seeding decisions in-place (may update
	// ChainSeededFor on `merged`) and collect enqueue calls to make
	// after the save. Doing enqueues after SaveAll keeps the store in a
	// consistent state if Cloud Tasks ever rejects an enqueue.
	enqueues := h.planChainSeeds(merged)

	if err := h.store.SaveAll(merged); err != nil {
		http.Error(w, "couldn't save matches", http.StatusInternalServerError)
		return
	}

	for _, e := range enqueues {
		if err := h.enqueuer.EnqueuePoll(r.Context(), e.matchID, e.runAt); err != nil {
			slog.Error("refresh_matches: enqueue failed", "matchID", e.matchID, "error", err)
		}
	}

	if h.onRefresh != nil {
		h.onRefresh(merged)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *RefreshMatchesHandler) mergeChainFields(fresh []models.Match) []models.Match {
	existing, err := h.store.GetAll()
	if err != nil {
		// Soft-fail: log and fall back to fresh-only (no merge). Worst case:
		// LastPollAt resets, refresh seeds a duplicate chain. Idempotent at the
		// poll level (both chains converge to the same final state).
		slog.Error("refresh_matches: existing store read failed", "error", err)
		return fresh
	}
	byID := make(map[string]models.Match, len(existing))
	for _, m := range existing {
		byID[m.ID] = m
	}
	for i, m := range fresh {
		if prev, ok := byID[m.ID]; ok {
			fresh[i].LastPollAt = prev.LastPollAt
			fresh[i].ChainSeededFor = prev.ChainSeededFor
			fresh[i].AbandonedAt = prev.AbandonedAt
		}
	}
	return fresh
}

type plannedEnqueue struct {
	matchID string
	runAt   time.Time
}

// planChainSeeds applies the state-rule table from
// docs/match-polling-architecture.md. Mutates merged matches to set
// ChainSeededFor when seeding a fresh chain; returns the list of enqueue
// calls to make after SaveAll.
func (h *RefreshMatchesHandler) planChainSeeds(merged []models.Match) []plannedEnqueue {
	if h.enqueuer == nil {
		return nil
	}
	now := time.Now().UTC()
	var out []plannedEnqueue
	for i, m := range merged {
		switch {
		case m.State == "pre":
			if m.Kickoff.IsZero() || m.Kickoff.Sub(now) > refreshSeedWindow {
				continue
			}
			if m.ChainSeededFor.Equal(m.Kickoff) {
				continue // already seeded for this kickoff
			}
			merged[i].ChainSeededFor = m.Kickoff
			out = append(out, plannedEnqueue{
				matchID: m.ID,
				runAt:   m.Kickoff.Add(-5 * time.Minute),
			})
		case m.State == "in":
			if now.Sub(m.LastPollAt) < refreshChainLivenessThreshold {
				continue // chain is alive, don't disturb it
			}
			out = append(out, plannedEnqueue{
				matchID: m.ID,
				runAt:   now,
			})
		default:
			// state == "post" or unknown → no action
		}
	}
	return out
}
