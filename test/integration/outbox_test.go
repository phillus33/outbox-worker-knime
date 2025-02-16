package integration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/phillus33/outbox-worker-knime/internal/outbox"
	outboxpkg "github.com/phillus33/outbox-worker-knime/pkg/outbox"
)

func TestOutboxIntegration(t *testing.T) {
	// Setup database
	db := outbox.SetupTestDB(t)
	defer db.Close()

	// Setup NATS
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		t.Skipf("Skipping test, NATS not available: %v", err)
		return
	}
	defer nc.Close()

	// Setup components
	store := outbox.NewPostgresStore(db)
	ob := outboxpkg.NewOutbox(store)

	worker := outbox.NewWorker(outbox.WorkerConfig{
		Store:        store,
		NatsConn:     nc,
		PollInterval: 100 * time.Millisecond,
		BatchSize:    10,
		IsLeader:     func() bool { return true },
	})

	// Setup message receiving
	receivedCh := make(chan []byte, 1)
	_, err = nc.Subscribe("test.topic", func(msg *nats.Msg) {
		receivedCh <- msg.Data
	})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start worker
	if err := worker.Start(ctx); err != nil {
		t.Fatalf("Failed to start worker: %v", err)
	}

	// Send message
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	testMessage := map[string]string{"test": "integration"}
	err = ob.SendMessage(ctx, tx, "test.topic", testMessage)
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}

	// Wait for message
	select {
	case received := <-receivedCh:
		var receivedMsg map[string]string
		if err := json.Unmarshal(received, &receivedMsg); err != nil {
			t.Fatalf("Failed to unmarshal received message: %v", err)
		}
		if receivedMsg["test"] != "integration" {
			t.Errorf("Expected message content 'integration', got '%s'", receivedMsg["test"])
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for message")
	}

	worker.Stop()
}
