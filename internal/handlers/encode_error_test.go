package handlers_test

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/guilfoos/crew-predictions/internal/handlers"
	"github.com/guilfoos/crew-predictions/internal/models"
	"github.com/guilfoos/crew-predictions/internal/repository"
)

// failWriter is an http.ResponseWriter whose Write always fails.
type failWriter struct {
	header http.Header
	code   int
}

func newFailWriter() *failWriter           { return &failWriter{header: http.Header{}} }
func (f *failWriter) Header() http.Header  { return f.header }
func (f *failWriter) WriteHeader(code int) { f.code = code }
func (f *failWriter) Write([]byte) (int, error) {
	return 0, errWriteFailed
}

type writeError struct{}

func (writeError) Error() string { return "write failed" }

var errWriteFailed = writeError{}

func captureLog(t *testing.T, fn func()) string {
	t.Helper()
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)
	fn()
	return buf.String()
}

func TestLeaderboardHandler_LogsEncodeError(t *testing.T) {
	h := handlers.NewLeaderboardHandler(
		repository.NewMemoryPredictionStore(),
		repository.NewMemoryResultStore(),
		repository.NewMemoryUserStore(),
		repository.NewMemorySeasonStore(),
		"Columbus Crew",
	)
	req := httptest.NewRequest(http.MethodGet, "/api/leaderboard", http.NoBody)
	got := captureLog(t, func() { h.APIList(newFailWriter(), req) })
	if !strings.Contains(got, "encode") {
		t.Errorf("expected encode error logged, got %q", got)
	}
}

func TestMatchesHandler_LogsEncodeError(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	matchStore.Seed([]models.Match{{ID: "m1", HomeTeam: "Columbus Crew", AwayTeam: "LA Galaxy", Kickoff: time.Now()}})
	h := handlers.NewMatchesHandler(repository.NewMemoryPredictionStore(), matchStore)
	req := httptest.NewRequest(http.MethodGet, "/api/matches", http.NoBody)
	got := captureLog(t, func() { h.APIList(newFailWriter(), req) })
	if !strings.Contains(got, "encode") {
		t.Errorf("expected encode error logged, got %q", got)
	}
}

func TestMeHandler_LogsEncodeError(t *testing.T) {
	h := handlers.NewMeHandler(repository.NewMemoryUserStore())
	req := httptest.NewRequest(http.MethodGet, "/api/me", http.NoBody)
	req.AddCookie(sessionCookie("u1", "TestUser"))
	got := captureLog(t, func() { h.Get(newFailWriter(), req) })
	if !strings.Contains(got, "encode") {
		t.Errorf("expected encode error logged, got %q", got)
	}
}

func TestProfileHandler_LogsEncodeError(t *testing.T) {
	users := repository.NewMemoryUserStore()
	_ = users.Upsert(context.TODO(), repository.User{UserID: "u1", Handle: "Fan"})
	h := handlers.NewProfileHandler(
		repository.NewMemoryPredictionStore(),
		repository.NewMemoryResultStore(),
		users,
		"Columbus Crew",
	)
	req := httptest.NewRequest(http.MethodGet, "/api/profile/u1", http.NoBody)
	req.SetPathValue("userID", "u1")
	got := captureLog(t, func() { h.Get(newFailWriter(), req) })
	if !strings.Contains(got, "encode") {
		t.Errorf("expected encode error logged, got %q", got)
	}
}

func TestMatchDetailHandler_LogsEncodeError(t *testing.T) {
	matchStore := repository.NewMemoryMatchStore()
	matchStore.Seed([]models.Match{{ID: "m1", HomeTeam: "Columbus Crew", AwayTeam: "LA Galaxy", Kickoff: time.Now()}})
	h := handlers.NewMatchDetailHandler(
		repository.NewMemoryPredictionStore(),
		repository.NewMemoryResultStore(),
		matchStore,
		repository.NewMemoryUserStore(),
		"Columbus Crew",
		nil,
	)
	req := httptest.NewRequest(http.MethodGet, "/api/matches/m1", http.NoBody)
	req.SetPathValue("matchId", "m1")
	got := captureLog(t, func() { h.Get(newFailWriter(), req) })
	if !strings.Contains(got, "encode") {
		t.Errorf("expected encode error logged, got %q", got)
	}
}
