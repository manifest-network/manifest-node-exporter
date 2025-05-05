package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/liftedinit/manifest-node-exporter/pkg/client"
	"github.com/liftedinit/manifest-node-exporter/pkg/collectors/grpc"
	serveConfig "github.com/liftedinit/manifest-node-exporter/pkg/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

		ctx, cancel := context.WithCancel(context.Background())
		handleInterrupt(cancel)

		grpcAddr := args[0]
		if err := validateGrpcAddress(grpcAddr); err != nil {
			return fmt.Errorf("invalid gRPC address '%s': %w", grpcAddr, err)
		}

		config := serveConfig.LoadServeConfig()

		grpcClient, err := client.NewGRPCClient(ctx, grpcAddr, config.Insecure)
		if err != nil {
			return fmt.Errorf("failed to initialize gRPC: %w", err)
		}

		_, err = registerGrpcCollectors(grpcClient)
		if err != nil {
			return fmt.Errorf("failed to register gRPC collectors: %w", err)
		}

		slog.Info("Starting Prometheus metrics server...", "address", config.ListenAddress)
		server, errChan := listen(config.ListenAddress)

		select {
		case err := <-errChan:
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			if shutdownErr := server.Shutdown(shutdownCtx); shutdownErr != nil {
				slog.Error("Failed to gracefully shutdown metrics server after error", "error", shutdownErr)
			}
			return err
		case <-ctx.Done():
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()

			if err := server.Shutdown(shutdownCtx); err != nil {
				slog.Error("Failed to gracefully shutdown metrics server", "error", err)
				return err
			}
			slog.Info("Metrics server stopped gracefully")
		}

		return nil
	},
}

func registerGrpcCollectors(grpcClient *client.GRPCClient) ([]prometheus.Collector, error) {
	collectors, err := grpc.DefaultGrpcRegistry.CreateGrpcCollectors(grpcClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC collectors: %w", err)
	}

	for _, collector := range collectors {
		if err := prometheus.Register(collector); err != nil {
			var alreadyRegistered prometheus.AlreadyRegisteredError
			if errors.As(err, &alreadyRegistered) {
				slog.Info("Collector already registered", "collector", alreadyRegistered.ExistingCollector)
			} else {
				return nil, fmt.Errorf("failed to register collector: %w", err)
			}
		}
	}

	return collectors, nil
}

func validateGrpcAddress(grpcAddr string) error {
	if grpcAddr == "" {
		return errors.New("gRPC address cannot be empty")
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

func init() {
	serveCmd.Flags().String("listen-address", ":2112", "Address to listen on")
	serveCmd.Flags().Bool("insecure", false, "Skip TLS certificate verification (INSECURE)")
	serveCmd.Flags().Uint("max-concurrency", 100, "Maximum request concurrency (advanced)")
	serveCmd.Flags().Uint("max-retries", 3, "Maximum number of retries for failed requests")

	if err := viper.BindPFlags(serveCmd.Flags()); err != nil {
		slog.Error("Failed to bind serveCmd flags", "error", err)
	}

	RootCmd.AddCommand(serveCmd)
}

// handleInterrupt handles interrupt signals for graceful shutdown.
func handleInterrupt(cancel context.CancelFunc) {
	// Handle interrupt signals for graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		slog.Info("Received interrupt signal, shutting down...")
		cancel()
	}()
}

func listen(addr string) (*http.Server, chan error) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{Addr: addr, Handler: mux}
	errChan := make(chan error)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- fmt.Errorf("prometheus server failed: %w", err)
		}
	}()

	return server, errChan
}
