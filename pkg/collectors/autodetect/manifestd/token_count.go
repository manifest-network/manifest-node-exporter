//go:build manifest_node_exporter
// +build manifest_node_exporter

package manifestd

import (
	"log/slog"

	bankv1beta1 "cosmossdk.io/api/cosmos/bank/v1beta1"
	queryv1beta1 "cosmossdk.io/api/cosmos/base/query/v1beta1"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/manifest-network/manifest-node-exporter/pkg/client"
	"github.com/manifest-network/manifest-node-exporter/pkg/collectors"
)

// TokenCountCollector collects the total number of denominations from the Cosmos SDK bank module via gRPC.
type TokenCountCollector struct {
	grpcClient     *client.GRPCClient
	tokenCountDesc *prometheus.Desc // Token count
	upDesc         *prometheus.Desc // gRPC query success
	initialError   error
}

// NewTokenCountCollector creates a new TokenCountCollector.
// It requires a gRPC client connection to query the bank module.
func NewTokenCountCollector(client *client.GRPCClient) *TokenCountCollector {
	var initialError error
	if client == nil {
		initialError = status.Error(codes.Internal, "gRPC client is nil")
	} else if client.Conn == nil {
		initialError = status.Error(codes.Internal, "gRPC client connection is nil")
	}

	return &TokenCountCollector{
		grpcClient:   client,
		initialError: initialError,
		tokenCountDesc: prometheus.NewDesc(
			prometheus.BuildFQName("manifest", "tokenomics", "token_count"),
			"Total number of denominations, including native, IBC and factory tokens.",
			[]string{},
			prometheus.Labels{"source": "grpc"},
		),
		upDesc: prometheus.NewDesc(
			prometheus.BuildFQName("manifest", "tokenomics", "count_grpc_up"),
			"Whether the gRPC query was successful.",
			nil,
			prometheus.Labels{"source": "grpc", "queries": "DenomsMetadata"},
		),
	}
}

// Describe implements the prometheus.Collector interface.
func (c *TokenCountCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.tokenCountDesc
	ch <- c.upDesc
}

// Collect implements the prometheus.Collector interface.
func (c *TokenCountCollector) Collect(ch chan<- prometheus.Metric) {
	// Check for initialization or connection errors first.
	if err := collectors.ValidateClient(c.grpcClient, c.initialError); err != nil {
		collectors.ReportUpMetric(ch, c.upDesc, 0) // Report gRPC down
		collectors.ReportInvalidMetric(ch, c.tokenCountDesc, err)
		return
	}

	bankQueryClient := bankv1beta1.NewQueryClient(c.grpcClient.Conn)
	denomsMetaResp, denomsMetaErr := bankQueryClient.DenomsMetadata(c.grpcClient.Ctx, &bankv1beta1.QueryDenomsMetadataRequest{Pagination: &queryv1beta1.PageRequest{CountTotal: true}})
	if denomsMetaErr != nil {
		slog.Error("Failed to query via gRPC", "query", "DenomsMetadata", "error", denomsMetaErr)
	}

	// Report 'up' metric based on query success
	upValue := 0.0
	if denomsMetaErr == nil {
		upValue = 1.0
	}
	collectors.ReportUpMetric(ch, c.upDesc, upValue)

	if denomsMetaResp == nil {
		collectors.ReportInvalidMetric(ch, c.tokenCountDesc, status.Error(codes.Internal, "DenomsMetadata response is nil"))
		return
	}

	if denomsMetaResp.Pagination == nil {
		collectors.ReportInvalidMetric(ch, c.tokenCountDesc, status.Error(codes.Internal, "Pagination response is nil"))
		return
	}

	metric, err := prometheus.NewConstMetric(
		c.tokenCountDesc,
		prometheus.GaugeValue,
		float64(denomsMetaResp.Pagination.Total),
	)
	if err != nil {
		slog.Error("Failed to create total token count metric", "error", err)
	} else {
		ch <- metric
	}
}

func init() {
	RegisterCollectorFactory("token_count", func(grpcClient *client.GRPCClient, extra ...interface{}) prometheus.Collector {
		return NewTokenCountCollector(grpcClient)
	})
}
