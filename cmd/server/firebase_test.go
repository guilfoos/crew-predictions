package main

import (
	"context"
	"testing"

	"github.com/guilfoos/crew-predictions/internal/handlers"
)

func TestBuildVerifier_NoFirebaseProjectReturnsNoop(t *testing.T) {
	cfg := Config{FirebaseProject: ""}
	v := buildVerifier(context.Background(), cfg)
	if _, ok := v.(handlers.NoopTokenVerifier); !ok {
		t.Errorf("expected NoopTokenVerifier when no FirebaseProject, got %T", v)
	}
}

func TestBuildVerifier_TestModeWithEmulatorReturnsNoop(t *testing.T) {
	// In TEST_MODE the e2e suite hits the auth emulator via the front-end,
	// not via the server-side verifier. The verifier path is not exercised,
	// but if anything constructs a Firebase client without auth credentials
	// in tests it should still get a sensible no-op fallback.
	cfg := Config{FirebaseProject: "", AuthEmulatorHost: "localhost:9099"}
	v := buildVerifier(context.Background(), cfg)
	if _, ok := v.(handlers.NoopTokenVerifier); !ok {
		t.Errorf("expected NoopTokenVerifier when FirebaseProject empty even with emulator, got %T", v)
	}
}
