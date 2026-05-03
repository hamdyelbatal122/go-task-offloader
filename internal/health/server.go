// Package health exposes a lightweight HTTP server with two endpoints:
//   - GET /health  — liveness probe (returns 200 OK immediately)
//   - GET /metrics — runtime stats: goroutine count, memory, uptime, CPU count
//
// These are consumed by Docker HEALTHCHECK, Kubernetes probes, or any
// monitoring agent (Prometheus node exporter, Datadog, etc.).
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"runtime"
	"time"

	"go.uber.org/zap"
)

// metricsResponse is the JSON schema returned by GET /metrics.
type metricsResponse struct {
	Status       string  `json:"status"`
	GoVersion    string  `json:"go_version"`
	NumCPU       int     `json:"num_cpu"`
	NumGoroutine int     `json:"num_goroutines"`
	MemAllocMB   float64 `json:"mem_alloc_mb"`
	MemSysMB     float64 `json:"mem_sys_mb"`
	GCRuns       uint32  `json:"gc_runs"`
	Uptime       string  `json:"uptime"`
}

// Server wraps an http.Server for the health/metrics endpoints.
type Server struct {
	port      string
	startTime time.Time
	logger    *zap.Logger
	srv       *http.Server
}

// NewServer creates a Server. Call Start to begin listening.
func NewServer(port string, logger *zap.Logger) *Server {
	return &Server{
		port:      port,
		startTime: time.Now(),
		logger:    logger,
	}
}

// Start begins serving in a background goroutine.
func (s *Server) Start() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/metrics", s.handleMetrics)

	s.srv = &http.Server{
		Addr:         ":" + s.port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	go func() {
		s.logger.Info("health server listening", zap.String("addr", s.srv.Addr))
		if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("health server error", zap.Error(err))
		}
	}()
}

// Stop gracefully shuts down the HTTP server within the given context deadline.
func (s *Server) Stop(ctx context.Context) {
	if err := s.srv.Shutdown(ctx); err != nil {
		s.logger.Error("health server shutdown error", zap.Error(err))
	}
}

// handleHealth is a simple liveness probe — always returns 200 if the process
// is running. Kubernetes/Docker checks this to decide whether to restart.
func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"}) //nolint:errcheck
}

// handleMetrics returns Go runtime statistics useful for capacity planning
// and detecting goroutine leaks or memory pressure.
func (s *Server) handleMetrics(w http.ResponseWriter, _ *http.Request) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	resp := metricsResponse{
		Status:       "ok",
		GoVersion:    runtime.Version(),
		NumCPU:       runtime.NumCPU(),
		NumGoroutine: runtime.NumGoroutine(),
		MemAllocMB:   float64(mem.Alloc) / (1024 * 1024),
		MemSysMB:     float64(mem.Sys) / (1024 * 1024),
		GCRuns:       mem.NumGC,
		Uptime:       time.Since(s.startTime).Round(time.Second).String(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp) //nolint:errcheck
}
