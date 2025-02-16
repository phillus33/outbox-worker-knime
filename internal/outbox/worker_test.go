package outbox

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"sync"
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

func TestFIFOMessageDelivery(t *testing.T) {
	store := &mockStore{}
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		t.Skip("NATS not available")
	}
	defer nc.Close()

	// Create worker
	worker := NewWorker(WorkerConfig{
		Store:        store,
		NatsConn:     nc,
		PollInterval: 10 * time.Millisecond,
		BatchSize:    10,
		IsLeader:     func() bool { return true },
	})

	// Track received messages
	receivedMessages := make([]int64, 0)
	var mu sync.Mutex

	// Subscribe to test topic
	_, err = nc.Subscribe("test.topic", func(msg *nats.Msg) {
		var payload struct{ Seq int64 }
		json.Unmarshal(msg.Data, &payload)
		mu.Lock()
		receivedMessages = append(receivedMessages, payload.Seq)
		mu.Unlock()
	})
	if err != nil {
		t.Fatal(err)
	}

	// Send messages with sequence numbers
	ctx := context.Background()
	for i := int64(1); i <= 5; i++ {
		payload := struct{ Seq int64 }{Seq: i}
		data, _ := json.Marshal(payload)
		store.messages = append(store.messages, &Message{
			ID:             i,
			Topic:          "test.topic",
			Payload:        data,
			SequenceNumber: i,
			Status:         StatusPending,
		})
	}

	// Start worker and wait for processing
	worker.Start(ctx)
	time.Sleep(100 * time.Millisecond)
	worker.Stop()

	// Verify order
	mu.Lock()
	defer mu.Unlock()
	for i, seq := range receivedMessages {
		if int64(i+1) != seq {
			t.Errorf("Message received out of order. Expected %d, got %d", i+1, seq)
		}
	}
}

func TestAtLeastOnceDelivery(t *testing.T) {
	store := &mockStore{}
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		t.Skip("NATS not available")
	}
	defer nc.Close()

	// Create worker with retry
	worker := NewWorker(WorkerConfig{
		Store:        store,
		NatsConn:     nc,
		PollInterval: 10 * time.Millisecond,
		BatchSize:    10,
		IsLeader:     func() bool { return true },
	})

	// Count message deliveries
	deliveryCount := make(map[int64]int)
	var mu sync.Mutex

	// Subscribe to test topic
	_, err = nc.Subscribe("test.topic", func(msg *nats.Msg) {
		var payload struct{ ID int64 }
		json.Unmarshal(msg.Data, &payload)
		mu.Lock()
		deliveryCount[payload.ID]++
		mu.Unlock()
	})
	if err != nil {
		t.Fatal(err)
	}

	// Create test message
	store.messages = append(store.messages, &Message{
		ID:             1,
		Topic:          "test.topic",
		Payload:        []byte(`{"ID": 1}`),
		SequenceNumber: 1,
		Status:         StatusPending,
	})

	// Start worker and wait for processing
	ctx := context.Background()
	worker.Start(ctx)
	time.Sleep(100 * time.Millisecond)
	worker.Stop()

	// Verify delivery
	mu.Lock()
	defer mu.Unlock()
	if count := deliveryCount[1]; count < 1 {
		t.Errorf("Message was not delivered at least once: got %d deliveries", count)
	}
}

func TestLeaderElectionProcessing(t *testing.T) {
	store := &mockStore{}
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		t.Skip("NATS not available")
	}
	defer nc.Close()

	isLeader := false
	worker := NewWorker(WorkerConfig{
		Store:        store,
		NatsConn:     nc,
		PollInterval: 10 * time.Millisecond,
		BatchSize:    10,
		IsLeader:     func() bool { return isLeader },
	})

	messageProcessed := false
	_, err = nc.Subscribe("test.topic", func(msg *nats.Msg) {
		messageProcessed = true
	})
	if err != nil {
		t.Fatal(err)
	}

	// Add test message
	store.messages = append(store.messages, &Message{
		ID:             1,
		Topic:          "test.topic",
		Payload:        []byte(`{"test":"data"}`),
		SequenceNumber: 1,
		Status:         StatusPending,
	})

	ctx := context.Background()
	worker.Start(ctx)

	// Wait with worker not being leader
	time.Sleep(50 * time.Millisecond)
	if messageProcessed {
		t.Error("Message was processed while not leader")
	}

	// Make worker leader
	isLeader = true
	time.Sleep(50 * time.Millisecond)

	if !messageProcessed {
		t.Error("Message was not processed while leader")
	}

	worker.Stop()
}
