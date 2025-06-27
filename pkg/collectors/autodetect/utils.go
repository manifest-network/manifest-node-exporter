package autodetect

import (
	"fmt"
	"log/slog"
	"net"
	"slices"

	"github.com/liftedinit/manifest-node-exporter/pkg/utils"
	gopnet "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

type PortInfo struct {
	Address string
	Port    uint32
}

// IsProcessRunning checks if a process with the given name is running.
// It returns true and the PID if the process is found, otherwise false and 0.
// An error is returned if there's an issue listing or inspecting processes.
func IsProcessRunning(processName string) (bool, int32, error) {
	processes, err := process.Processes()
	if err != nil {
		return false, 0, fmt.Errorf("failed to list processes: %w", err)
	}

	for _, p := range processes {
		name, err := p.Name()
		// Ignore errors for individual processes (e.g., permission denied)
		if err != nil {
			continue
		}

		if name == processName {
			return true, p.Pid, nil
		}
	}

	// Process not found
	return false, 0, nil
}

// GetListeningPorts returns a list of TCP ports the process with the given PID is listening on.
func GetListeningPorts(pid int32) ([]PortInfo, error) {
	// Get all network connections (TCP only for listening ports)
	// Use "tcp" to get both tcp4 and tcp6
	connections, err := gopnet.ConnectionsPid("tcp", pid)
	if err != nil {
		return nil, fmt.Errorf("failed to get connections for pid %d: %w", pid, err)
	}

	var listeningPorts []PortInfo

	for _, conn := range connections {
		// Check if the connection status is LISTEN
		if conn.Status == "LISTEN" {
			// Ensure Laddr is not nil and has IP and Port
			if conn.Laddr.IP != "" && conn.Laddr.Port != 0 {
				listeningPorts = append(listeningPorts, PortInfo{
					Address: conn.Laddr.IP,
					Port:    conn.Laddr.Port,
				})
			}
		}
	}

	if len(listeningPorts) == 0 {
		slog.Warn("No listen ports found")
	}

	return listeningPorts, nil
}

func DetectProcessWithGrpc(processName string, defaultPort uint32) (*ProcessInfo, error) {
	ok, pid, err := IsProcessRunning(processName)
	if err != nil {
		return nil, fmt.Errorf("failed to check if %s is running: %w", processName, err)
	}
	if !ok {
		slog.Info("Process not found", "name", processName)
		return nil, nil
	}

	ports, err := GetListeningPorts(pid)
	if err != nil {
		return nil, fmt.Errorf("failed to get listening ports for process %d: %w", pid, err)
	}

	if len(ports) == 0 {
		slog.Warn("Process found but no listening ports detected", "name", processName, "pid", pid)
		return nil, fmt.Errorf("%s process found (PID %d) but has no listening ports", processName, pid)
	}

	defaultPortIndex := slices.IndexFunc(ports, func(port PortInfo) bool {
		return port.Port == defaultPort
	})
	if defaultPortIndex != -1 {
		slog.Debug("Process listening on default port", "name", processName, "pid", pid, "port", defaultPort)
		defaultPortInfo := ports[defaultPortIndex]
		target := net.JoinHostPort(defaultPortInfo.Address, fmt.Sprint(defaultPortInfo.Port))
		if utils.IsGrpcPort(target) {
			slog.Debug("gRPC connection successful", "target", target)
			return &ProcessInfo{
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
			return &ProcessInfo{
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
