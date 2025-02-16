package outbox

import (
	"database/sql"
	"testing"

	_ "github.com/lib/pq"
)

func SetupTestDB(t *testing.T) *sql.DB {
	// Connect to the default database to create the test database
	db, err := sql.Open("postgres", "host=localhost port=5432 user=outbox password=outbox dbname=postgres sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Drop the test database if it exists
	_, err = db.Exec(`DROP DATABASE IF EXISTS outbox_test`)
	if err != nil {
		t.Fatalf("Failed to drop database: %v", err)
	}

	// Create the test database
	_, err = db.Exec(`CREATE DATABASE outbox_test`)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	// Connect to the test database
	db.Close() // Close the connection to the default database
	db, err = sql.Open("postgres", "host=localhost port=5432 user=outbox password=outbox dbname=outbox_test sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Create necessary tables for testing
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS outbox_messages (
		id SERIAL PRIMARY KEY,
		topic VARCHAR(255) NOT NULL,
		payload BYTEA NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		published_at TIMESTAMP,
		status VARCHAR(50) NOT NULL,
		sequence_number INT NOT NULL
	)`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	return db
}


