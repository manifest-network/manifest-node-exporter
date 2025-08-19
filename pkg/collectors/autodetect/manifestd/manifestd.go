package manifestd

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/manifest-network/manifest-node-exporter/pkg/client"
	"github.com/manifest-network/manifest-node-exporter/pkg/collectors/autodetect"
)

const processName = "manifestd"
const defaultPort = 9090 // Default gRPC port for manifestd

// Ensure manifestdMonitor implements ProcessMonitor
var _ autodetect.ProcessMonitor = (*manifestdMonitor)(nil)

type manifestdMonitor struct{}

func init() {
	autodetect.RegisterMonitor(&manifestdMonitor{})
}

// Name returns the name of the process being monitored by manifestd Monitor.
func (m *manifestdMonitor) Name() string {
	return processName
}

// Detect checks if the monitored process is running, validates its gRPC readiness, and retrieves process information.
func (m *manifestdMonitor) Detect() (*autodetect.ProcessInfo, error) {
	return autodetect.DetectProcessWithGrpc(processName, defaultPort)
}

// CollectCollectors gathers all registered Prometheus collectors for the manifestd process using a provided gRPC client.
// It requires valid process information to establish a gRPC connection.
// Returns a slice of Prometheus collectors or an error if the process information is nil or the gRPC client cannot be created.
func (m *manifestdMonitor) CollectCollectors(ctx context.Context, processInfo *autodetect.ProcessInfo) ([]prometheus.Collector, error) {
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
