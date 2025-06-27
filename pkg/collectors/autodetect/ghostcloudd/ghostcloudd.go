package ghostcloudd

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/liftedinit/manifest-node-exporter/pkg/client"
	"github.com/liftedinit/manifest-node-exporter/pkg/collectors/autodetect"
)

const processName = "ghostcloudd"
const defaultPort = 9090 // Default gRPC port for ghostcloudd

// Ensure ghostcloudd implements ProcessMonitor
var _ autodetect.ProcessMonitor = (*ghostclouddMonitor)(nil)

type ghostclouddMonitor struct{}

func init() {
	autodetect.RegisterMonitor(&ghostclouddMonitor{})
}

// Name returns the name of the process being monitored by ghostcloudd Monitor.
func (m *ghostclouddMonitor) Name() string {
	return processName
}

// Detect checks if the monitored process is running, validates its gRPC readiness, and retrieves process information.
func (m *ghostclouddMonitor) Detect() (*autodetect.ProcessInfo, error) {
	return autodetect.DetectProcessWithGrpc(processName, defaultPort)
}

// CollectCollectors gathers all registered Prometheus collectors for the ghostcloudd process using a provided gRPC client.
// It requires valid process information to establish a gRPC connection.
// Returns a slice of Prometheus collectors or an error if the process information is nil or the gRPC client cannot be created.
func (m *ghostclouddMonitor) CollectCollectors(ctx context.Context, processInfo *autodetect.ProcessInfo) ([]prometheus.Collector, error) {
	if processInfo == nil {
		return nil, fmt.Errorf("processInfo is nil")
	}

	// ProcessInfo should contain the necessary information to create a gRPC client
	target := net.JoinHostPort(processInfo.Address, strconv.Itoa(int(processInfo.Port)))
	grpcClient, err := client.NewGRPCClient(ctx, target)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client: %w", err)
	}

	var resultCollectors []prometheus.Collector
	for _, collector := range GetAllCollectorFactories() {
		resultCollectors = append(resultCollectors, collector(grpcClient))
	}

	return resultCollectors, nil
}
