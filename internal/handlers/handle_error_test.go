package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/guilfoos/crew-predictions/internal/repository"
)

type errUserStore struct{ repository.UserStore }

func (e *errUserStore) Upsert(_ context.Context, _ repository.User) error {
	return fmt.Errorf("store failed")
}
func (e *errUserStore) GetAll(_ context.Context) ([]repository.User, error) {
	return nil, fmt.Errorf("store failed")
}

func TestHandleHandler_Returns500WhenUpsertFails(t *testing.T) {
	h := NewHandleHandler(&errUserStore{UserStore: repository.NewMemoryUserStore()})
	form := url.Values{"handle": {"CrewForever"}}
	req := httptest.NewRequest(http.MethodPost, "/auth/handle", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(sessionCookie("firebase:abc", "oldfan"))
	w := httptest.NewRecorder()
	h.Update(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}
