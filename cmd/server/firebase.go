package main

import (
	"context"
	"log/slog"

	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"

	"github.com/guilfoos/crew-predictions/internal/handlers"
)

// buildVerifier returns a Firebase Auth token verifier when configuration
// allows it; otherwise returns a no-op verifier so the server still boots.
// The no-op verifier rejects every token, which is fine in environments
// where auth isn't configured (local dev without Firebase CLI, tests).
//
// Logs are intentionally non-fatal: a Firebase outage at startup should
// degrade the app gracefully rather than crash. Auth-protected endpoints
// will start returning 401 until a verifier becomes available.
func buildVerifier(ctx context.Context, cfg Config) handlers.TokenVerifier {
	if cfg.FirebaseProject == "" {
		return handlers.NoopTokenVerifier{}
	}

	var appOpts []option.ClientOption
	if cfg.AuthEmulatorHost != "" {
		appOpts = append(appOpts, option.WithoutAuthentication())
	}
	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: cfg.FirebaseProject}, appOpts...)
	if err != nil {
		slog.Error("Firebase unavailable", "error", err)
		return handlers.NoopTokenVerifier{}
	}
	authClient, err := app.Auth(ctx)
	if err != nil {
		slog.Error("Firebase Auth unavailable", "error", err)
		return handlers.NoopTokenVerifier{}
	}
	slog.Info("Firebase Auth initialized", "project", cfg.FirebaseProject)
	return handlers.NewFirebaseTokenVerifier(authClient)
}
