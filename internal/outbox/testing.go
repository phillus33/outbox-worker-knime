package outbox

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

func setupTestDB(t *testing.T) *sql.DB {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://outbox:outbox@localhost:5432/outbox?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Clean up existing data
	_, err = db.Exec("TRUNCATE TABLE outbox_messages")
	if err != nil {
		t.Fatalf("Failed to truncate table: %v", err)
	}

	return db
}
