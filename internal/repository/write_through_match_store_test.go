package repository_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/guilfoos/crew-predictions/internal/models"
	"github.com/guilfoos/crew-predictions/internal/repository"
)

func TestWriteThroughMatchStore_SaveAllWritesToBothStores(t *testing.T) {
	primary := repository.NewMemoryMatchStore()
	secondary := repository.NewMemoryMatchStore()
	s := repository.NewWriteThroughMatchStore(primary, secondary)

	matches := []models.Match{{ID: "m1", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas", Kickoff: time.Now()}}
	if err := s.SaveAll(matches); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := primary.GetAll()
	if len(got) != 1 {
		t.Errorf("expected primary store to have 1 match, got %d", len(got))
	}
	got2, _ := secondary.GetAll()
	if len(got2) != 1 {
		t.Errorf("expected secondary store to have 1 match, got %d", len(got2))
	}
}

func TestWriteThroughMatchStore_GetAllReadsFromPrimary(t *testing.T) {
	primary := repository.NewMemoryMatchStore()
	secondary := repository.NewMemoryMatchStore()

	primary.SaveAll([]models.Match{{ID: "m1", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas", Kickoff: time.Now()}})
	// secondary is empty

	s := repository.NewWriteThroughMatchStore(primary, secondary)
	got, _ := s.GetAll()
	if len(got) != 1 {
		t.Errorf("expected 1 match from primary, got %d", len(got))
	}
}

func TestWriteThroughMatchStore_ResetOnlyClearsPrimary(t *testing.T) {
	primary := repository.NewMemoryMatchStore()
	secondary := repository.NewMemoryMatchStore()
	s := repository.NewWriteThroughMatchStore(primary, secondary)

	matches := []models.Match{{ID: "m1", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas", Kickoff: time.Now()}}
	s.SaveAll(matches)
	s.Reset()

	got, _ := primary.GetAll()
	if len(got) != 0 {
		t.Errorf("expected primary cleared after Reset, got %d", len(got))
	}
	got2, _ := secondary.GetAll()
	if len(got2) != 1 {
		t.Errorf("expected secondary untouched after Reset, got %d", len(got2))
	}
}

func TestWriteThroughMatchStore_SaveAllContinuesWhenSecondaryFails(t *testing.T) {
	primary := repository.NewMemoryMatchStore()
	s := repository.NewWriteThroughMatchStore(primary, &failingMatchStore{})

	matches := []models.Match{{ID: "m1", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas", Kickoff: time.Now()}}
	if err := s.SaveAll(matches); err != nil {
		t.Errorf("expected SaveAll to succeed even when secondary fails, got %v", err)
	}
	got, _ := primary.GetAll()
	if len(got) != 1 {
		t.Errorf("expected primary updated despite secondary failure, got %d", len(got))
	}
}

func TestWriteThroughMatchStore_SaveAllReturnsErrorWhenPrimaryFails(t *testing.T) {
	s := repository.NewWriteThroughMatchStore(&failingMatchStore{}, repository.NewMemoryMatchStore())

	matches := []models.Match{{ID: "m1", HomeTeam: "Columbus Crew", AwayTeam: "FC Dallas", Kickoff: time.Now()}}
	if err := s.SaveAll(matches); err == nil {
		t.Errorf("expected error when primary fails, got nil")
	}
}

type failingMatchStore struct{}

func (f *failingMatchStore) SaveAll(_ []models.Match) error  { return fmt.Errorf("firestore down") }
func (f *failingMatchStore) GetAll() ([]models.Match, error) { return nil, nil }
func (f *failingMatchStore) Reset()                          {}
