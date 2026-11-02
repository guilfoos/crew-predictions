package repository

import (
	"log"

	"github.com/guilfoos/crew-predictions/internal/models"
)

type writeThroughMatchStore struct {
	primary   MatchStore
	secondary MatchStore
}

func NewWriteThroughMatchStore(primary, secondary MatchStore) MatchStore {
	return &writeThroughMatchStore{primary: primary, secondary: secondary}
}

func (s *writeThroughMatchStore) SaveAll(matches []models.Match) error {
	if err := s.primary.SaveAll(matches); err != nil {
		return err
	}
	if err := s.secondary.SaveAll(matches); err != nil {
		log.Printf("write-through secondary SaveAll failed: %v", err)
	}
	return nil
}

func (s *writeThroughMatchStore) GetAll() ([]models.Match, error) {
	return s.primary.GetAll()
}

func (s *writeThroughMatchStore) Reset() {
	s.primary.Reset()
}
