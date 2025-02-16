package outbox

import (
	"encoding/json"
	"time"
)

// MessageStatus represents the status of an outbox message.
type MessageStatus string

// MessageStatus values
const (
	StatusPending   MessageStatus = "pending"
	StatusPublished MessageStatus = "published"
	StatusFailed    MessageStatus = "failed"
)

// Message represents an outbox message with its metadata and payload.
// SequenceNumber is used to guarantee FIFO message delivery.
type Message struct {
	ID             int64           `json:"id"`
	Topic          string          `json:"topic"`
	Payload        json.RawMessage `json:"payload"`
	CreatedAt      time.Time       `json:"created_at"`
	PublishedAt    *time.Time      `json:"published_at,omitempty"`
	Status         MessageStatus   `json:"status"`
	SequenceNumber int64           `json:"sequence_number"`
}
