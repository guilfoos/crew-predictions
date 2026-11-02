// Package handlers exposes the HTTP handlers that serve the JSON, HTMX, and
// admin endpoints. Handlers are constructed in cmd/server and wired to the
// repository layer via interfaces.
package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/guilfoos/crew-predictions/internal/repository"
)

var sessionSecret []byte

// SetSessionSecret configures the HMAC key for session cookie signing.
// Call at startup from main.go; in tests, call with t.Cleanup to restore nil.
func SetSessionSecret(key []byte) {
	sessionSecret = key
}

func Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "__session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   os.Getenv("FIREBASE_AUTH_EMULATOR_HOST") == "",
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/matches", http.StatusFound)
}

type sessionPayload struct {
	UserID        string `json:"userID"`
	Handle        string `json:"handle"`
	Provider      string `json:"provider"`
	EmailVerified bool   `json:"emailVerified"`
}

func UserFromSession(r *http.Request) *repository.User {
	cookie, err := r.Cookie("__session")
	if err != nil {
		return nil
	}
	value := cookie.Value
	if len(sessionSecret) > 0 {
		parts := strings.SplitN(value, ".", 2)
		if len(parts) != 2 {
			return nil
		}
		got, err := hex.DecodeString(parts[1])
		if err != nil {
			return nil
		}
		mac := hmac.New(sha256.New, sessionSecret)
		mac.Write([]byte(parts[0]))
		if !hmac.Equal(mac.Sum(nil), got) {
			return nil
		}
		value = parts[0]
	}
	data, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil
	}
	var session sessionPayload
	if err := json.Unmarshal(data, &session); err != nil {
		return nil
	}
	if session.UserID == "" || session.Handle == "" {
		return nil
	}
	return &repository.User{
		UserID:        session.UserID,
		Handle:        session.Handle,
		Provider:      session.Provider,
		EmailVerified: session.EmailVerified,
	}
}
