package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/guilfoos/crew-predictions/internal/bot"
	"github.com/guilfoos/crew-predictions/internal/espn"
	"github.com/guilfoos/crew-predictions/internal/handlers"
	"github.com/guilfoos/crew-predictions/internal/models"
	"github.com/guilfoos/crew-predictions/internal/poll"
	"github.com/guilfoos/crew-predictions/internal/recalculator"
	"github.com/guilfoos/crew-predictions/internal/tasks"
)

// Deps is the bag of dependencies passed into route registration and
// background-job startup. Everything in here is constructed at server
// startup and lives for the process lifetime.
type Deps struct {
	Cfg            Config
	Stores         Stores
	Verifier       handlers.TokenVerifier
	SummaryFetcher func(matchID string) (models.MatchSummary, error)
	RefreshFetcher func() ([]models.Match, error)
	RecalcFn       func(ctx context.Context)
	TwoOneBot      *bot.TwoOneBot
	MatchPoller    *poll.MatchPoller // nil in test mode (no background polling)
	Enqueuer       tasks.Enqueuer    // nil in test mode without explicit fake; real Cloud Tasks in prod when configured
	enqueuerCloser func() error      // optional shutdown hook for the real Cloud Tasks client
}

// Close releases any process-lifetime resources held by Deps (Cloud Tasks
// gRPC client, etc.). Safe to call on a zero-value Deps or when no
// closers were registered. Intended for `defer deps.Close()` in main.
func (d Deps) Close() {
	if d.enqueuerCloser != nil {
		if err := d.enqueuerCloser(); err != nil {
			slog.Warn("deps.Close: enqueuer close failed", "error", err)
		}
	}
}

// buildDeps wires the runtime dependencies that bridge stores and handlers:
// the summary fetcher (live ESPN vs. fixture-backed in test mode), the
// match-refresh fetcher (live ESPN vs. read-from-store in test mode), the
// recalculation closure that handlers invoke after persisting state, the
// 2-1 bot, and the background match poller (production only).
func buildDeps(cfg Config, stores Stores, verifier handlers.TokenVerifier) Deps {
	summaryFetcher := chooseSummaryFetcher(cfg)
	refreshFetcher := chooseRefreshFetcher(cfg, stores.Match)

	recalcFn := func(ctx context.Context) {
		if err := recalculator.Recalculate(ctx, stores.Prediction, stores.Result, stores.User, nil, recalculator.SeasonWindow{}, cfg.TargetTeam); err != nil {
			slog.Error("recalculate failed", "error", err)
		}
	}

	twoOneBot := bot.New(stores.Prediction, stores.User, cfg.TargetTeam)

	deps := Deps{
		Cfg:            cfg,
		Stores:         stores,
		Verifier:       verifier,
		SummaryFetcher: summaryFetcher,
		RefreshFetcher: refreshFetcher,
		RecalcFn:       recalcFn,
		TwoOneBot:      twoOneBot,
	}

	if !cfg.TestMode {
		deps.MatchPoller = poll.NewMatchPoller(
			stores.Match, stores.Result, refreshFetcher,
			func(d time.Duration, f func()) { time.AfterFunc(d, f) },
		)
		deps.MatchPoller.SetSummaryFetcher(summaryFetcher)
		deps.MatchPoller.SetOnResultSaved(recalcFn)

		// Real Cloud Tasks enqueuer when all routing env vars are set.
		// Missing vars → no chain enqueueing (falls back to in-process poller
		// only). Logged once at startup so an incomplete config is obvious.
		if e, closer := buildCloudTasksEnqueuer(cfg); e != nil {
			deps.Enqueuer = e
			deps.enqueuerCloser = closer
		}
	}

	return deps
}

// buildCloudTasksEnqueuer constructs the real Cloud Tasks enqueuer when
// every CLOUD_TASKS_* env var is present, or returns (nil, nil) so the
// server falls back to in-process polling. Returns a closer function so the
// caller can release the gRPC client on shutdown.
func buildCloudTasksEnqueuer(cfg Config) (tasks.Enqueuer, func() error) {
	if cfg.CloudTasksProject == "" || cfg.CloudTasksLocation == "" || cfg.CloudTasksQueue == "" || cfg.CloudTasksTarget == "" {
		slog.Info("cloud tasks: routing config incomplete, chain enqueueing disabled",
			"project", cfg.CloudTasksProject, "location", cfg.CloudTasksLocation,
			"queue", cfg.CloudTasksQueue, "targetURL", cfg.CloudTasksTarget)
		return nil, nil
	}
	e, err := tasks.NewCloudTasksEnqueuer(context.Background(), tasks.CloudTasksConfig{
		ProjectID: cfg.CloudTasksProject,
		Location:  cfg.CloudTasksLocation,
		QueueID:   cfg.CloudTasksQueue,
		TargetURL: cfg.CloudTasksTarget,
		AdminKey:  cfg.AdminKey,
	})
	if err != nil {
		slog.Error("cloud tasks: enqueuer init failed; chain enqueueing disabled", "error", err)
		return nil, nil
	}
	slog.Info("cloud tasks: enqueuer ready",
		"project", cfg.CloudTasksProject, "location", cfg.CloudTasksLocation,
		"queue", cfg.CloudTasksQueue)
	return e, e.Close
}

// chooseSummaryFetcher returns the live ESPN summary fetcher in production
// or a fixture-backed one in TEST_MODE so e2e tests don't depend on ESPN.
func chooseSummaryFetcher(cfg Config) func(matchID string) (models.MatchSummary, error) {
	if cfg.TestMode {
		return espn.FixtureFetcher("internal/espn/testdata")
	}
	return espn.FetchSummary
}

// chooseRefreshFetcher returns the live ESPN scoreboard fetcher in production
// or a no-op (read-from-store) in TEST_MODE so the seed endpoints fully
// control match data.
func chooseRefreshFetcher(cfg Config, matchStore interface {
	GetAll() ([]models.Match, error)
}) func() ([]models.Match, error) {
	if cfg.TestMode {
		return matchStore.GetAll
	}
	return espn.FetchCrewMatches
}
