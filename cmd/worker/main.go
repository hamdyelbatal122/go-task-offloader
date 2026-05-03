// Go-Sidecar-Worker — entry point.
//
// Boot sequence:
//  1. Load config from environment variables.
//  2. Connect to Redis and verify with PING.
//  3. Build the handler registry (one entry per Laravel job class).
//  4. Start the worker pool (N goroutines polling BLPOP concurrently).
//  5. Start the health/metrics HTTP server.
//  6. Block until SIGTERM or SIGINT.
//  7. Cancel the context → all workers finish their current job, then exit.
//  8. Wait for the pool to drain, then shut down HTTP and close Redis.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/go-sidecar-worker/internal/config"
	"github.com/go-sidecar-worker/internal/handlers"
	"github.com/go-sidecar-worker/internal/health"
	"github.com/go-sidecar-worker/internal/queue"
	"github.com/go-sidecar-worker/internal/worker"
)

func main() {
	// ── 1. Structured logger ─────────────────────────────────────────────────
	logger, err := zap.NewProduction()
	if err != nil {
		// Fallback: if zap itself fails to initialise, panic is acceptable here.
		panic("failed to initialise logger: " + err.Error())
	}
	defer logger.Sync() //nolint:errcheck

	// ── 2. Configuration ─────────────────────────────────────────────────────
	cfg := config.Load()
	logger.Info("configuration loaded",
		zap.String("redis_addr", cfg.RedisAddr),
		zap.String("queue", cfg.QueueName),
		zap.Int("workers", cfg.WorkerCount),
		zap.Int("max_retries", cfg.MaxRetries),
	)

	// ── 3. Redis connection ──────────────────────────────────────────────────
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,

		// Connection pool settings tuned for a high-concurrency worker.
		PoolSize:     cfg.WorkerCount + 5,
		MinIdleConns: cfg.WorkerCount / 2,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 5 * time.Second,
	})

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()

	if err := rdb.Ping(pingCtx).Err(); err != nil {
		logger.Fatal("redis ping failed — check REDIS_ADDR and credentials", zap.Error(err))
		os.Exit(1)
	}
	logger.Info("redis connected", zap.String("addr", cfg.RedisAddr))

	// ── 4. Handler registry ──────────────────────────────────────────────────
	// Register one handler per Laravel job class name.
	// The key must exactly match the "displayName" field in the JSON payload.
	registry := handlers.NewRegistry()
	registry.Register("App\\Jobs\\ProcessImageJob", handlers.NewImageProcessor(logger))
	registry.Register("App\\Jobs\\CrunchDataJob", handlers.NewDataCruncher(logger))
	// Add more handlers here as your Laravel application grows:
	// registry.Register("App\\Jobs\\SendReportJob", handlers.NewReportSender(logger))

	// ── 5. Consumer + Worker pool ────────────────────────────────────────────
	consumer := queue.NewConsumer(rdb, cfg.QueueName, cfg.DLQName, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool := worker.NewPool(cfg, consumer, registry, logger)
	pool.Start(ctx)

	// ── 6. Health / metrics HTTP server ──────────────────────────────────────
	healthSrv := health.NewServer(cfg.HealthPort, logger)
	healthSrv.Start()

	logger.Info("go-sidecar-worker is running — waiting for jobs")

	// ── 7. Graceful shutdown on SIGTERM / SIGINT ──────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	sig := <-quit
	logger.Info("shutdown signal received",
		zap.String("signal", sig.String()),
		zap.String("action", "draining workers"),
	)

	// Cancel the context: BLPOP will unblock after its 5-second timeout and
	// each goroutine will check ctx.Done() before looping again.
	cancel()

	// Wait for every goroutine to finish its current job.
	pool.Wait()

	// Shut down the HTTP server gracefully.
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutCancel()
	healthSrv.Stop(shutCtx)

	// Close the Redis connection pool.
	if err := rdb.Close(); err != nil {
		logger.Error("redis close error", zap.Error(err))
	}

	logger.Info("go-sidecar-worker stopped cleanly")
}
