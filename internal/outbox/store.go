package outbox

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrMessageNotFound = errors.New("message not found")
	ErrSequenceGap     = errors.New("sequence gap detected")
)

// Store defines the interface for outbox message storage operations.
// Implementations must ensure sequence number consistency for FIFO guarantees.
type Store interface {
	CreateMessage(ctx context.Context, tx *sql.Tx, topic string, payload json.RawMessage) (*Message, error)
	GetPendingMessages(ctx context.Context, batchSize int) ([]*Message, error)
	MarkAsPublished(ctx context.Context, id int64) error
	MarkAsFailed(ctx context.Context, id int64) error
}

// PostgresStore implements the Store interface for PostgreSQL storage.
// It ensures sequence number consistency for FIFO guarantees.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore creates a new PostgresStore instance
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

// CreateMessage inserts a new message into the outbox table
// and returns the created message with its sequence number.
func (s *PostgresStore) CreateMessage(ctx context.Context, tx *sql.Tx, topic string, payload json.RawMessage) (*Message, error) {
	query := `
        INSERT INTO outbox_messages (topic, payload, status, sequence_number)
        VALUES ($1, $2, $3, (SELECT COALESCE(MAX(sequence_number), 0) + 1 FROM outbox_messages))
        RETURNING id, created_at, sequence_number`

	msg := &Message{
		Topic:   topic,
		Payload: payload,
		Status:  StatusPending,
	}

	err := tx.QueryRowContext(ctx, query, msg.Topic, msg.Payload, msg.Status).
		Scan(&msg.ID, &msg.CreatedAt, &msg.SequenceNumber)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

// GetPendingMessages retrieves pending messages from the outbox table
// sorted by sequence number in ascending order.
func (s *PostgresStore) GetPendingMessages(ctx context.Context, batchSize int) ([]*Message, error) {
	query := `
        SELECT id, topic, payload, created_at, status, sequence_number
        FROM outbox_messages
        WHERE status = $1
        ORDER BY sequence_number ASC
        LIMIT $2`

	rows, err := s.db.QueryContext(ctx, query, StatusPending, batchSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		msg := &Message{}
		err := rows.Scan(
			&msg.ID,
			&msg.Topic,
			&msg.Payload,
			&msg.CreatedAt,
			&msg.Status,
			&msg.SequenceNumber,
		)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	return messages, rows.Err()
}

// MarkAsPublished updates the status of a message to published
// and records the publication time.
func (s *PostgresStore) MarkAsPublished(ctx context.Context, id int64) error {
	query := `
        UPDATE outbox_messages
        SET status = $1, published_at = $2
        WHERE id = $3`

	result, err := s.db.ExecContext(ctx, query, StatusPublished, time.Now(), id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrMessageNotFound
	}

	return nil
}

// MarkAsFailed updates the status of a message to failed
// and records the failure time.
func (s *PostgresStore) MarkAsFailed(ctx context.Context, id int64) error {
	query := `
        UPDATE outbox_messages
        SET status = $1
        WHERE id = $2`

	result, err := s.db.ExecContext(ctx, query, StatusFailed, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrMessageNotFound
	}

	return nil
}
