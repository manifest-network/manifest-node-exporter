package manifest_node_exporter

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/manifest-network/manifest-node-exporter/pkg"
	"github.com/manifest-network/manifest-node-exporter/pkg/collectors"
	"github.com/manifest-network/manifest-node-exporter/pkg/collectors/autodetect"
	_ "github.com/manifest-network/manifest-node-exporter/pkg/collectors/autodetect/ghostcloudd" // RegisterMonitor the ghostcloudd monitor (side-effect)
	_ "github.com/manifest-network/manifest-node-exporter/pkg/collectors/autodetect/manifestd"   // RegisterMonitor the manifestd monitor (side-effect)
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve [flags]",
	Short: "Serve Manifest node metrics",
	RunE: func(cmd *cobra.Command, args []string) error {
		if parent := cmd.Parent(); parent != nil && parent.PreRunE != nil {
			if err := parent.PreRunE(parent, args); err != nil {
				return err
			}
		}
		slog.Info("Starting manifest-node-exporter")

		config := pkg.LoadServeConfig()

		rootCtx, rootCancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer rootCancel()

		var allCollectors []prometheus.Collector
		if config.IpBaseKey == "" {
			slog.Warn("No ipbase API key specified. Skipping GeoIP collection.")
		} else {
			geoIpCollector := collectors.NewGeoIPCollector(config.IpBaseKey, config.StateFile)
			allCollectors = append(allCollectors, geoIpCollector)
		}

		// Setup process monitors and fetch all registered collectors
		monitorCollectors, err := setupMonitors(rootCtx)
		if err != nil {
			return fmt.Errorf("failed to setup monitors: %w", err)
		}
		allCollectors = append(allCollectors, monitorCollectors...)

		// Register all collectors with Prometheus
		registerCollectors(allCollectors)

		// Setup and start metrics server
		metricsSrv := pkg.NewMetricsServer(config.ListenAddress)
		serverErrChan := metricsSrv.Start()

		// Wait for server errors or shutdown signal
		select {
		case err := <-serverErrChan:
			slog.Error("Metrics server encountered an error", "error", err)
			rootCancel()
			return fmt.Errorf("metrics server failed: %w", err)

		case <-rootCtx.Done():
			slog.Info("Shutdown signal received, initiating graceful shutdown...")
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer shutdownCancel()

			if err := metricsSrv.Shutdown(shutdownCtx); err != nil {
				slog.Error("Error during graceful shutdown of metrics server", "error", err)
			} else {
				slog.Info("Metrics server has shut down.")
			}
		}

		slog.Info("Application shut down complete.")
		return nil
	},
}

// setupMonitors initializes and sets up all registered process monitors.
// It detects the processes and collects the corresponding Prometheus collectors.
func setupMonitors(ctx context.Context) ([]prometheus.Collector, error) {
	var allCollectors []prometheus.Collector
	registeredMonitors := autodetect.GetAllMonitors()
	if len(registeredMonitors) == 0 {
		return nil, fmt.Errorf("no registered monitors found")
	} else {
		slog.Info("Registered monitors", "count", len(registeredMonitors))
		for _, monitor := range registeredMonitors {
			slog.Debug("Monitor", "name", monitor.Name())
		}
	}

	for _, monitor := range registeredMonitors {
		slog.Info("Attempting to detect process", "name", monitor.Name())
		processInfo, err := monitor.Detect()
		if err != nil {
			slog.Error("Failed to detect process", "name", monitor.Name(), "error", err)
			continue
		}

		if processInfo == nil {
			continue
		}

		collectors, err := monitor.CollectCollectors(ctx, processInfo)
		if err != nil {
			slog.Error("Failed to collect collectors", "name", monitor.Name(), "error", err)
			continue
		}
		allCollectors = append(allCollectors, collectors...)
	}

	if len(allCollectors) == 0 {
		slog.Warn("No collectors found for any registered monitors")
	}

	return allCollectors, nil

}

// registerCollectors registers the provided Prometheus collectors with the default Prometheus registry.
func registerCollectors(collectors []prometheus.Collector) {
	for _, collector := range collectors {
		collectorType := fmt.Sprintf("%T", collector) // Get type for logging
		if err := prometheus.DefaultRegisterer.Register(collector); err != nil {
			var alreadyRegistered prometheus.AlreadyRegisteredError
			if errors.As(err, &alreadyRegistered) {
				slog.Debug("Collector already registered with Prometheus, skipping registration.", "collector_type", collectorType)
			} else {
				slog.Error("Failed to register collector with Prometheus", "collector_type", collectorType, "error", err)
			}
		} else {
			slog.Info("Successfully registered collector with Prometheus.", "collector_type", collectorType)
		}
	}
}

func init() {
	serveCmd.Flags().String("listen-address", "0.0.0.0:2112", "Address to listen on")
	serveCmd.Flags().String("ipbase-key", "", "IPBase API key to use for GeoIP lookup")
	serveCmd.Flags().String("state-file", "./state.json", "Path to the state file for GeoIP data persistence")

	if err := viper.BindPFlags(serveCmd.Flags()); err != nil {
		slog.Error("Failed to bind serveCmd flags", "error", err)
	}

	RootCmd.AddCommand(serveCmd)
}
