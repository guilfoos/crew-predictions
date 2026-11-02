package tasks_test

import (
	"context"
	"testing"
	"time"

	"github.com/guilfoos/crew-predictions/internal/tasks"
)

func TestFakeEnqueuer_RecordsCallsInOrder(t *testing.T) {
	fake := tasks.NewFakeEnqueuer()
	ctx := context.Background()

	runAt1 := time.Date(2026, 5, 15, 23, 25, 0, 0, time.UTC)
	runAt2 := time.Date(2026, 5, 15, 23, 27, 0, 0, time.UTC)

	if err := fake.EnqueuePoll(ctx, "m-100", runAt1); err != nil {
		t.Fatalf("EnqueuePoll #1: %v", err)
	}
	if err := fake.EnqueuePoll(ctx, "m-101", runAt2); err != nil {
		t.Fatalf("EnqueuePoll #2: %v", err)
	}

	calls := fake.Calls()
	if len(calls) != 2 {
		t.Fatalf("expected 2 recorded calls, got %d", len(calls))
	}
	if calls[0].MatchID != "m-100" || !calls[0].RunAt.Equal(runAt1) {
		t.Errorf("first call: got %+v, want matchID=m-100 runAt=%v", calls[0], runAt1)
	}
	if calls[1].MatchID != "m-101" || !calls[1].RunAt.Equal(runAt2) {
		t.Errorf("second call: got %+v, want matchID=m-101 runAt=%v", calls[1], runAt2)
	}
}

func TestFakeEnqueuer_ResetClearsCalls(t *testing.T) {
	fake := tasks.NewFakeEnqueuer()
	_ = fake.EnqueuePoll(context.Background(), "m-1", time.Now())
	if len(fake.Calls()) != 1 {
		t.Fatalf("precondition: expected 1 call, got %d", len(fake.Calls()))
	}
	fake.Reset()
	if len(fake.Calls()) != 0 {
		t.Errorf("after Reset: expected 0 calls, got %d", len(fake.Calls()))
	}
}
