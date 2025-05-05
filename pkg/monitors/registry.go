package monitors

import (
	"context"
	"log/slog"
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
	// RegisterCollectors registers the collectors for the process.
	RegisterCollectors(context.Context, *ProcessInfo) error
}

var registry = make(map[string]ProcessMonitor)

// Register adds a ProcessMonitor to the global registry.
// It should typically be called from the init() function of a monitor package.
func Register(monitor ProcessMonitor) {
	name := monitor.Name()
	if _, exists := registry[name]; exists {
		slog.Warn("ProcessMonitor already registered, overwriting", "name", name)
	}
	slog.Debug("Registering ProcessMonitor", "name", name)
	registry[name] = monitor
}

// GetRegisteredMonitors returns a slice of all registered monitors.
func GetRegisteredMonitors() []ProcessMonitor {
	monitors := make([]ProcessMonitor, 0, len(registry))
	for _, monitor := range registry {
		monitors = append(monitors, monitor)
	}
	return monitors
}
