package outbox

import (
	"encoding/json"
	"time"
)

type MessageStatus string

const (
	StatusPending   MessageStatus = "pending"
	StatusPublished MessageStatus = "published"
	StatusFailed    MessageStatus = "failed"
)

type Message struct {
	ID             int64           `json:"id"`
	Topic          string          `json:"topic"`
	Payload        json.RawMessage `json:"payload"`
	CreatedAt      time.Time       `json:"created_at"`
	PublishedAt    *time.Time      `json:"published_at,omitempty"`
	Status         MessageStatus   `json:"status"`
	SequenceNumber int64           `json:"sequence_number"`
}
