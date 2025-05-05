package pkg

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsServer wraps the HTTP server for Prometheus metrics.
type MetricsServer struct {
	httpServer *http.Server
	listenAddr string
}

// NewMetricsServer creates a new MetricsServer instance.
// It configures an HTTP server to listen on the given address
// and expose the default Prometheus registry via promhttp.Handler() on "/metrics".
func NewMetricsServer(listenAddr string) *MetricsServer {
	mux := http.NewServeMux()
	// Note: Prometheus collectors should be registered *before* the server is started.
	// This handler uses the default Prometheus registry.
	mux.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:         listenAddr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return &MetricsServer{
		httpServer: srv,
		listenAddr: listenAddr,
	}
}

// Start runs the server in a background goroutine.
// It returns a channel that will receive an error if the server
// fails to start or stops unexpectedly (excluding http.ErrServerClosed).
func (s *MetricsServer) Start() <-chan error {
	errChan := make(chan error, 1) // Buffered to prevent blocking sender on unexpected error
	slog.Info("Starting Prometheus metrics server...", "address", s.listenAddr)

	go func() {
		// ListenAndServe blocks until the server stops.
		err := s.httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			// Only send non-nil errors that aren't the expected ErrServerClosed
			errChan <- fmt.Errorf("prometheus metrics server failed: %w", err)
		}
		// Close the channel only if an error occurred, otherwise leave it open.
		// The caller uses context cancellation for shutdown signal, not channel closing.
		// close(errChan) // Don't close here on normal shutdown
	}()

	// Give a brief moment to allow ListenAndServe to start or fail early
	time.Sleep(50 * time.Millisecond)

	return errChan
}

// Shutdown gracefully shuts down the HTTP server.
// It waits for the duration specified by the context's deadline.
func (s *MetricsServer) Shutdown(ctx context.Context) error {
	slog.Info("Attempting graceful shutdown of metrics server...")
	err := s.httpServer.Shutdown(ctx)
	if err == nil {
		slog.Info("Metrics server stopped gracefully.")
	} else if errors.Is(err, context.DeadlineExceeded) {
		slog.Warn("Metrics server shutdown timed out.", "error", err)
	} else {
		slog.Error("Error during metrics server shutdown.", "error", err)
	}
	return err // Return the error for the caller
}
