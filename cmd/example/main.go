package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/lib/pq"
	"github.com/nats-io/nats.go"
	"github.com/phillus33/outbox-worker-knime/internal/config"
	"github.com/phillus33/outbox-worker-knime/internal/leader"
	"github.com/phillus33/outbox-worker-knime/internal/outbox"
	outboxpkg "github.com/phillus33/outbox-worker-knime/pkg/outbox"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Setup logging with instance ID
	instanceID := os.Getenv("INSTANCE_ID")
	if instanceID == "" {
		instanceID = "1"
	}
	log.SetPrefix(fmt.Sprintf("[Instance %s] ", instanceID))

	// Connect to PostgreSQL with configured settings
	db, err := sql.Open("postgres", cfg.Database.DSN)
	if err != nil {
		log.Fatal(err)
	}
	db.SetMaxOpenConns(cfg.Database.MaxConnections)
	defer db.Close()

	// Connect to NATS with configured settings
	nc, err := nats.Connect(cfg.NATS.URL, nats.MaxReconnects(cfg.NATS.MaxReconnects))
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	// Create store
	store := outbox.NewPostgresStore(db)

	// Create leader election
	election := leader.NewElection()

	// Create and start worker with configured settings
	worker := outbox.NewWorker(outbox.WorkerConfig{
		Store:        store,
		NatsConn:     nc,
		PollInterval: cfg.Worker.PollInterval,
		BatchSize:    cfg.Worker.BatchSize,
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
