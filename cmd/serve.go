package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/liftedinit/manifest-node-exporter/pkg"
	"github.com/liftedinit/manifest-node-exporter/pkg/collectors/grpc"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve grpc-addr [flags]",
	Short: "Serve Manifest node metrics",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if parent := cmd.Parent(); parent != nil && parent.PreRunE != nil {
			if err := parent.PreRunE(parent, args); err != nil {
				return err
			}
		}

		config := pkg.LoadServeConfig()

		rootCtx, rootCancel := context.WithCancel(context.Background())
		defer rootCancel()
		handleInterrupt(rootCancel)

		// Setup gRPC
		grpcAddr := args[0]
		if err := setupGrpc(rootCtx, grpcAddr, config.Insecure); err != nil {
			return fmt.Errorf("failed to setup gRPC: %w", err)
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

func setupGrpc(ctx context.Context, grpcAddr string, insecure bool) error {
	if err := validateGrpcAddress(grpcAddr); err != nil {
		return fmt.Errorf("invalid gRPC address '%s': %w", grpcAddr, err)
	}

	grpcClient, err := pkg.NewGRPCClient(ctx, grpcAddr, insecure)
	if err != nil {
		return fmt.Errorf("failed to initialize gRPC: %w", err)
	}
	defer func() {
		if grpcClient != nil && grpcClient.Conn != nil {
			slog.Debug("Closing gRPC connection", "target", grpcClient.Conn.Target())
			if err := grpcClient.Conn.Close(); err != nil {
				slog.Error("Failed to close gRPC connection", "error", err)
			}
		}
	}()

	_, err = grpc.RegisterCollectors(grpcClient)
	if err != nil {
		return fmt.Errorf("failed to register gRPC collectors: %w", err)
	}

	return nil
}

func validateGrpcAddress(grpcAddr string) error {
	if grpcAddr == "" {
		return fmt.Errorf("gRPC address cannot be empty")
	}

	_, portStr, err := net.SplitHostPort(grpcAddr)
	if err != nil {
		return fmt.Errorf("invalid gRPC address format '%s', expected host:port: %w", grpcAddr, err)
	}

	port, err := net.LookupPort("tcp", portStr)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("invalid port number: '%s', expected a valid port number: %w", portStr, err)
	}

	return nil
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
	serveCmd.Flags().Uint("max-concurrency", 100, "Maximum request concurrency (advanced)")
	serveCmd.Flags().Uint("max-retries", 3, "Maximum number of retries for failed requests")

	if err := viper.BindPFlags(serveCmd.Flags()); err != nil {
		slog.Error("Failed to bind serveCmd flags", "error", err)
	}

	RootCmd.AddCommand(serveCmd)
}
