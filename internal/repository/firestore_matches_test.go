//go:build integration

package repository_test

import (
	"os"
	"testing"
	"time"

	"github.com/guilfoos/crew-predictions/internal/models"
	"github.com/guilfoos/crew-predictions/internal/repository"
)

func firestoreMatchStoreOrSkip(t *testing.T) *repository.FirestoreMatchStore {
	t.Helper()
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("FIRESTORE_EMULATOR_HOST not set; skipping Firestore integration tests")
	}
	store, err := repository.NewFirestoreMatchStore("crew-predictions-test")
	if err != nil {
		t.Fatalf("failed to create FirestoreMatchStore: %v", err)
	}
	return store
}

func TestFirestoreMatchStore_SaveAllAndGetAll(t *testing.T) {
	store := firestoreMatchStoreOrSkip(t)

	matches := []models.Match{
		{ID: "fs-match-1", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas", Kickoff: time.Date(2025, 4, 1, 20, 0, 0, 0, time.UTC)},
		{ID: "fs-match-2", HomeTeam: "Columbus Crew", AwayTeam: "LA Galaxy", Kickoff: time.Date(2025, 4, 8, 19, 0, 0, 0, time.UTC)},
	}
	if err := store.SaveAll(matches); err != nil {
		t.Fatalf("unexpected error saving: %v", err)
	}

	got, err := store.GetAll()
	if err != nil {
		t.Fatalf("unexpected error retrieving: %v", err)
	}
	found := map[string]bool{}
	for _, m := range got {
		found[m.ID] = true
	}
	if !found["fs-match-1"] || !found["fs-match-2"] {
		t.Errorf("expected both matches, got IDs: %v", found)
	}
}

func TestFirestoreMatchStore_VenueRoundTrips(t *testing.T) {
	store := firestoreMatchStoreOrSkip(t)

	if err := store.SaveAll([]models.Match{
		{ID: "fs-match-venue", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas",
			Kickoff: time.Date(2026, 5, 1, 20, 0, 0, 0, time.UTC),
			Venue:   "ScottsMiracle-Gro Field"},
	}); err != nil {
		t.Fatalf("unexpected error saving: %v", err)
	}

	got, err := store.GetAll()
	if err != nil {
		t.Fatalf("unexpected error retrieving: %v", err)
	}
	for _, m := range got {
		if m.ID == "fs-match-venue" {
			if m.Venue != "ScottsMiracle-Gro Field" {
				t.Errorf("expected Venue 'ScottsMiracle-Gro Field', got %q", m.Venue)
			}
			return
		}
	}
	t.Error("match fs-match-venue not found in GetAll results")
}

func TestFirestoreMatchStore_RecordsAndFormRoundTrip(t *testing.T) {
	store := firestoreMatchStoreOrSkip(t)

	if err := store.SaveAll([]models.Match{
		{ID: "fs-match-form", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas",
			Kickoff:    time.Date(2026, 5, 1, 20, 0, 0, 0, time.UTC),
			HomeRecord: "5-3-2", AwayRecord: "4-4-2",
			HomeForm: "WWWLL", AwayForm: "LWDWL"},
	}); err != nil {
		t.Fatalf("unexpected error saving: %v", err)
	}

	got, err := store.GetAll()
	if err != nil {
		t.Fatalf("unexpected error retrieving: %v", err)
	}
	for _, m := range got {
		if m.ID == "fs-match-form" {
			if m.HomeRecord != "5-3-2" {
				t.Errorf("HomeRecord: got %q, want %q", m.HomeRecord, "5-3-2")
			}
			if m.AwayRecord != "4-4-2" {
				t.Errorf("AwayRecord: got %q, want %q", m.AwayRecord, "4-4-2")
			}
			if m.HomeForm != "WWWLL" {
				t.Errorf("HomeForm: got %q, want %q", m.HomeForm, "WWWLL")
			}
			if m.AwayForm != "LWDWL" {
				t.Errorf("AwayForm: got %q, want %q", m.AwayForm, "LWDWL")
			}
			return
		}
	}
	t.Error("match fs-match-form not found in GetAll results")
}

func TestFirestoreMatchStore_AttendanceRoundTrips(t *testing.T) {
	store := firestoreMatchStoreOrSkip(t)

	if err := store.SaveAll([]models.Match{
		{ID: "fs-match-attendance", HomeTeam: "Columbus Crew", AwayTeam: "Philadelphia Union",
			Kickoff: time.Date(2026, 4, 25, 23, 0, 0, 0, time.UTC),
			State:   "post", Status: "STATUS_FULL_TIME", Attendance: 19903},
	}); err != nil {
		t.Fatalf("unexpected error saving: %v", err)
	}

	got, err := store.GetAll()
	if err != nil {
		t.Fatalf("unexpected error retrieving: %v", err)
	}
	for _, m := range got {
		if m.ID == "fs-match-attendance" {
			if m.Attendance != 19903 {
				t.Errorf("expected Attendance 19903, got %d", m.Attendance)
			}
			return
		}
	}
	t.Error("match fs-match-attendance not found in GetAll results")
}

// TestFirestoreMatchStore_FullMatchRoundTrip is a guard against the
// "field added to model but forgot to update encoder/decoder" bug class.
// Same shape as the April 2026 toUser/UpdateScores incident: a write path
// and a read path must stay in sync, but unit tests use the Memory store
// and would silently pass if Firestore round-trip dropped a field.
//
// This test populates every field of a Match (and every sub-field of
// MatchEvent), persists via SaveAll, reads back via GetAll, and asserts
// each value matches. A future commit that adds a field to models.Match
// or models.MatchEvent without updating both SaveAll's write map AND
// toMatch's firestore-tagged struct will fail this test loudly.
//
// Uses t.Errorf (not t.Fatalf) for the field comparisons so all mismatches
// from a regression surface in one run rather than stopping at the first.
func TestFirestoreMatchStore_FullMatchRoundTrip(t *testing.T) {
	store := firestoreMatchStoreOrSkip(t)

	kickoff := time.Date(2026, 4, 25, 23, 0, 0, 0, time.UTC)
	original := models.Match{
		ID:           "fs-full-roundtrip",
		HomeTeam:     "Columbus Crew",
		AwayTeam:     "Philadelphia Union",
		Kickoff:      kickoff,
		Status:       "STATUS_FULL_TIME",
		State:        "post",
		HomeScore:    "2",
		AwayScore:    "0",
		DisplayClock: "FT",
		Venue:        "ScottsMiracle-Gro Field",
		HomeRecord:   "5-3-2",
		AwayRecord:   "4-4-2",
		HomeForm:     "WWWLL",
		AwayForm:     "LWDWL",
		HomeLogo:     "https://a.espncdn.com/i/teamlogos/soccer/500/183.png",
		AwayLogo:     "https://a.espncdn.com/i/teamlogos/soccer/500/10739.png",
		Attendance:   19903,
		Referee:      "Pierre-Luc Lauziere",
		Events: []models.MatchEvent{
			{Clock: "4'", TypeID: "goal", Team: "Columbus Crew", Players: []string{"Max Arfsten"}},
			{Clock: "39'", TypeID: "yellow-card", Team: "Philadelphia Union", Players: []string{"Danley Jean Jacques"}},
			{Clock: "73'", TypeID: "substitution", Team: "Columbus Crew", Players: []string{"Steven Moreira", "Hugo Picard"}},
		},
	}
	if err := store.SaveAll([]models.Match{original}); err != nil {
		t.Fatalf("SaveAll failed: %v", err)
	}

	got, err := store.GetAll()
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	var found *models.Match
	for i, m := range got {
		if m.ID == "fs-full-roundtrip" {
			found = &got[i]
			break
		}
	}
	if found == nil {
		t.Fatal("match fs-full-roundtrip not found in GetAll results")
	}

	// Top-level fields
	if found.HomeTeam != original.HomeTeam {
		t.Errorf("HomeTeam: got %q, want %q", found.HomeTeam, original.HomeTeam)
	}
	if found.AwayTeam != original.AwayTeam {
		t.Errorf("AwayTeam: got %q, want %q", found.AwayTeam, original.AwayTeam)
	}
	if !found.Kickoff.Equal(original.Kickoff) {
		t.Errorf("Kickoff: got %v, want %v", found.Kickoff, original.Kickoff)
	}
	if found.Status != original.Status {
		t.Errorf("Status: got %q, want %q", found.Status, original.Status)
	}
	if found.State != original.State {
		t.Errorf("State: got %q, want %q", found.State, original.State)
	}
	if found.HomeScore != original.HomeScore {
		t.Errorf("HomeScore: got %q, want %q", found.HomeScore, original.HomeScore)
	}
	if found.AwayScore != original.AwayScore {
		t.Errorf("AwayScore: got %q, want %q", found.AwayScore, original.AwayScore)
	}
	if found.DisplayClock != original.DisplayClock {
		t.Errorf("DisplayClock: got %q, want %q", found.DisplayClock, original.DisplayClock)
	}
	if found.Venue != original.Venue {
		t.Errorf("Venue: got %q, want %q", found.Venue, original.Venue)
	}
	if found.HomeRecord != original.HomeRecord {
		t.Errorf("HomeRecord: got %q, want %q", found.HomeRecord, original.HomeRecord)
	}
	if found.AwayRecord != original.AwayRecord {
		t.Errorf("AwayRecord: got %q, want %q", found.AwayRecord, original.AwayRecord)
	}
	if found.HomeForm != original.HomeForm {
		t.Errorf("HomeForm: got %q, want %q", found.HomeForm, original.HomeForm)
	}
	if found.AwayForm != original.AwayForm {
		t.Errorf("AwayForm: got %q, want %q", found.AwayForm, original.AwayForm)
	}
	if found.HomeLogo != original.HomeLogo {
		t.Errorf("HomeLogo: got %q, want %q", found.HomeLogo, original.HomeLogo)
	}
	if found.AwayLogo != original.AwayLogo {
		t.Errorf("AwayLogo: got %q, want %q", found.AwayLogo, original.AwayLogo)
	}
	if found.Attendance != original.Attendance {
		t.Errorf("Attendance: got %d, want %d", found.Attendance, original.Attendance)
	}
	if found.Referee != original.Referee {
		t.Errorf("Referee: got %q, want %q", found.Referee, original.Referee)
	}

	// Events: count and per-field check across all entries
	if len(found.Events) != len(original.Events) {
		t.Fatalf("Events length: got %d, want %d", len(found.Events), len(original.Events))
	}
	for i, want := range original.Events {
		got := found.Events[i]
		if got.Clock != want.Clock {
			t.Errorf("Events[%d].Clock: got %q, want %q", i, got.Clock, want.Clock)
		}
		if got.TypeID != want.TypeID {
			t.Errorf("Events[%d].TypeID: got %q, want %q", i, got.TypeID, want.TypeID)
		}
		if got.Team != want.Team {
			t.Errorf("Events[%d].Team: got %q, want %q", i, got.Team, want.Team)
		}
		if len(got.Players) != len(want.Players) {
			t.Errorf("Events[%d].Players length: got %d, want %d", i, len(got.Players), len(want.Players))
			continue
		}
		for j, p := range want.Players {
			if got.Players[j] != p {
				t.Errorf("Events[%d].Players[%d]: got %q, want %q", i, j, got.Players[j], p)
			}
		}
	}
}

func TestFirestoreMatchStore_SaveAllOverwritesExisting(t *testing.T) {
	store := firestoreMatchStoreOrSkip(t)

	_ = store.SaveAll([]models.Match{
		{ID: "fs-match-overwrite", HomeTeam: "Columbus Crew", AwayTeam: "Old Opponent", Kickoff: time.Now()},
	})
	_ = store.SaveAll([]models.Match{
		{ID: "fs-match-overwrite", HomeTeam: "Columbus Crew", AwayTeam: "New Opponent", Kickoff: time.Now()},
	})

	got, err := store.GetAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, m := range got {
		if m.ID == "fs-match-overwrite" && m.AwayTeam != "New Opponent" {
			t.Errorf("expected overwritten record, got AwayTeam=%q", m.AwayTeam)
		}
	}
}

func TestFirestoreMatchStore_RoundTripsLastPollAt(t *testing.T) {
	store := firestoreMatchStoreOrSkip(t)

	polledAt := time.Date(2026, 5, 13, 23, 50, 0, 0, time.UTC)
	want := models.Match{
		ID: "fs-match-lastpollat", HomeTeam: "Columbus Crew", AwayTeam: "NYRB",
		Kickoff: time.Date(2026, 5, 13, 23, 30, 0, 0, time.UTC), Status: "STATUS_IN_PROGRESS", State: "in",
		LastPollAt: polledAt,
	}
	if err := store.SaveAll([]models.Match{want}); err != nil {
		t.Fatalf("SaveAll: %v", err)
	}
	got, err := store.GetAll()
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	var found *models.Match
	for i, m := range got {
		if m.ID == "fs-match-lastpollat" {
			found = &got[i]
			break
		}
	}
	if found == nil {
		t.Fatal("expected fs-match-lastpollat to be present")
	}
	if !found.LastPollAt.Equal(polledAt) {
		t.Errorf("LastPollAt round-trip: got %v, want %v", found.LastPollAt, polledAt)
	}
}

func TestFirestoreMatchStore_RoundTripsChainSeededForAndAbandonedAt(t *testing.T) {
	store := firestoreMatchStoreOrSkip(t)

	seededFor := time.Date(2026, 5, 13, 23, 30, 0, 0, time.UTC)
	abandonedAt := time.Date(2026, 5, 14, 3, 30, 0, 0, time.UTC)
	want := models.Match{
		ID: "fs-match-chain-fields", HomeTeam: "Columbus Crew", AwayTeam: "NYRB",
		Kickoff: seededFor, Status: "STATUS_IN_PROGRESS", State: "in",
		ChainSeededFor: seededFor,
		AbandonedAt:    abandonedAt,
	}
	if err := store.SaveAll([]models.Match{want}); err != nil {
		t.Fatalf("SaveAll: %v", err)
	}
	got, err := store.GetAll()
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	var found *models.Match
	for i, m := range got {
		if m.ID == "fs-match-chain-fields" {
			found = &got[i]
			break
		}
	}
	if found == nil {
		t.Fatal("expected fs-match-chain-fields to be present")
	}
	if !found.ChainSeededFor.Equal(seededFor) {
		t.Errorf("ChainSeededFor round-trip: got %v, want %v", found.ChainSeededFor, seededFor)
	}
	if !found.AbandonedAt.Equal(abandonedAt) {
		t.Errorf("AbandonedAt round-trip: got %v, want %v", found.AbandonedAt, abandonedAt)
	}
}
