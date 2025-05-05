package grpc

import (
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/liftedinit/manifest-node-exporter/pkg"
)

func reportUpMetric(ch chan<- prometheus.Metric, desc *prometheus.Desc, value float64) {
	metric, err := prometheus.NewConstMetric(desc, prometheus.GaugeValue, value)
	if err != nil {
		slog.Error("Failed to create up metric", "error", err)
	} else {
		ch <- metric
	}
}

func reportInvalidMetric(ch chan<- prometheus.Metric, desc *prometheus.Desc, err error) {
	ch <- prometheus.NewInvalidMetric(desc, err)
}

func validateGrpcClient(client *pkg.GRPCClient) error {
	if client == nil {
		return status.Error(codes.Internal, "gRPC client is nil during collect")
	}
	if client.Conn == nil {
		return status.Error(codes.Internal, "gRPC client connection is nil during collect")
	}
	return nil
}

func validateClient(client *pkg.GRPCClient, initErr error) error {
	if initErr != nil {
		return initErr
	}
	if clientErr := validateGrpcClient(client); clientErr != nil {
		return clientErr
	}
	return nil
}
