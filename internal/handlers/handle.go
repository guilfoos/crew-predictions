package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"

	"github.com/guilfoos/crew-predictions/internal/repository"
)

type HandleHandler struct {
	users repository.UserStore
}

func NewHandleHandler(users repository.UserStore) *HandleHandler {
	return &HandleHandler{users: users}
}

func (h *HandleHandler) Update(w http.ResponseWriter, r *http.Request) {
	user := UserFromSession(r)
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	_ = r.ParseForm()
	handle := r.FormValue("handle")
	if handle == "" {
		http.Error(w, "handle is required", http.StatusBadRequest)
		return
	}
	location := r.FormValue("location")

	if err := h.users.Upsert(r.Context(), repository.User{
		UserID:   user.UserID,
		Handle:   handle,
		Provider: user.Provider,
		Location: location,
	}); err != nil {
		http.Error(w, "could not save handle", http.StatusInternalServerError)
		return
	}

	writeSessionCookie(w, sessionPayload{
		UserID:        user.UserID,
		Handle:        handle,
		Provider:      user.Provider,
		EmailVerified: user.EmailVerified,
	})
	w.WriteHeader(http.StatusOK)
}

func writeSessionCookie(w http.ResponseWriter, payload sessionPayload) {
	data, _ := json.Marshal(payload)
	encoded := base64.StdEncoding.EncodeToString(data)
	value := encoded
	if len(sessionSecret) > 0 {
		mac := hmac.New(sha256.New, sessionSecret)
		mac.Write([]byte(encoded))
		value = encoded + "." + hex.EncodeToString(mac.Sum(nil))
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "__session",
		Value:    value,
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
		Secure:   os.Getenv("FIREBASE_AUTH_EMULATOR_HOST") == "",
		SameSite: http.SameSiteLaxMode,
	})
}
