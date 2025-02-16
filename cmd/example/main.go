package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/nats-io/nats.go"
	"github.com/phillus33/outbox-worker-knime/internal/leader"
	"github.com/phillus33/outbox-worker-knime/internal/outbox"
	outboxpkg "github.com/phillus33/outbox-worker-knime/pkg/outbox"
)

func main() {
	// Add instance ID for logging
	instanceID := os.Getenv("INSTANCE_ID")
	if instanceID == "" {
		instanceID = "1"
	}
	
	log.SetPrefix(fmt.Sprintf("[Instance %s] ", instanceID))
	

	// Connect to PostgreSQL
	db, err := sql.Open("postgres", "postgres://outbox:outbox@localhost:5432/outbox?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Connect to NATS
	nc, err := nats.Connect("nats://localhost:4222")
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	// Create store
	store := outbox.NewPostgresStore(db)

	// Create leader election
	election := leader.NewElection()

	// Create and start worker
	worker := outbox.NewWorker(outbox.WorkerConfig{
		Store:        store,
		NatsConn:     nc,
		PollInterval: 1 * time.Second,
		BatchSize:    100,
		IsLeader:     election.IsLeader,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := worker.Start(ctx); err != nil {
		log.Fatal(err)
	}

	// Create outbox instance for sending messages
	ob := outboxpkg.NewOutbox(store)

	// Example: Send a message within a transaction
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	message := map[string]string{
		"hello": "world",
	}

	err = ob.SendMessage(ctx, tx, "example.topic", message)
	if err != nil {
		tx.Rollback()
		log.Fatal(err)
	}

	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	// Graceful shutdown
	worker.Stop()
}
