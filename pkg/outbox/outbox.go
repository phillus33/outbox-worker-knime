package outbox

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/phillus33/outbox-worker-knime/internal/outbox"
)

type Outbox struct {
	store outbox.Store
}

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
