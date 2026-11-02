package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/guilfoos/crew-predictions/internal/repository"
)

func TestWriteSessionCookie_SecureFlagSetOutsideEmulator(t *testing.T) {
	t.Setenv("FIREBASE_AUTH_EMULATOR_HOST", "")
	w := httptest.NewRecorder()
	writeSessionCookie(w, sessionPayload{UserID: "u1", Handle: "fan"})
	for _, c := range w.Result().Cookies() {
		if c.Name == "__session" && !c.Secure {
			t.Error("expected Secure=true when FIREBASE_AUTH_EMULATOR_HOST is unset")
		}
	}
}

func TestWriteSessionCookie_SecureFlagOffInEmulator(t *testing.T) {
	t.Setenv("FIREBASE_AUTH_EMULATOR_HOST", "localhost:9099")
	w := httptest.NewRecorder()
	writeSessionCookie(w, sessionPayload{UserID: "u1", Handle: "fan"})
	for _, c := range w.Result().Cookies() {
		if c.Name == "__session" && c.Secure {
			t.Error("expected Secure=false when FIREBASE_AUTH_EMULATOR_HOST is set (local dev)")
		}
	}
}

func sessionCookie(userID, handle string) *http.Cookie {
	data, _ := json.Marshal(sessionPayload{UserID: userID, Handle: handle, EmailVerified: true})
	return &http.Cookie{Name: "__session", Value: base64.StdEncoding.EncodeToString(data)}
}

func TestHandleHandler_UpdatesUserStoreAndRewritesCookie(t *testing.T) {
	users := repository.NewMemoryUserStore()
	h := NewHandleHandler(users)

	form := url.Values{"handle": {"CrewForever"}}
	req := httptest.NewRequest(http.MethodPost, "/auth/handle", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(sessionCookie("firebase:abc", "oldfan"))
	w := httptest.NewRecorder()

	h.Update(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	u, _ := users.GetByID(context.Background(), "firebase:abc")
	if u == nil || u.Handle != "CrewForever" {
		t.Errorf("expected UserStore updated with CrewForever, got %+v", u)
	}

	var newCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "__session" {
			newCookie = c
		}
	}
	if newCookie == nil {
		t.Fatal("expected __session cookie to be rewritten")
	}
	data, _ := base64.StdEncoding.DecodeString(newCookie.Value)
	var payload sessionPayload
	json.Unmarshal(data, &payload)
	if payload.Handle != "CrewForever" {
		t.Errorf("expected cookie handle CrewForever, got %q", payload.Handle)
	}
	if payload.UserID != "firebase:abc" {
		t.Errorf("expected cookie userID preserved, got %q", payload.UserID)
	}
}

func TestHandleHandler_SavesLocationToUserStore(t *testing.T) {
	users := repository.NewMemoryUserStore()
	h := NewHandleHandler(users)

	form := url.Values{"handle": {"CrewForever"}, "location": {"Columbus, OH"}}
	req := httptest.NewRequest(http.MethodPost, "/auth/handle", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(sessionCookie("firebase:abc", "oldfan"))
	w := httptest.NewRecorder()
	h.Update(w, req)

	u, _ := users.GetByID(context.Background(), "firebase:abc")
	if u == nil || u.Location != "Columbus, OH" {
		t.Errorf("expected location Columbus, OH stored, got %+v", u)
	}
}

func TestHandleHandler_RequiresSession(t *testing.T) {
	h := NewHandleHandler(repository.NewMemoryUserStore())
	req := httptest.NewRequest(http.MethodPost, "/auth/handle", strings.NewReader("handle=x"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.Update(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestHandleHandler_RejectsMissingHandle(t *testing.T) {
	h := NewHandleHandler(repository.NewMemoryUserStore())
	req := httptest.NewRequest(http.MethodPost, "/auth/handle", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(sessionCookie("firebase:abc", "oldfan"))
	w := httptest.NewRecorder()
	h.Update(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
