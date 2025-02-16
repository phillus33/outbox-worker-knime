// Package outbox implements the core worker functionality for the transactional
// outbox pattern. It handles message polling, leader election, and guaranteed
// FIFO delivery of messages to NATS.

package outbox

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

// Worker handles the polling and publishing of messages from the outbox.
// It ensures FIFO delivery and supports leader election for distributed deployments.
type Worker struct {
	store        Store
	natsConn     *nats.Conn
	pollInterval time.Duration
	batchSize    int
	isLeader     func() bool
	mu           sync.Mutex
	running      bool
	stopCh       chan struct{}
}

// WorkerConfig provides configuration options for the Worker.
type WorkerConfig struct {
	Store        Store
	NatsConn     *nats.Conn
	PollInterval time.Duration
	BatchSize    int
	IsLeader     func() bool
}

// NewWorker creates a new Worker instance with the given configuration
// and initializes the stopCh channel for signaling when to stop the worker.
func NewWorker(config WorkerConfig) *Worker {
	return &Worker{
		store:        config.Store,
		natsConn:     config.NatsConn,
		pollInterval: config.PollInterval,
		batchSize:    config.BatchSize,
		isLeader:     config.IsLeader,
		stopCh:       make(chan struct{}),
	}
}

// Start starts the worker and ensures thread safety for the running state.
// Starts a new goroutine to run the worker loop.
func (w *Worker) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return nil
	}
	w.running = true
	w.mu.Unlock()

	go w.run(ctx)
	return nil
}

// Stop gracefully stops the worker by closing the stop channel
// and setting the running state to false.
func (w *Worker) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.running {
		return
	}

	close(w.stopCh)
	w.running = false
}

// run is the main worker loop that polls for messages and processes them.
// It uses a ticker to poll for messages at the specified interval.
// It listens to three kinds of events:
// - context done: if the context is canceled, the worker stops
// - stop channel: if the stop channel is closed, the worker stops
// - ticker firing: if the ticker fires, the worker polls for messages unless it is not the leader
func (w *Worker) run(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			if !w.isLeader() {
				continue
			}

			if err := w.processMessages(ctx); err != nil {
				log.Printf("Error processing messages: %v", err)
			}
		}
	}
}

// processMessages retrieves and processes pending messages from the store.
// It ensures sequence number consistency and publishes messages to NATS.
// It returns an error if there is a sequence number gap or if the message fails to publish.
func (w *Worker) processMessages(ctx context.Context) error {
	messages, err := w.store.GetPendingMessages(ctx, w.batchSize)
	if err != nil {
		return err
	}

	var lastProcessedSeq int64
	for _, msg := range messages {
		if lastProcessedSeq > 0 && msg.SequenceNumber != lastProcessedSeq+1 {
			return ErrSequenceGap
		}

		if err := w.publishMessage(ctx, msg); err != nil {
			log.Printf("Failed to publish message %d: %v", msg.ID, err)
			return err // Return immediately to maintain FIFO
		}

		if err := w.store.MarkAsPublished(ctx, msg.ID); err != nil {
			log.Printf("Failed to mark message %d as published: %v", msg.ID, err)
			return err // Return immediately to maintain FIFO
		}

		lastProcessedSeq = msg.SequenceNumber
	}
	return nil
}

// publishMessage attempts to publish a message to NATS with exponential backoff.
// It retries up to 3 times if the message fails to publish.
func (w *Worker) publishMessage(ctx context.Context, msg *Message) error {
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		err := w.natsConn.Publish(msg.Topic, msg.Payload)
		if err == nil {
			return nil
		}
		time.Sleep(time.Second * time.Duration(i+1))
	}
	return fmt.Errorf("failed to publish after %d retries", maxRetries)
}
