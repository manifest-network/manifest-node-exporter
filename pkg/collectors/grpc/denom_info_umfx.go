package grpc

import (
	"github.com/liftedinit/manifest-node-exporter/pkg/client"
	"github.com/prometheus/client_golang/prometheus"
)

// TODO: Specify the denoms to monitor via the CLI/config.
// init is called to register the `umfx` Bank collector factory with the default gRPC registry.
func init() {
	RegisterGrpcCollectorFactory(func(client *client.GRPCClient, extraParams ...interface{}) (prometheus.Collector, error) {
		return NewDenomInfoCollector(client, "umfx"), nil
	})
}
