package collectors

import (
	"maps"
	"slices"

	"github.com/liftedinit/manifest-node-exporter/pkg/client"
	"github.com/liftedinit/manifest-node-exporter/pkg/utils"
	"github.com/prometheus/client_golang/prometheus"
)

type ManifestdCollectorFactory = func(grpcClient *client.GRPCClient, extra ...interface{}) prometheus.Collector

var manifestdCollectorRegistry = utils.NewRegistry[ManifestdCollectorFactory]()

func RegisterCollectorFactory(name string, factory ManifestdCollectorFactory) {
	manifestdCollectorRegistry.Register(name, factory)
}

func GetAllCollectorFactories() []ManifestdCollectorFactory {
	return slices.Collect(maps.Values(manifestdCollectorRegistry.GetAll()))
}
