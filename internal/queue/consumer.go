// Package queue implements a Redis consumer compatible with Laravel's queue format.
// Laravel pushes jobs with RPUSH to "queues:{name}". We pop with BLPOP (FIFO).
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/hamdyelbatal122/go-task-offloader/internal/job"
)

// Consumer reads jobs from a Redis list and manages retry/DLQ logic.
type Consumer struct {
	client   *redis.Client
	queueKey string // e.g. "queues:default"
	dlqKey   string // e.g. "queues:failed"
	logger   *zap.Logger
}

// NewConsumer creates a Consumer for the given queue and DLQ names.
func NewConsumer(client *redis.Client, queueName, dlqName string, logger *zap.Logger) *Consumer {
	return &Consumer{
		client:   client,
		queueKey: fmt.Sprintf("queues:%s", queueName),
		dlqKey:   fmt.Sprintf("queues:%s", dlqName),
		logger:   logger,
	}
}

// Dequeue blocks for up to 5 seconds waiting for a job.
// Returns (nil, nil) on timeout so the caller can check context cancellation.
func (c *Consumer) Dequeue(ctx context.Context) (*job.LaravelPayload, error) {
	// BLPOP pops from the left (head). Laravel pushes with RPUSH to the tail,
	// so this gives FIFO ordering — oldest job is processed first.
	results, err := c.client.BLPop(ctx, 5*time.Second, c.queueKey).Result()
	if err != nil {
		if err == redis.Nil || ctx.Err() != nil {
			return nil, nil // timeout or shutdown — not an error
		}
		return nil, fmt.Errorf("blpop on %q failed: %w", c.queueKey, err)
	}

	// BLPop returns [key, value]; index 1 is the raw JSON payload.
	if len(results) < 2 {
		return nil, fmt.Errorf("unexpected blpop response length: %d", len(results))
	}

	raw := results[1]
	var payload job.LaravelPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		c.logger.Error("malformed job payload — discarding",
			zap.String("raw", raw),
			zap.Error(err),
		)
		return nil, fmt.Errorf("json unmarshal: %w", err)
	}

	c.logger.Debug("job dequeued",
		zap.String("uuid", payload.UUID),
		zap.String("job", payload.DisplayName),
		zap.Int("attempt", payload.Attempts),
	)

	return &payload, nil
}

// Retry either re-queues the job (if under maxRetries) or sends it to the DLQ.
// The attempts counter on the payload is incremented before re-queuing.
func (c *Consumer) Retry(ctx context.Context, payload *job.LaravelPayload, maxRetries int) error {
	payload.Attempts++

	if payload.Attempts >= maxRetries {
		return c.SendToDLQ(ctx, payload)
	}

	updated, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload for retry: %w", err)
	}

	// Push to the tail so it doesn't cut in front of other waiting jobs.
	if err := c.client.RPush(ctx, c.queueKey, string(updated)).Err(); err != nil {
		return fmt.Errorf("rpush retry to %q: %w", c.queueKey, err)
	}

	c.logger.Warn("job requeued for retry",
		zap.String("uuid", payload.UUID),
		zap.Int("attempt", payload.Attempts),
		zap.Int("max", maxRetries),
	)
	return nil
}

// SendToDLQ pushes a permanently failed job to the Dead Letter Queue.
// The DLQ can be inspected and replayed manually or via Laravel's queue:retry.
func (c *Consumer) SendToDLQ(ctx context.Context, payload *job.LaravelPayload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal DLQ payload: %w", err)
	}

	if err := c.client.RPush(ctx, c.dlqKey, string(data)).Err(); err != nil {
		return fmt.Errorf("rpush to DLQ %q: %w", c.dlqKey, err)
	}

	c.logger.Error("job permanently failed — sent to DLQ",
		zap.String("uuid", payload.UUID),
		zap.String("job", payload.DisplayName),
		zap.Int("attempts", payload.Attempts),
	)
	return nil
}
