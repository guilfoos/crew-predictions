package handlers_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/guilfoos/crew-predictions/internal/handlers"
	"github.com/guilfoos/crew-predictions/internal/repository"
)

type mockVerifier struct {
	token *handlers.FirebaseToken
	err   error
}

func (m *mockVerifier) VerifyIDToken(ctx context.Context, idToken string) (*handlers.FirebaseToken, error) {
	return m.token, m.err
}

func newSession(v handlers.TokenVerifier) *handlers.SessionHandler {
	return handlers.NewSessionHandler(v, repository.NewMemoryUserStore())
}

func TestSessionHandler_UpsertUserOnCreate(t *testing.T) {
	tok := &handlers.FirebaseToken{UID: "uid123", DisplayName: "CrewFan", Provider: "google.com"}
	users := repository.NewMemoryUserStore()
	h := handlers.NewSessionHandler(&mockVerifier{token: tok}, users)
	req := httptest.NewRequest(http.MethodPost, "/auth/session", strings.NewReader("idToken=valid"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.Create(httptest.NewRecorder(), req)

	u, _ := users.GetByID(context.Background(), "firebase:uid123")
	if u == nil || u.Handle != "CrewFan" {
		t.Errorf("expected user upserted with handle CrewFan, got %+v", u)
	}
}

func TestSessionHandler_Returns400WhenNoToken(t *testing.T) {
	h := newSession(&mockVerifier{})
	req := httptest.NewRequest(http.MethodPost, "/auth/session", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.Create(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSessionHandler_Returns401WhenTokenInvalid(t *testing.T) {
	h := newSession(&mockVerifier{err: errors.New("bad token")})
	req := httptest.NewRequest(http.MethodPost, "/auth/session", strings.NewReader("idToken=invalid"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.Create(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestSessionHandler_SetsSessionCookieOnValidToken(t *testing.T) {
	tok := &handlers.FirebaseToken{UID: "uid123", Email: "user@example.com", Provider: "google.com"}
	h := newSession(&mockVerifier{token: tok})
	req := httptest.NewRequest(http.MethodPost, "/auth/session", strings.NewReader("idToken=valid"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.Create(w, req)

	var found bool
	for _, c := range w.Result().Cookies() {
		if c.Name == "__session" && c.Value != "" {
			found = true
		}
	}
	if !found {
		t.Error("expected session cookie to be set")
	}
}

func TestSessionHandler_Returns200OnSuccess(t *testing.T) {
	tok := &handlers.FirebaseToken{UID: "uid123", Email: "user@example.com", Provider: "google.com"}
	h := newSession(&mockVerifier{token: tok})
	req := httptest.NewRequest(http.MethodPost, "/auth/session", strings.NewReader("idToken=valid"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.Create(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestSessionHandler_HandleIsDisplayNameWhenSet(t *testing.T) {
	tok := &handlers.FirebaseToken{UID: "uid456", Email: "user@example.com", DisplayName: "Nordecke Regular", Provider: "password"}
	h := newSession(&mockVerifier{token: tok})
	req := httptest.NewRequest(http.MethodPost, "/auth/session", strings.NewReader("idToken=valid"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.Create(w, req)

	resp := w.Result()
	req2 := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	for _, c := range resp.Cookies() {
		req2.AddCookie(c)
	}
	user := handlers.UserFromSession(req2)
	if user == nil {
		t.Fatal("expected user from session, got nil")
	}
	if user.Handle != "Nordecke Regular" {
		t.Errorf("expected handle %q, got %q", "Nordecke Regular", user.Handle)
	}
}

func TestSessionHandler_HandleFallsBackToEmailWhenNoDisplayName(t *testing.T) {
	tok := &handlers.FirebaseToken{UID: "uid789", Email: "user@example.com", Provider: "password"}
	h := newSession(&mockVerifier{token: tok})
	req := httptest.NewRequest(http.MethodPost, "/auth/session", strings.NewReader("idToken=valid"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.Create(w, req)

	resp := w.Result()
	req2 := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	for _, c := range resp.Cookies() {
		req2.AddCookie(c)
	}
	user := handlers.UserFromSession(req2)
	if user == nil {
		t.Fatal("expected user from session, got nil")
	}
	if user.Handle != "user@example.com" {
		t.Errorf("expected handle %q, got %q", "user@example.com", user.Handle)
	}
}

func TestSessionHandler_SessionContainsEmailVerifiedTrue(t *testing.T) {
	tok := &handlers.FirebaseToken{UID: "uid1", Email: "user@example.com", EmailVerified: true, Provider: "password"}
	h := newSession(&mockVerifier{token: tok})
	req := httptest.NewRequest(http.MethodPost, "/auth/session", strings.NewReader("idToken=valid"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.Create(w, req)

	resp := w.Result()
	req2 := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	for _, c := range resp.Cookies() {
		req2.AddCookie(c)
	}
	user := handlers.UserFromSession(req2)
	if user == nil {
		t.Fatal("expected user from session, got nil")
	}
	if !user.EmailVerified {
		t.Error("expected EmailVerified to be true")
	}
}

func TestSessionHandler_SessionContainsFirebaseUID(t *testing.T) {
	tok := &handlers.FirebaseToken{UID: "uid123", Email: "user@example.com", Provider: "google.com"}
	h := newSession(&mockVerifier{token: tok})
	req := httptest.NewRequest(http.MethodPost, "/auth/session", strings.NewReader("idToken=valid"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.Create(w, req)

	resp := w.Result()
	req2 := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	for _, c := range resp.Cookies() {
		req2.AddCookie(c)
	}
	user := handlers.UserFromSession(req2)
	if user == nil {
		t.Fatal("expected user from session, got nil")
	}
	if user.UserID != "firebase:uid123" {
		t.Errorf("expected userID firebase:uid123, got %q", user.UserID)
	}
	if user.Handle != "user@example.com" {
		t.Errorf("expected handle user@example.com, got %q", user.Handle)
	}
}
