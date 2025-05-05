package grpc

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/liftedinit/manifest-node-exporter/pkg"
)

type GrpcCollectorFactory func(client *pkg.GRPCClient, extraParams ...interface{}) (prometheus.Collector, error)

// GrpcRegistry holds factories for gRPC-based collectors.
type GrpcRegistry struct {
	factories []GrpcCollectorFactory
}

// NewGrpcRegistry creates a new registry for gRPC collectors.
func NewGrpcRegistry() *GrpcRegistry {
	return &GrpcRegistry{
		factories: make([]GrpcCollectorFactory, 0),
	}
}

// Register adds a new gRPC collector factory.
func (r *GrpcRegistry) Register(factory GrpcCollectorFactory) {
	r.factories = append(r.factories, factory)
}

// CreateGrpcCollectors instantiates all registered gRPC collectors.
func (r *GrpcRegistry) CreateGrpcCollectors(client *pkg.GRPCClient, extraParams ...interface{}) ([]prometheus.Collector, error) {
	if client == nil {
		return nil, errors.New("gRPC client is nil")
	}
	if client.Conn == nil {
		return nil, errors.New("gRPC client connection is nil for gRPC collectors")
	}

	collectors := make([]prometheus.Collector, 0, len(r.factories))
	for _, factory := range r.factories {
		collector, err := factory(client, extraParams...)
		if err != nil {
			// Consider logging the specific factory that failed
			return nil, err
		}
		collectors = append(collectors, collector)
	}
	return collectors, nil
}

// DefaultGrpcRegistry is the default registry instance for gRPC collectors.
var DefaultGrpcRegistry = NewGrpcRegistry()

// RegisterGrpcCollectorFactory registers a factory with the default gRPC registry.
func RegisterGrpcCollectorFactory(factory GrpcCollectorFactory) {
	DefaultGrpcRegistry.Register(factory)
}

func RegisterCollectors(grpcClient *pkg.GRPCClient) ([]prometheus.Collector, error) {
	if grpcClient == nil || grpcClient.Conn == nil {
		return nil, fmt.Errorf("cannot register collectors with a nil or unconnected gRPC client")
	}

	// Use the DefaultGrpcRegistry defined in common.go to create collectors
	collectors, err := DefaultGrpcRegistry.CreateGrpcCollectors(grpcClient)
	if err != nil {
		// Error already logged by CreateGrpcCollectors if a factory failed
		return nil, fmt.Errorf("failed to create gRPC collectors: %w", err)
	}

	var registeredCollectors []prometheus.Collector
	registeredCount := 0
	skippedCount := 0

	for _, collector := range collectors {
		collectorType := fmt.Sprintf("%T", collector) // Get type for logging
		if err := prometheus.DefaultRegisterer.Register(collector); err != nil {
			var alreadyRegistered prometheus.AlreadyRegisteredError
			if errors.As(err, &alreadyRegistered) {
				// This is often benign during development or restarts, log as Info or Debug
				slog.Debug("Collector already registered with Prometheus, skipping registration.", "collector_type", collectorType)
				// We might still want to include it in the returned list if it was successfully *created*
				// Depending on desired behavior, you could fetch the existing collector:
				// registeredCollectors = append(registeredCollectors, alreadyRegistered.ExistingCollector)
				skippedCount++
			} else {
				// This is a more serious registration error
				slog.Error("Failed to register collector with Prometheus", "collector_type", collectorType, "error", err)
				// Decide if you want to fail entirely or just skip this collector
				// Failing fast is often safer.
				return registeredCollectors, fmt.Errorf("failed to register collector type %s: %w", collectorType, err)
			}
		} else {
			slog.Info("Successfully registered collector with Prometheus.", "collector_type", collectorType)
			registeredCollectors = append(registeredCollectors, collector)
			registeredCount++
		}
	}

	slog.Info("gRPC collector registration complete.", "newly_registered", registeredCount, "skipped_existing", skippedCount, "total_created", len(collectors))
	return registeredCollectors, nil
}
