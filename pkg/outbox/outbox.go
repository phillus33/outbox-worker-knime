// Package outbox provides a transactional outbox pattern implementation
// that guarantees at-least-once message delivery with FIFO ordering
// using PostgreSQL and NATS. It ensures messages are only published
// after their associated transactions are committed.

package outbox

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/phillus33/outbox-worker-knime/internal/outbox"
)

// Outbox represents the transactional outbox pattern implementation.
// It provides methods to create and send messages within transactions.
type Outbox struct {
	store outbox.Store
}

// NewOutbox creates a new Outbox instance
func NewOutbox(store outbox.Store) *Outbox {
	return &Outbox{
		store: store,
	}
}

// SendMessage sends a message within a transaction
func (o *Outbox) SendMessage(ctx context.Context, tx *sql.Tx, topic string, payload interface{}) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	_, err = o.store.CreateMessage(ctx, tx, topic, payloadBytes)
	return err
}
