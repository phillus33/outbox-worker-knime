CREATE TABLE IF NOT EXISTS outbox_messages (
    id BIGSERIAL PRIMARY KEY,
    topic VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at TIMESTAMP,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    sequence_number BIGINT NOT NULL,
    CONSTRAINT unique_seq_num UNIQUE (sequence_number)
);

CREATE INDEX idx_status_created_at ON outbox_messages(status, created_at); 