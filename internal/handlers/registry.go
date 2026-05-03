// Package handlers provides the Handler interface and the Registry that maps
// Laravel job class names to their Go handler implementations.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
)

// Handler is the interface every job handler must satisfy.
// Handle receives the raw "data" field from the Laravel payload and returns
// an error if processing fails (triggers retry logic in the worker pool).
type Handler interface {
	Handle(ctx context.Context, data json.RawMessage) error
}

// Registry maps a Laravel job's DisplayName (e.g. "App\\Jobs\\ProcessImageJob")
// to the Go Handler that should process it.
type Registry struct {
	handlers map[string]Handler
}

// NewRegistry creates an empty Registry. Register handlers before starting
// the worker pool.
func NewRegistry() *Registry {
	return &Registry{handlers: make(map[string]Handler)}
}

// Register associates a handler with a job name.
// name must exactly match the "displayName" field in the Laravel payload.
func (r *Registry) Register(name string, h Handler) {
	r.handlers[name] = h
}

// Resolve returns the handler for the given job name.
// Returns an error if no handler has been registered for that name.
func (r *Registry) Resolve(name string) (Handler, error) {
	h, ok := r.handlers[name]
	if !ok {
		return nil, fmt.Errorf("no handler registered for %q", name)
	}
	return h, nil
}
