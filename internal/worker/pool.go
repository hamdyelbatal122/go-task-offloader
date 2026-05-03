// Package worker implements a fixed-size goroutine pool that concurrently
// consumes jobs from Redis and dispatches them to registered handlers.
package worker

import (
	"context"
	"sync"

	"go.uber.org/zap"

	"github.com/hamdyelbatal122/go-task-offloader/internal/config"
	"github.com/hamdyelbatal122/go-task-offloader/internal/handlers"
	"github.com/hamdyelbatal122/go-task-offloader/internal/job"
	"github.com/hamdyelbatal122/go-task-offloader/internal/queue"
)

// Pool manages a fixed set of goroutines, each independently polling Redis.
// This pattern avoids channel bottlenecks — every goroutine calls BLPOP
// directly, which Redis handles efficiently with O(1) complexity.
type Pool struct {
	cfg      *config.Config
	consumer *queue.Consumer
	registry *handlers.Registry
	logger   *zap.Logger
	wg       sync.WaitGroup
}

// NewPool constructs a Pool. Call Start to launch the goroutines.
func NewPool(
	cfg *config.Config,
	consumer *queue.Consumer,
	registry *handlers.Registry,
	logger *zap.Logger,
) *Pool {
	return &Pool{
		cfg:      cfg,
		consumer: consumer,
		registry: registry,
		logger:   logger,
	}
}

// Start launches cfg.WorkerCount goroutines. Each goroutine runs independently
// and stops when ctx is cancelled (SIGTERM triggers this in main.go).
func (p *Pool) Start(ctx context.Context) {
	p.logger.Info("starting worker pool", zap.Int("workers", p.cfg.WorkerCount))
	for i := 0; i < p.cfg.WorkerCount; i++ {
		p.wg.Add(1)
		go p.runWorker(ctx, i)
	}
}

// Wait blocks until every goroutine has finished its current job and exited.
// Call this after cancelling the context to ensure a clean shutdown.
func (p *Pool) Wait() {
	p.wg.Wait()
	p.logger.Info("all workers stopped gracefully")
}

// runWorker is the per-goroutine event loop. It dequeues and processes jobs
// one at a time. The BLPOP timeout (5 s) gives it a chance to re-check ctx.
func (p *Pool) runWorker(ctx context.Context, id int) {
	defer p.wg.Done()
	p.logger.Info("worker started", zap.Int("id", id))

	for {
		// Check for shutdown before each dequeue attempt.
		select {
		case <-ctx.Done():
			p.logger.Info("worker shutting down", zap.Int("id", id))
			return
		default:
		}

		payload, err := p.consumer.Dequeue(ctx)
		if err != nil {
			p.logger.Error("dequeue error", zap.Int("id", id), zap.Error(err))
			continue
		}
		if payload == nil {
			continue // BLPOP timed out — loop to re-check context
		}

		p.process(ctx, payload, id)
	}
}

// process routes a single job to its handler and manages retry/DLQ on failure.
func (p *Pool) process(ctx context.Context, payload *job.LaravelPayload, workerID int) {
	log := p.logger.With(
		zap.String("uuid", payload.UUID),
		zap.String("job", payload.DisplayName),
		zap.Int("worker", workerID),
		zap.Int("attempt", payload.Attempts),
	)

	handler, err := p.registry.Resolve(payload.DisplayName)
	if err != nil {
		// No handler registered — this job can never succeed; send to DLQ immediately.
		log.Error("no handler registered for job type", zap.Error(err))
		if dlqErr := p.consumer.SendToDLQ(ctx, payload); dlqErr != nil {
			log.Error("failed to send unhandled job to DLQ", zap.Error(dlqErr))
		}
		return
	}

	log.Info("processing job")

	if err := handler.Handle(ctx, payload.Data); err != nil {
		log.Error("handler returned error", zap.Error(err))
		if retryErr := p.consumer.Retry(ctx, payload, p.cfg.MaxRetries); retryErr != nil {
			log.Error("retry/DLQ failed", zap.Error(retryErr))
		}
		return
	}

	log.Info("job completed successfully")
}
