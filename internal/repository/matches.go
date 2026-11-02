package repository

import (
	"sync"

	"github.com/guilfoos/crew-predictions/internal/models"
)

type MatchStore interface {
	GetAll() ([]models.Match, error)
	SaveAll(matches []models.Match) error
	Reset()
}

type MemoryMatchStore struct {
	mu      sync.RWMutex
	matches []models.Match
}

func NewMemoryMatchStore() *MemoryMatchStore {
	return &MemoryMatchStore{}
}

func (s *MemoryMatchStore) Seed(matches []models.Match) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.matches = append(s.matches, matches...)
}

func (s *MemoryMatchStore) SaveAll(matches []models.Match) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.matches = matches
	return nil
}

func (s *MemoryMatchStore) GetAll() ([]models.Match, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]models.Match, len(s.matches))
	copy(out, s.matches)
	return out, nil
}

func (s *MemoryMatchStore) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.matches = nil
}
