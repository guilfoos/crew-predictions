package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/guilfoos/crew-predictions/internal/models"
	"github.com/guilfoos/crew-predictions/internal/repository"
)

// errReader always returns an error on Read, simulating a broken HTTP body.
type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, fmt.Errorf("body read failed") }
func (errReader) Close() error             { return nil }

func bodyErrRequest(method, path string) *http.Request {
	r := httptest.NewRequest(method, path, io.NopCloser(errReader{}))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return r
}

func TestPredictionsHandler_Returns400WhenParseFails(t *testing.T) {
	future := time.Now().Add(24 * time.Hour)
	fetcher := func() ([]models.Match, error) {
		return []models.Match{{ID: "m1", Kickoff: future, Status: "STATUS_SCHEDULED"}}, nil
	}
	h := NewPredictionsHandler(repository.NewMemoryPredictionStore(), fetcher)
	req := bodyErrRequest(http.MethodPost, "/api/predictions")
	data, _ := json.Marshal(sessionPayload{UserID: "u1", Handle: "fan"})
	req.AddCookie(&http.Cookie{Name: "__session", Value: base64.StdEncoding.EncodeToString(data)})
	w := httptest.NewRecorder()
	h.Submit(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 on ParseForm error, got %d", w.Code)
	}
}

func TestResultsHandler_Returns400WhenParseFails(t *testing.T) {
	h := NewResultsHandler(repository.NewMemoryResultStore(), func(_ context.Context) {})
	req := bodyErrRequest(http.MethodPost, "/admin/results")
	w := httptest.NewRecorder()
	h.Submit(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 on ParseForm error, got %d", w.Code)
	}
}

func TestHandleHandler_Returns400WhenParseFails(t *testing.T) {
	h := NewHandleHandler(repository.NewMemoryUserStore())
	req := bodyErrRequest(http.MethodPost, "/auth/handle")
	data, _ := json.Marshal(sessionPayload{UserID: "u1", Handle: "fan"})
	req.AddCookie(&http.Cookie{Name: "__session", Value: base64.StdEncoding.EncodeToString(data)})
	w := httptest.NewRecorder()
	h.Update(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 on ParseForm error, got %d", w.Code)
	}
}

func TestSessionHandler_Returns400WhenParseFails(t *testing.T) {
	h := NewSessionHandler(NoopTokenVerifier{}, repository.NewMemoryUserStore())
	req := bodyErrRequest(http.MethodPost, "/auth/session")
	w := httptest.NewRecorder()
	h.Create(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 on ParseForm error, got %d", w.Code)
	}
}

func TestSeedPredictionHandler_Returns400WhenParseFails(t *testing.T) {
	h := NewSeedPredictionHandler(repository.NewMemoryPredictionStore())
	req := bodyErrRequest(http.MethodPost, "/admin/seed-prediction")
	w := httptest.NewRecorder()
	h.Submit(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 on ParseForm error, got %d", w.Code)
	}
}
