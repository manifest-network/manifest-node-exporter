package ghostcloudd

import (
	"maps"
	"slices"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/manifest-network/manifest-node-exporter/pkg/client"
	"github.com/manifest-network/manifest-node-exporter/pkg/utils"
)

// GhostclouddCollectorFactory is a function type that creates a prometheus.Collector.
// It takes a gRPC client and optional extra parameters and returns a prometheus.Collector.
// This is used to register different types of collectors for the ghostcloudd process.
// The gRPC client is found at runtime by the autodetection process.
// The extra parameters can be used to pass additional configuration or context to the collector factory.
type GhostclouddCollectorFactory = func(grpcClient *client.GRPCClient, extra ...interface{}) prometheus.Collector

// ghostclouddCollectorRegistry is a registry for all ghostcloudd collector factories.
var ghostclouddCollectorRegistry = utils.NewRegistry[GhostclouddCollectorFactory]()

// RegisterCollectorFactory registers a new collector factory with the ghostcloudd collector registry.
func RegisterCollectorFactory(name string, factory GhostclouddCollectorFactory) {
	ghostclouddCollectorRegistry.Register(name, factory)
}

// GetAllCollectorFactories retrieves all the collector factories from the registry.
func GetAllCollectorFactories() []GhostclouddCollectorFactory {
	return slices.Collect(maps.Values(ghostclouddCollectorRegistry.GetAll()))
}
