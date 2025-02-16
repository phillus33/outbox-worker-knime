package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Database struct {
		DSN            string
		MaxConnections int
	}
	NATS struct {
		URL           string
		MaxReconnects int
	}
	Worker struct {
		PollInterval time.Duration
		BatchSize    int
		MaxRetries   int
		RetryBackoff time.Duration
	}
}

func LoadConfig() (*Config, error) {
	// Load from environment variables with sensible defaults
	cfg := &Config{}
	cfg.Database.DSN = getEnv("DATABASE_URL", "postgres://outbox:outbox@localhost:5432/outbox?sslmode=disable")
	cfg.Database.MaxConnections = getEnvInt("DATABASE_MAX_CONNECTIONS", 10)
	cfg.NATS.URL = getEnv("NATS_URL", "nats://localhost:4222")
	cfg.Worker.PollInterval = getEnvDuration("WORKER_POLL_INTERVAL", 1*time.Second)
	cfg.Worker.BatchSize = getEnvInt("WORKER_BATCH_SIZE", 100)
	cfg.Worker.MaxRetries = getEnvInt("WORKER_MAX_RETRIES", 3)
	return cfg, nil
}

// Helper functions to handle defaults
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	value, err := time.ParseDuration(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
} 