package config

import (
	"os"
	"strconv"
)

// Config holds all runtime configuration loaded from environment variables.
// This makes the worker 12-factor app compliant and easy to deploy in Docker/K8s.
type Config struct {
	// Redis connection settings
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// Queue settings — must match Laravel's queue name in config/queue.php
	QueueName string // default: "default"
	DLQName   string // dead letter queue name

	// Worker pool settings
	WorkerCount int // number of concurrent goroutines
	MaxRetries  int // max attempts before sending job to DLQ

	// Health server
	HealthPort string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),

		QueueName: getEnv("QUEUE_NAME", "default"),
		DLQName:   getEnv("DLQ_NAME", "failed"),

		WorkerCount: getEnvInt("WORKER_COUNT", 10),
		MaxRetries:  getEnvInt("MAX_RETRIES", 3),

		HealthPort: getEnv("HEALTH_PORT", "8080"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
