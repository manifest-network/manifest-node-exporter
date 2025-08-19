package collectors

import (
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/manifest-network/manifest-node-exporter/pkg/client"
)

func ReportUpMetric(ch chan<- prometheus.Metric, desc *prometheus.Desc, value float64) {
	metric, err := prometheus.NewConstMetric(desc, prometheus.GaugeValue, value)
	if err != nil {
		slog.Error("Failed to create up metric", "error", err)
	} else {
		ch <- metric
	}
}

func ReportInvalidMetric(ch chan<- prometheus.Metric, desc *prometheus.Desc, err error) {
	ch <- prometheus.NewInvalidMetric(desc, err)
}

func ValidateGrpcClient(client *client.GRPCClient) error {
	if client == nil {
		return status.Error(codes.Internal, "gRPC client is nil during collect")
	}
	if client.Conn == nil {
		return status.Error(codes.Internal, "gRPC client connection is nil during collect")
	}
	return nil
}

func ValidateClient(client *client.GRPCClient, initErr error) error {
	if initErr != nil {
		return initErr
	}
	if clientErr := ValidateGrpcClient(client); clientErr != nil {
		return clientErr
	}
	return nil
}
