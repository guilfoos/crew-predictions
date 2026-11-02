package main

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/guilfoos/crew-predictions/internal/espn"
	"github.com/guilfoos/crew-predictions/internal/handlers"
	"github.com/guilfoos/crew-predictions/internal/models"
	"github.com/guilfoos/crew-predictions/internal/repository"
)

// fnPtr returns the underlying code pointer of a function value so we can
// assert "this is the same function" without invoking it. Useful for
// distinguishing the live ESPN fetcher from the fixture-backed one.
func fnPtr(f any) uintptr {
	return reflect.ValueOf(f).Pointer()
}

func TestChooseSummaryFetcher_ProductionReturnsLiveFetcher(t *testing.T) {
	got := chooseSummaryFetcher(Config{TestMode: false})
	if fnPtr(got) != fnPtr(espn.FetchSummary) {
		t.Errorf("expected espn.FetchSummary in production mode, got a different function")
	}
}

func TestChooseSummaryFetcher_TestModeReturnsFixtureBacked(t *testing.T) {
	got := chooseSummaryFetcher(Config{TestMode: true})
	if got == nil {
		t.Fatal("expected non-nil fetcher")
	}
	// FixtureFetcher returns a closure, not espn.FetchSummary directly.
	if fnPtr(got) == fnPtr(espn.FetchSummary) {
		t.Error("expected fixture-backed fetcher in test mode, got live FetchSummary")
	}
}

func TestChooseRefreshFetcher_ProductionReturnsLiveFetcher(t *testing.T) {
	got := chooseRefreshFetcher(Config{TestMode: false}, repository.NewMemoryMatchStore())
	if fnPtr(got) != fnPtr(espn.FetchCrewMatches) {
		t.Errorf("expected espn.FetchCrewMatches in production mode, got a different function")
	}
}

func TestChooseRefreshFetcher_TestModeReadsFromStore(t *testing.T) {
	store := repository.NewMemoryMatchStore()
	seeded := []models.Match{{ID: "seeded-1", Kickoff: time.Now()}}
	if err := store.SaveAll(seeded); err != nil {
		t.Fatalf("seed store: %v", err)
	}
	fetcher := chooseRefreshFetcher(Config{TestMode: true}, store)
	got, err := fetcher()
	if err != nil {
		t.Fatalf("fetcher() error: %v", err)
	}
	if len(got) != 1 || got[0].ID != "seeded-1" {
		t.Errorf("expected refresh fetcher to read from store, got %+v", got)
	}
}

func TestBuildDeps_TestModeOmitsMatchPoller(t *testing.T) {
	cfg := Config{TestMode: true, TargetTeam: "Columbus Crew"}
	stores, err := buildStores(t.Context(), cfg)
	if err != nil {
		t.Fatalf("buildStores: %v", err)
	}
	deps := buildDeps(cfg, stores, handlers.NoopTokenVerifier{})

	if deps.MatchPoller != nil {
		t.Errorf("expected MatchPoller nil in test mode, got %T", deps.MatchPoller)
	}
	if deps.SummaryFetcher == nil {
		t.Error("expected SummaryFetcher populated")
	}
	if deps.RefreshFetcher == nil {
		t.Error("expected RefreshFetcher populated")
	}
	if deps.RecalcFn == nil {
		t.Error("expected RecalcFn populated")
	}
	if deps.TwoOneBot == nil {
		t.Error("expected TwoOneBot populated")
	}
}

func TestBuildDeps_ProductionPopulatesMatchPoller(t *testing.T) {
	cfg := Config{TestMode: false, FirestoreProject: "", TargetTeam: "Columbus Crew"}
	stores, err := buildStores(t.Context(), cfg)
	if err != nil {
		t.Fatalf("buildStores: %v", err)
	}
	deps := buildDeps(cfg, stores, handlers.NoopTokenVerifier{})

	if deps.MatchPoller == nil {
		t.Error("expected MatchPoller populated outside test mode")
	}
}

func TestBuildDeps_PassesThroughCfgStoresVerifier(t *testing.T) {
	cfg := Config{TestMode: true, TargetTeam: "Columbus Crew", AdminKey: "marker"}
	stores, err := buildStores(t.Context(), cfg)
	if err != nil {
		t.Fatalf("buildStores: %v", err)
	}
	verifier := handlers.NoopTokenVerifier{}
	deps := buildDeps(cfg, stores, verifier)

	if deps.Cfg.AdminKey != "marker" {
		t.Errorf("Cfg not passed through: AdminKey=%q", deps.Cfg.AdminKey)
	}
	if deps.Stores.Match == nil {
		t.Error("Stores not passed through: Match nil")
	}
	if _, ok := deps.Verifier.(handlers.NoopTokenVerifier); !ok {
		t.Errorf("Verifier not passed through: got %T", deps.Verifier)
	}
}

func TestBuildDeps_RecalcFnDoesNotPanicOnEmptyStores(t *testing.T) {
	cfg := Config{TestMode: true, TargetTeam: "Columbus Crew"}
	stores, err := buildStores(t.Context(), cfg)
	if err != nil {
		t.Fatalf("buildStores: %v", err)
	}
	deps := buildDeps(cfg, stores, handlers.NoopTokenVerifier{})
	deps.RecalcFn(t.Context()) // empty stores: should be a no-op, not a panic
}

func TestDeps_Close_NoopWhenNoEnqueuer(t *testing.T) {
	// Zero-value Deps and Deps without a registered closer should both
	// be safe to Close. The main path defers deps.Close() unconditionally.
	(Deps{}).Close()
	(Deps{Cfg: Config{TestMode: true}}).Close()
}

func TestDeps_Close_InvokesRegisteredCloser(t *testing.T) {
	called := 0
	d := Deps{enqueuerCloser: func() error { called++; return nil }}
	d.Close()
	if called != 1 {
		t.Errorf("expected enqueuerCloser called once, got %d", called)
	}
}

func TestDeps_Close_SwallowsCloserError(t *testing.T) {
	// A closer error should be logged (via slog) but not panic or block
	// shutdown. We can't easily assert on slog output here; this test just
	// verifies the call doesn't panic and the closer is still invoked.
	called := 0
	d := Deps{enqueuerCloser: func() error { called++; return fmt.Errorf("close failed") }}
	d.Close()
	if called != 1 {
		t.Errorf("expected closer invoked even when returning error, got %d", called)
	}
}

func TestBuildCloudTasksEnqueuer_NilWhenConfigIncomplete(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
	}{
		{"all blank", Config{}},
		{"missing project", Config{CloudTasksLocation: "us-east4", CloudTasksQueue: "match-polling", CloudTasksTarget: "https://x/p"}},
		{"missing location", Config{CloudTasksProject: "p", CloudTasksQueue: "match-polling", CloudTasksTarget: "https://x/p"}},
		{"missing queue", Config{CloudTasksProject: "p", CloudTasksLocation: "us-east4", CloudTasksTarget: "https://x/p"}},
		{"missing target", Config{CloudTasksProject: "p", CloudTasksLocation: "us-east4", CloudTasksQueue: "match-polling"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e, closer := buildCloudTasksEnqueuer(tc.cfg)
			if e != nil {
				t.Errorf("expected nil enqueuer for incomplete config, got %T", e)
			}
			if closer != nil {
				t.Errorf("expected nil closer for incomplete config, got non-nil")
			}
		})
	}
}
