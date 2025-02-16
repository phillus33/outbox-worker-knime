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

	messageReceived := make(chan bool)
	_, err = nc.Subscribe("test.topic", func(msg *nats.Msg) {
		var payload map[string]interface{}
		if err := json.Unmarshal(msg.Data, &payload); err != nil {
			t.Errorf("Failed to unmarshal message: %v", err)
			return
		}
		if payload["test"] == "integration" {
			messageReceived <- true
		}
	})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := worker.Start(ctx); err != nil {
		t.Fatalf("Failed to start worker: %v", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	testMessage := map[string]string{"test": "integration"}
	err = ob.SendMessage(ctx, tx, "test.topic", testMessage)
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to send message: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}

	select {
	case <-messageReceived:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for message")
	}

	worker.Stop()
}
