package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestChirpJSONMarshaling(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	chirp := Chirp{
		ID:        uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
		CreatedAt: now,
		UpdatedAt: now,
		Body:      "Hello, world!",
		UserID:    uuid.MustParse("987e6543-e21b-12d3-a456-426614174999"),
	}

	data, err := json.Marshal(chirp)
	if err != nil {
		t.Fatalf("Failed to marshal Chirp: %v", err)
	}

	var unmarshaled Chirp
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal Chirp: %v", err)
	}

	if chirp.ID != unmarshaled.ID {
		t.Errorf("ID mismatch: got %v, want %v", unmarshaled.ID, chirp.ID)
	}
	if !chirp.CreatedAt.Equal(unmarshaled.CreatedAt) {
		t.Errorf("CreatedAt mismatch: got %v, want %v", unmarshaled.CreatedAt, chirp.CreatedAt)
	}
	if !chirp.UpdatedAt.Equal(unmarshaled.UpdatedAt) {
		t.Errorf("UpdatedAt mismatch: got %v, want %v", unmarshaled.UpdatedAt, chirp.UpdatedAt)
	}
	if chirp.Body != unmarshaled.Body {
		t.Errorf("Body mismatch: got %q, want %q", unmarshaled.Body, chirp.Body)
	}
	if chirp.UserID != unmarshaled.UserID {
		t.Errorf("UserID mismatch: got %v, want %v", unmarshaled.UserID, chirp.UserID)
	}
}
