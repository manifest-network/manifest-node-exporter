package manifestd

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"slices"
	"strconv"

	"github.com/liftedinit/manifest-node-exporter/pkg/autodetect"
	"github.com/liftedinit/manifest-node-exporter/pkg/autodetect/manifestd/collectors"
	"github.com/liftedinit/manifest-node-exporter/pkg/client"
	"github.com/liftedinit/manifest-node-exporter/pkg/utils"
)

const processName = "manifestd"
const defaultPort = 9090 // Default gRPC port for manifestd

// Ensure manifestdMonitor implements ProcessMonitor
var _ autodetect.ProcessMonitor = (*manifestdMonitor)(nil)

type manifestdMonitor struct{}

func init() {
	autodetect.Register(&manifestdMonitor{})
}

func (m *manifestdMonitor) Name() string {
	return processName
}

func (m *manifestdMonitor) Detect() (*autodetect.ProcessInfo, error) {
	ok, pid, err := autodetect.IsProcessRunning(processName)
	if err != nil {
		return nil, fmt.Errorf("failed to check if %s is running: %w", processName, err)
	}
	if !ok {
		slog.Info("Process not found", "name", processName)
		return nil, nil // Return nil, nil to indicate not found without error
	}

	ports, err := autodetect.GetListeningPorts(pid)
	if err != nil {
		return nil, fmt.Errorf("failed to get listening ports for process %d: %w", pid, err)
	}

	if len(ports) == 0 {
		slog.Warn("Process found but no listening ports detected", "name", processName, "pid", pid)
		return nil, fmt.Errorf("%s process found (PID %d) but has no listening ports", processName, pid)
	}

	defaultPortIndex := slices.IndexFunc(ports, func(port autodetect.PortInfo) bool {
		return port.Port == defaultPort
	})
	// Check the default gRPC port first
	if defaultPortIndex != -1 {
		slog.Debug("Process listening on default port", "name", processName, "pid", pid, "port", defaultPort)
		defaultPortInfo := ports[defaultPortIndex]
		target := net.JoinHostPort(defaultPortInfo.Address, fmt.Sprint(defaultPortInfo.Port))

		if utils.IsGrpcPort(target) {
			slog.Debug("gRPC connection successful", "target", target)
			return &autodetect.ProcessInfo{
				Pid:     pid,
				Address: defaultPortInfo.Address,
				Port:    defaultPortInfo.Port,
			}, nil
		} else {
			slog.Warn("gRPC connection failed on default port", "target", target)
		}
	}

	slog.Debug("Default port not found, checking other ports", "name", processName, "pid", pid)
	for _, port := range ports {
		target := net.JoinHostPort(port.Address, fmt.Sprint(port.Port))
		if utils.IsGrpcPort(target) {
			slog.Debug("gRPC connection successful", "target", target)
			return &autodetect.ProcessInfo{
				Pid:     pid,
				Address: port.Address,
				Port:    port.Port,
			}, nil
		} else {
			slog.Warn("gRPC connection failed", "target", target)
		}
	}

	return nil, fmt.Errorf("no gRPC connection found for %s process (PID %d)", processName, pid)
}

func (m *manifestdMonitor) RegisterCollectors(ctx context.Context, processInfo *autodetect.ProcessInfo) error {
	if processInfo == nil {
		return fmt.Errorf("processInfo is nil")
	}

	// ProcessInfo should contain the necessary information to create a gRPC client
	target := net.JoinHostPort(processInfo.Address, strconv.Itoa(int(processInfo.Port)))
	grpcClient, err := client.NewGRPCClient(ctx, target, true)
	if err != nil {
		return fmt.Errorf("failed to create gRPC client: %w", err)
	}

	// Register the gRPC client with the registry
	_, err = collectors.RegisterCollectors(grpcClient)
	if err != nil {
		return fmt.Errorf("failed to create gRPC collectors: %w", err)
	}

	return nil
}
