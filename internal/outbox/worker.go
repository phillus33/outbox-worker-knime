package outbox

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

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

type WorkerConfig struct {
	Store        Store
	NatsConn     *nats.Conn
	PollInterval time.Duration
	BatchSize    int
	IsLeader     func() bool
}

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

func (w *Worker) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.running {
		return
	}

	close(w.stopCh)
	w.running = false
}

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

func (w *Worker) processMessages(ctx context.Context) error {
	messages, err := w.store.GetPendingMessages(ctx, w.batchSize)
	if err != nil {
		return err
	}

	for _, msg := range messages {
		if err := w.publishMessage(ctx, msg); err != nil {
			log.Printf("Failed to publish message %d: %v", msg.ID, err)
			if err := w.store.MarkAsFailed(ctx, msg.ID); err != nil {
				log.Printf("Failed to mark message %d as failed: %v", msg.ID, err)
			}
			continue
		}

		if err := w.store.MarkAsPublished(ctx, msg.ID); err != nil {
			log.Printf("Failed to mark message %d as published: %v", msg.ID, err)
		}
	}

	return nil
}

func (w *Worker) publishMessage(ctx context.Context, msg *Message) error {
	return w.natsConn.Publish(msg.Topic, msg.Payload)
}
