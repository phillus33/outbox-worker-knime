package outbox

import (
	"context"
	"encoding/json"
	"testing"
)

func TestPostgresStore_CreateMessage(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Close()

	store := NewPostgresStore(db)
	ctx := context.Background()

	t.Run("successful message creation", func(t *testing.T) {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}
		defer tx.Rollback()

		payload := json.RawMessage(`{"test":"data"}`)
		msg, err := store.CreateMessage(ctx, tx, "test.topic", payload)
		if err != nil {
			t.Fatalf("Failed to create message: %v", err)
		}

		if msg.ID == 0 {
			t.Error("Expected non-zero ID")
		}
		if msg.SequenceNumber != 1 {
			t.Errorf("Expected sequence number 1, got %d", msg.SequenceNumber)
		}
		if msg.Status != StatusPending {
			t.Errorf("Expected status pending, got %s", msg.Status)
		}

		if err := tx.Commit(); err != nil {
			t.Fatalf("Failed to commit transaction: %v", err)
		}
	})
}

func TestPostgresStore_GetPendingMessages(t *testing.T) {
	db := SetupTestDB(t)
	defer db.Close()

	store := NewPostgresStore(db)
	ctx := context.Background()

	// Create test messages
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	payload := json.RawMessage(`{"test":"data"}`)
	for i := 0; i < 3; i++ {
		_, err := store.CreateMessage(ctx, tx, "test.topic", payload)
		if err != nil {
			t.Fatalf("Failed to create message: %v", err)
		}
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}

	t.Run("fetch pending messages", func(t *testing.T) {
		messages, err := store.GetPendingMessages(ctx, 10)
		if err != nil {
			t.Fatalf("Failed to get pending messages: %v", err)
		}

		if len(messages) != 3 {
			t.Errorf("Expected 3 messages, got %d", len(messages))
		}

		// Verify sequence
		for i, msg := range messages {
			if msg.SequenceNumber != int64(i+1) {
				t.Errorf("Expected sequence number %d, got %d", i+1, msg.SequenceNumber)
			}
		}
	})
}
