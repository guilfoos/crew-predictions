package repository

import (
	"context"
	"fmt"

	"github.com/guilfoos/crew-predictions/internal/models"
)

// ErrorPredictionStore is a test double that always fails on Save.
type ErrorPredictionStore struct{}

func NewErrorPredictionStore() *ErrorPredictionStore { return &ErrorPredictionStore{} }

func (e *ErrorPredictionStore) Save(_ context.Context, _ Prediction) error {
	return fmt.Errorf("simulated store failure")
}

func (e *ErrorPredictionStore) GetByMatchAndUser(_ context.Context, _, _ string) (*Prediction, error) {
	return nil, nil
}

func (e *ErrorPredictionStore) GetByMatch(_ context.Context, _ string) ([]Prediction, error) {
	return nil, nil
}

func (e *ErrorPredictionStore) GetAll(_ context.Context) ([]Prediction, error) {
	return nil, nil
}

// GetByMatchAndUserErrorPredictionStore delegates to a MemoryPredictionStore for all
// methods except GetByMatchAndUser, which always fails. This lets tests verify that
// the bot skips prediction saves without losing access to GetByMatch/GetAll.
type GetByMatchAndUserErrorPredictionStore struct {
	*MemoryPredictionStore
}

func NewGetByMatchAndUserErrorPredictionStore() *GetByMatchAndUserErrorPredictionStore {
	return &GetByMatchAndUserErrorPredictionStore{NewMemoryPredictionStore()}
}

func (e *GetByMatchAndUserErrorPredictionStore) GetByMatchAndUser(_ context.Context, _, _ string) (*Prediction, error) {
	return nil, fmt.Errorf("simulated GetByMatchAndUser failure")
}

// ErrorGetAllPredictionStore is a test double that always fails on GetAll.
type ErrorGetAllPredictionStore struct{}

func NewErrorGetAllPredictionStore() *ErrorGetAllPredictionStore {
	return &ErrorGetAllPredictionStore{}
}

func (e *ErrorGetAllPredictionStore) Save(_ context.Context, _ Prediction) error { return nil }

func (e *ErrorGetAllPredictionStore) GetByMatchAndUser(_ context.Context, _, _ string) (*Prediction, error) {
	return nil, nil
}

func (e *ErrorGetAllPredictionStore) GetByMatch(_ context.Context, _ string) ([]Prediction, error) {
	return nil, nil
}

func (e *ErrorGetAllPredictionStore) GetAll(_ context.Context) ([]Prediction, error) {
	return nil, fmt.Errorf("simulated GetAll failure")
}

// ErrorUpsertUserStore is a test double that always fails on Upsert.
type ErrorUpsertUserStore struct{}

func NewErrorUpsertUserStore() *ErrorUpsertUserStore { return &ErrorUpsertUserStore{} }

func (e *ErrorUpsertUserStore) Upsert(_ context.Context, _ User) error {
	return fmt.Errorf("simulated upsert failure")
}

func (e *ErrorUpsertUserStore) GetByID(_ context.Context, _ string) (*User, error) {
	return nil, nil
}

func (e *ErrorUpsertUserStore) GetAll(_ context.Context) ([]User, error) {
	return nil, nil
}

func (e *ErrorUpsertUserStore) UpdateScores(_ context.Context, _ string, _, _, _, _ int) error {
	return fmt.Errorf("simulated UpdateScores failure")
}

// ErrorGetAllUserStore is a test double that always fails on GetAll.
type ErrorGetAllUserStore struct{}

func NewErrorGetAllUserStore() *ErrorGetAllUserStore { return &ErrorGetAllUserStore{} }

func (e *ErrorGetAllUserStore) Upsert(_ context.Context, _ User) error { return nil }

func (e *ErrorGetAllUserStore) GetByID(_ context.Context, _ string) (*User, error) {
	return nil, nil
}

func (e *ErrorGetAllUserStore) GetAll(_ context.Context) ([]User, error) {
	return nil, fmt.Errorf("simulated GetAll failure")
}

func (e *ErrorGetAllUserStore) UpdateScores(_ context.Context, _ string, _, _, _, _ int) error {
	return nil
}

// ErrorUpsertWithUserStore returns one user from GetAll but always fails on Upsert.
// Use this when testing code that iterates users and then tries to write back.
type ErrorUpsertWithUserStore struct{}

func NewErrorUpsertWithUserStore() *ErrorUpsertWithUserStore { return &ErrorUpsertWithUserStore{} }

func (e *ErrorUpsertWithUserStore) Upsert(_ context.Context, _ User) error {
	return fmt.Errorf("simulated upsert failure")
}

func (e *ErrorUpsertWithUserStore) GetByID(_ context.Context, _ string) (*User, error) {
	return nil, nil
}

func (e *ErrorUpsertWithUserStore) GetAll(_ context.Context) ([]User, error) {
	return []User{{UserID: "u1", Handle: "Fan"}}, nil
}

func (e *ErrorUpsertWithUserStore) UpdateScores(_ context.Context, _ string, _, _, _, _ int) error {
	return fmt.Errorf("simulated UpdateScores failure")
}

// ErrorGetAllWithExistingUserStore returns a fixed user from GetByID but fails on GetAll.
// Use this to test handlers that call GetByID (succeeds) then GetAll (fails → 500).
type ErrorGetAllWithExistingUserStore struct{}

func NewErrorGetAllWithExistingUserStore() *ErrorGetAllWithExistingUserStore {
	return &ErrorGetAllWithExistingUserStore{}
}

func (e *ErrorGetAllWithExistingUserStore) Upsert(_ context.Context, _ User) error { return nil }

