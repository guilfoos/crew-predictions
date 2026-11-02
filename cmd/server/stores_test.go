package main

import (
	"context"
	"testing"

	"github.com/guilfoos/crew-predictions/internal/repository"
)

func TestBuildStores_TestModeUsesAllInMemory(t *testing.T) {
	cfg := Config{TestMode: true, FirestoreProject: ""}
	stores, err := buildStores(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Each store should be a memory implementation. We confirm this by type
	// assertion — production builds would return Firestore implementations.
	if _, ok := stores.User.(*repository.MemoryUserStore); !ok {
		t.Errorf("expected MemoryUserStore, got %T", stores.User)
	}
	if _, ok := stores.Prediction.(*repository.MemoryPredictionStore); !ok {
		t.Errorf("expected MemoryPredictionStore, got %T", stores.Prediction)
	}
	if _, ok := stores.Result.(*repository.MemoryResultStore); !ok {
		t.Errorf("expected MemoryResultStore, got %T", stores.Result)
	}
	if _, ok := stores.Match.(*repository.MemoryMatchStore); !ok {
		t.Errorf("expected MemoryMatchStore, got %T", stores.Match)
	}
	if stores.MemMatch == nil {
		t.Error("expected MemMatch to be populated even in test mode")
	}
}

func TestBuildStores_NoCloudProjectAlsoUsesInMemory(t *testing.T) {
	// Even in non-test mode, an empty FirestoreProject means in-memory.
	cfg := Config{TestMode: false, FirestoreProject: ""}
	stores, err := buildStores(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := stores.User.(*repository.MemoryUserStore); !ok {
		t.Errorf("expected MemoryUserStore when no FirestoreProject, got %T", stores.User)
	}
}

func TestBuildStores_PopulatesSeasonAndConfigStores(t *testing.T) {
	stores, err := buildStores(context.Background(), Config{TestMode: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stores.Season == nil {
		t.Error("expected Season store populated")
	}
	if stores.ConfigStore == nil {
		t.Error("expected ConfigStore populated")
	}
}
