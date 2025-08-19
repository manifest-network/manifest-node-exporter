package autodetect

import (
	"context"
	"maps"
	"slices"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/manifest-network/manifest-node-exporter/pkg/utils"
)

// ProcessInfo holds information about a detected process.
type ProcessInfo struct {
	Pid     int32
	Address string // Primary listening address (e.g., gRPC)
	Port    uint32 // Primary listening port
}

// ProcessMonitor defines the interface for monitoring a specific process.
type ProcessMonitor interface {
	// Name returns the name of the process to monitor (e.g., "manifestd").
	Name() string
	// Detect checks if the process is running and returns its info.
	Detect() (*ProcessInfo, error)
	// CollectCollectors registers the collectors for the process.
	CollectCollectors(context.Context, *ProcessInfo) ([]prometheus.Collector, error)
}

// processMonitorRegistry is a registry for all process monitors.
var processMonitorRegistry = utils.NewRegistry[ProcessMonitor]()

// RegisterMonitor registers a new process monitor with the registry.
func RegisterMonitor(monitor ProcessMonitor) {
	processMonitorRegistry.Register(monitor.Name(), monitor)
}

// GetAllMonitors retrieves all the registered process monitors.
func GetAllMonitors() []ProcessMonitor {
	return slices.Collect(maps.Values(processMonitorRegistry.GetAll()))
}
