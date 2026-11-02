package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/guilfoos/crew-predictions/internal/repository"
)

type FirebaseToken struct {
	UID           string
	Email         string
	DisplayName   string
	EmailVerified bool
	Provider      string
}

type TokenVerifier interface {
	VerifyIDToken(ctx context.Context, idToken string) (*FirebaseToken, error)
}

// NoopTokenVerifier is used in local dev when Firebase credentials are absent.
type NoopTokenVerifier struct{}

func (NoopTokenVerifier) VerifyIDToken(ctx context.Context, idToken string) (*FirebaseToken, error) {
	return nil, fmt.Errorf("firebase auth not configured")
}

type SessionHandler struct {
	verifier TokenVerifier
	users    repository.UserStore
}

func NewSessionHandler(v TokenVerifier, users repository.UserStore) *SessionHandler {
	return &SessionHandler{verifier: v, users: users}
}

func (h *SessionHandler) Create(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	idToken := r.FormValue("idToken")
	if idToken == "" {
		http.Error(w, "missing idToken", http.StatusBadRequest)
		return
	}

	tok, err := h.verifier.VerifyIDToken(r.Context(), idToken)
	if err != nil {
		log.Printf("VerifyIDToken failed: %v", err)
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	handle := tok.DisplayName
	if handle == "" {
		handle = tok.Email
	}
	userID := "firebase:" + tok.UID
	if err := h.users.Upsert(r.Context(), repository.User{
		UserID:   userID,
		Handle:   handle,
		Provider: tok.Provider,
	}); err != nil {
		log.Printf("upsert user on session create failed: %v", err)
	}

	writeSessionCookie(w, sessionPayload{
		UserID:        userID,
		Handle:        handle,
		Provider:      tok.Provider,
		EmailVerified: tok.EmailVerified,
	})
	w.WriteHeader(http.StatusOK)
}
