package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/guilfoos/crew-predictions/internal/repository"
)

type MeHandler struct {
	users repository.UserStore
}

func NewMeHandler(users repository.UserStore) *MeHandler {
	return &MeHandler{users: users}
}

func (h *MeHandler) Get(w http.ResponseWriter, r *http.Request) {
	user := UserFromSession(r)
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if user.UserID != "" {
		if err := h.users.Upsert(r.Context(), repository.User{UserID: user.UserID, Handle: user.Handle, Provider: user.Provider}); err != nil {
			log.Printf("me: lazy upsert failed for %s: %v", user.UserID, err)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"userID":        user.UserID,
		"handle":        user.Handle,
		"emailVerified": user.EmailVerified,
	}); err != nil {
		log.Printf("me: encode response: %v", err)
	}
}