func (e *ErrorGetAllWithExistingUserStore) GetByID(_ context.Context, _ string) (*User, error) {
	u := User{UserID: "u1", Handle: "Fan"}
	return &u, nil
}

func (e *ErrorGetAllWithExistingUserStore) GetAll(_ context.Context) ([]User, error) {
	return nil, fmt.Errorf("simulated GetAll failure")
}

func (e *ErrorGetAllWithExistingUserStore) UpdateScores(_ context.Context, _ string, _, _, _, _ int) error {
	return nil
}

// ErrorGetByIDUserStore is a test double that always fails on GetByID.
type ErrorGetByIDUserStore struct{}

func NewErrorGetByIDUserStore() *ErrorGetByIDUserStore { return &ErrorGetByIDUserStore{} }

func (e *ErrorGetByIDUserStore) Upsert(_ context.Context, _ User) error { return nil }

func (e *ErrorGetByIDUserStore) GetByID(_ context.Context, _ string) (*User, error) {
	return nil, fmt.Errorf("simulated GetByID failure")
}

func (e *ErrorGetByIDUserStore) GetAll(_ context.Context) ([]User, error) { return nil, nil }

func (e *ErrorGetByIDUserStore) UpdateScores(_ context.Context, _ string, _, _, _, _ int) error {
	return nil
}

// ErrorSeasonStore is a test double that always fails on Save.
type ErrorSeasonStore struct{}

func NewErrorSeasonStore() *ErrorSeasonStore { return &ErrorSeasonStore{} }

func (e *ErrorSeasonStore) Save(_ context.Context, _ SeasonSnapshot) error {
	return fmt.Errorf("simulated season store failure")
}

func (e *ErrorSeasonStore) GetByID(_ context.Context, _ string) (*SeasonSnapshot, error) {
	return nil, nil
}

func (e *ErrorSeasonStore) ListAll(_ context.Context) ([]SeasonSnapshot, error) { return nil, nil }

func (e *ErrorSeasonStore) Exists(_ context.Context, _ string) bool { return false }

func (e *ErrorSeasonStore) Reset() {}

// ErrorGetByIDSeasonStore always fails on GetByID.
type ErrorGetByIDSeasonStore struct{}

func NewErrorGetByIDSeasonStore() *ErrorGetByIDSeasonStore { return &ErrorGetByIDSeasonStore{} }

func (e *ErrorGetByIDSeasonStore) Save(_ context.Context, _ SeasonSnapshot) error { return nil }

func (e *ErrorGetByIDSeasonStore) GetByID(_ context.Context, _ string) (*SeasonSnapshot, error) {
	return nil, fmt.Errorf("simulated season store GetByID failure")
}

func (e *ErrorGetByIDSeasonStore) ListAll(_ context.Context) ([]SeasonSnapshot, error) {
	return nil, nil
}

func (e *ErrorGetByIDSeasonStore) Exists(_ context.Context, _ string) bool { return false }

func (e *ErrorGetByIDSeasonStore) Reset() {}

// ErrorGetAllMatchStore is a test double that always fails on GetAll.
type ErrorGetAllMatchStore struct{}

func NewErrorGetAllMatchStore() *ErrorGetAllMatchStore { return &ErrorGetAllMatchStore{} }

func (e *ErrorGetAllMatchStore) GetAll() ([]models.Match, error) {
	return nil, fmt.Errorf("simulated GetAll failure")
}

func (e *ErrorGetAllMatchStore) SaveAll(_ []models.Match) error { return nil }

func (e *ErrorGetAllMatchStore) Reset() {}

// ErrorSaveAllMatchStore is a test double whose GetAll succeeds (returns a
// configurable seed) but whose SaveAll always fails. Use to test handlers
// that read the store, mutate the slice, and then write it back.
type ErrorSaveAllMatchStore struct {
	Matches []models.Match
}

func NewErrorSaveAllMatchStore(seed []models.Match) *ErrorSaveAllMatchStore {
	return &ErrorSaveAllMatchStore{Matches: seed}
}

func (e *ErrorSaveAllMatchStore) GetAll() ([]models.Match, error) {
	return e.Matches, nil
}

func (e *ErrorSaveAllMatchStore) SaveAll(_ []models.Match) error {
	return fmt.Errorf("simulated SaveAll failure")
}

func (e *ErrorSaveAllMatchStore) Reset() {}

// ErrorGetByMatchPredictionStore is a test double that always fails on GetByMatch.
type ErrorGetByMatchPredictionStore struct{}

func NewErrorGetByMatchPredictionStore() *ErrorGetByMatchPredictionStore {
	return &ErrorGetByMatchPredictionStore{}
}

func (e *ErrorGetByMatchPredictionStore) Save(_ context.Context, _ Prediction) error { return nil }

func (e *ErrorGetByMatchPredictionStore) GetByMatchAndUser(_ context.Context, _, _ string) (*Prediction, error) {
	return nil, nil
}

func (e *ErrorGetByMatchPredictionStore) GetByMatch(_ context.Context, _ string) ([]Prediction, error) {
	return nil, fmt.Errorf("simulated GetByMatch failure")
}

func (e *ErrorGetByMatchPredictionStore) GetAll(_ context.Context) ([]Prediction, error) {
	return nil, nil
}

// ErrorResultStore is a test double that always fails on SaveResult.
type ErrorResultStore struct{}

func NewErrorResultStore() *ErrorResultStore { return &ErrorResultStore{} }

func (e *ErrorResultStore) SaveResult(_ context.Context, _ Result) error {
	return fmt.Errorf("simulated result store failure")
}

func (e *ErrorResultStore) GetResult(_ context.Context, _ string) (*Result, error) {
	return nil, nil
}
