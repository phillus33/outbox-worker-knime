package outbox

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
)

type mockStore struct {
	messages  []*Message
	published []int64
	failed    []int64
}

func (m *mockStore) CreateMessage(ctx context.Context, tx *sql.Tx, topic string, payload json.RawMessage) (*Message, error) {
	msg := &Message{
		ID:             int64(len(m.messages) + 1),
		Topic:          topic,
		Payload:        payload,
		CreatedAt:      time.Now(),
		Status:         StatusPending,
		SequenceNumber: int64(len(m.messages) + 1),
	}
	m.messages = append(m.messages, msg)
	log.Printf("Created message: %+v", msg)
	return msg, nil
}

func (m *mockStore) GetPendingMessages(ctx context.Context, batchSize int) ([]*Message, error) {
	var pendingMessages []*Message
	for _, msg := range m.messages {
		// Only include messages that haven't been published
		isPublished := false
		for _, pubID := range m.published {
			if msg.ID == pubID {
				isPublished = true
				break
			}
		}
		if !isPublished {
			pendingMessages = append(pendingMessages, msg)
		}
	}
	return pendingMessages, nil
}

func (m *mockStore) MarkAsPublished(ctx context.Context, id int64) error {
	m.published = append(m.published, id)
	return nil
}

func (m *mockStore) MarkAsFailed(ctx context.Context, id int64) error {
	m.failed = append(m.failed, id)
	return nil
}

func TestWorker(t *testing.T) {
	// Setup NATS
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		t.Skipf("Skipping test, NATS not available: %v", err)
		return
	}
	defer nc.Close()

	store := &mockStore{
		messages: []*Message{
			{
				ID:             1,
				Topic:          "test.topic",
				Payload:        json.RawMessage(`{"test":"data"}`),
				Status:         StatusPending,
				SequenceNumber: 1,
			},
		},
	}

	worker := NewWorker(WorkerConfig{
		Store:        store,
		NatsConn:     nc,
		PollInterval: 100 * time.Millisecond,
		BatchSize:    10,
		IsLeader:     func() bool { return true },
	})

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := worker.Start(ctx); err != nil {
		t.Fatalf("Failed to start worker: %v", err)
	}

	// Wait for processing
	time.Sleep(200 * time.Millisecond)
	worker.Stop()

	if len(store.published) != 1 {
		t.Errorf("Expected 1 published message, got %d", len(store.published))
	}
}
