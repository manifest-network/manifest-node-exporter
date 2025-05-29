package manifestd

import (
	"maps"
	"slices"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/liftedinit/manifest-node-exporter/pkg/client"
	"github.com/liftedinit/manifest-node-exporter/pkg/utils"
)

// ManifestdCollectorFactory is a function type that creates a prometheus.Collector.
// It takes a gRPC client and optional extra parameters and returns a prometheus.Collector.
// This is used to register different types of collectors for the manifestd process.
// The gRPC client is found at runtime by the autodetection process.
// The extra parameters can be used to pass additional configuration or context to the collector factory.
type ManifestdCollectorFactory = func(grpcClient *client.GRPCClient, extra ...interface{}) prometheus.Collector

// manifestdCollectorRegistry is a registry for all manifestd collector factories.
var manifestdCollectorRegistry = utils.NewRegistry[ManifestdCollectorFactory]()

// RegisterCollectorFactory registers a new collector factory with the manifestd collector registry.
func RegisterCollectorFactory(name string, factory ManifestdCollectorFactory) {
	manifestdCollectorRegistry.Register(name, factory)
}

// GetAllCollectorFactories retrieves all the collector factories from the registry.
func GetAllCollectorFactories() []ManifestdCollectorFactory {
	return slices.Collect(maps.Values(manifestdCollectorRegistry.GetAll()))
}
