package handlers_test

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/guilfoos/crew-predictions/internal/handlers"
	"github.com/guilfoos/crew-predictions/internal/repository"
)

func cookieWithPayload(payload map[string]string) *http.Cookie {
	data, _ := json.Marshal(payload)
	return &http.Cookie{Name: "__session", Value: base64.StdEncoding.EncodeToString(data)}
}

func TestUserFromSession_ReturnsNilForEmptyUserID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.AddCookie(cookieWithPayload(map[string]string{"userID": "", "handle": "fan@bsky.mock"}))
	if got := handlers.UserFromSession(req); got != nil {
		t.Errorf("expected nil for empty userID, got %+v", got)
	}
}

func TestUserFromSession_ReturnsNilForEmptyHandle(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.AddCookie(cookieWithPayload(map[string]string{"userID": "google:abc", "handle": ""}))
	if got := handlers.UserFromSession(req); got != nil {
		t.Errorf("expected nil for empty handle, got %+v", got)
	}
}

func TestUserFromSession_RejectsUnsignedCookieWhenSecretSet(t *testing.T) {
	handlers.SetSessionSecret([]byte("test-secret"))
	t.Cleanup(func() { handlers.SetSessionSecret(nil) })

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.AddCookie(cookieWithPayload(map[string]string{"userID": "u1", "handle": "fan"}))
	if got := handlers.UserFromSession(req); got != nil {
		t.Error("expected nil for unsigned cookie when secret is set")
	}
}

func TestUserFromSession_AcceptsValidSignedCookie(t *testing.T) {
	handlers.SetSessionSecret([]byte("test-secret"))
	t.Cleanup(func() { handlers.SetSessionSecret(nil) })

	tok := &handlers.FirebaseToken{UID: "uid1", DisplayName: "Fan", Provider: "google.com"}
	h := handlers.NewSessionHandler(&mockVerifier{token: tok}, repository.NewMemoryUserStore())
	req := httptest.NewRequest(http.MethodPost, "/auth/session", strings.NewReader("idToken=valid"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.Create(w, req)

	req2 := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	for _, c := range w.Result().Cookies() {
		req2.AddCookie(c)
	}
	if got := handlers.UserFromSession(req2); got == nil {
		t.Error("expected user from valid signed cookie")
	}
}

func TestUserFromSession_RejectsTamperedPayload(t *testing.T) {
	handlers.SetSessionSecret([]byte("test-secret"))
	t.Cleanup(func() { handlers.SetSessionSecret(nil) })

	tok := &handlers.FirebaseToken{UID: "uid1", DisplayName: "Fan", Provider: "google.com"}
	h := handlers.NewSessionHandler(&mockVerifier{token: tok}, repository.NewMemoryUserStore())
	req := httptest.NewRequest(http.MethodPost, "/auth/session", strings.NewReader("idToken=valid"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.Create(w, req)

	var cookieVal string
	for _, c := range w.Result().Cookies() {
		if c.Name == "__session" {
			cookieVal = c.Value
		}
	}
	parts := strings.SplitN(cookieVal, ".", 2)
	if len(parts) != 2 {
		t.Fatal("expected signed cookie with dot separator, got: " + cookieVal)
	}
	payload := []byte(parts[0])
	payload[len(payload)-1] ^= 0x01
	tampered := string(payload) + "." + parts[1]

	req2 := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req2.AddCookie(&http.Cookie{Name: "__session", Value: tampered})
	if got := handlers.UserFromSession(req2); got != nil {
		t.Error("expected nil for tampered payload")
	}
}

func TestLogout_ClearsSessionCookieAndRedirects(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/auth/logout", http.NoBody)
	req.AddCookie(&http.Cookie{Name: "__session", Value: "somevalue"})
	w := httptest.NewRecorder()

	handlers.Logout(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected 302 redirect, got %d", w.Code)
	}
	if w.Header().Get("Location") != "/matches" {
		t.Errorf("expected redirect to /matches, got %s", w.Header().Get("Location"))
	}
	var found bool
	for _, c := range w.Result().Cookies() {
		if c.Name == "__session" && c.MaxAge == -1 {
			found = true
		}
	}
	if !found {
		t.Error("expected session cookie to be cleared (MaxAge=-1)")
	}
}
