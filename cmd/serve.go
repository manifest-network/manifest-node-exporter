package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/liftedinit/manifest-node-exporter/pkg"
	"github.com/liftedinit/manifest-node-exporter/pkg/monitors"
	_ "github.com/liftedinit/manifest-node-exporter/pkg/monitors/manifestd"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

		rootCtx, rootCancel := context.WithCancel(context.Background())
		defer rootCancel()
		handleInterrupt(rootCancel)

		registeredMonitors := monitors.GetRegisteredMonitors()
		if len(registeredMonitors) == 0 {
			return fmt.Errorf("no registered monitors found")
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

			if err := monitor.RegisterCollectors(rootCtx, processInfo); err != nil {
				slog.Error("Failed to register collectors", "name", monitor.Name(), "error", err)
				continue
			}
		}

		// Setup metrics server
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

// handleInterrupt handles interrupt signals for graceful shutdown.
func handleInterrupt(cancel context.CancelFunc) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		slog.Info("Received interrupt signal, shutting down...")
		cancel()
	}()
}

func init() {
	serveCmd.Flags().String("listen-address", "0.0.0.0:2112", "Address to listen on")
	serveCmd.Flags().Bool("insecure", false, "Skip TLS certificate verification (INSECURE)")

	if err := viper.BindPFlags(serveCmd.Flags()); err != nil {
		slog.Error("Failed to bind serveCmd flags", "error", err)
	}

	RootCmd.AddCommand(serveCmd)
}
