//go:build manifest_node_exporter
// +build manifest_node_exporter

package manifestd

import (
	"log/slog"

	bankv1beta1 "cosmossdk.io/api/cosmos/bank/v1beta1"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/manifest-network/manifest-node-exporter/pkg/client"
	"github.com/manifest-network/manifest-node-exporter/pkg/collectors"
)

// DenomInfoCollector collects denom metadata and total supply metrics from the Cosmos SDK bank module via gRPC.
// Initialize the collector with the denom you want to monitor.
type DenomInfoCollector struct {
	grpcClient      *client.GRPCClient
	denom           string
	denomMetaDesc   *prometheus.Desc // Denom metadata
	upDesc          *prometheus.Desc // gRPC query success
	totalSupplyDesc *prometheus.Desc // Token supply
	initialError    error
}

// NewDenomInfoCollector creates a new DenomInfoCollector.
// It requires a gRPC client connection to query the bank module.
func NewDenomInfoCollector(client *client.GRPCClient, denom string) *DenomInfoCollector {
	var initialError error
	if client == nil {
		initialError = status.Error(codes.Internal, "gRPC client is nil")
	} else if client.Conn == nil {
		initialError = status.Error(codes.Internal, "gRPC client connection is nil")
	}
	if denom == "" {
		initialError = status.Error(codes.InvalidArgument, "denom is empty")
	}

	return &DenomInfoCollector{
		grpcClient:   client,
		initialError: initialError,
		denom:        denom,
		denomMetaDesc: prometheus.NewDesc(
			prometheus.BuildFQName("manifest", "tokenomics", "denom_metadata"),
			"Information about a Cosmos SDK denomination.",
			[]string{"symbol", "denom", "name", "display"},
			prometheus.Labels{"source": "grpc"},
		),
		totalSupplyDesc: prometheus.NewDesc(
			prometheus.BuildFQName("manifest", "tokenomics", "total_supply"),
			"Total supply of a specific denomination.",
			[]string{"denom", "supply"},
			prometheus.Labels{"source": "grpc"},
		),
		upDesc: prometheus.NewDesc(
			prometheus.BuildFQName("manifest", "tokenomics", "denom_grpc_up"),
			"Whether the gRPC query was successful.",
			nil,
			prometheus.Labels{"source": "grpc", "queries": "DenomMetadata, SupplyOf"},
		),
	}
}

// Describe implements the prometheus.Collector interface.
func (c *DenomInfoCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.denomMetaDesc
	ch <- c.totalSupplyDesc
	ch <- c.upDesc
}

// Collect implements the prometheus.Collector interface.
func (c *DenomInfoCollector) Collect(ch chan<- prometheus.Metric) {
	// Check for initialization or connection errors first.
	if err := collectors.ValidateClient(c.grpcClient, c.initialError); err != nil {
		collectors.ReportUpMetric(ch, c.upDesc, 0) // Report gRPC down
		collectors.ReportInvalidMetric(ch, c.totalSupplyDesc, err)
		collectors.ReportInvalidMetric(ch, c.denomMetaDesc, err)
		return
	}

	bankQueryClient := bankv1beta1.NewQueryClient(c.grpcClient.Conn)
	denomMetaResp, denomMetaErr := bankQueryClient.DenomMetadata(c.grpcClient.Ctx, &bankv1beta1.QueryDenomMetadataRequest{Denom: c.denom})
	if denomMetaErr != nil {
		slog.Error("Failed to query via gRPC", "query", "DenomMetadata", "error", denomMetaErr)
	}

	totalSupplyResp, totalSupplyErr := bankQueryClient.SupplyOf(c.grpcClient.Ctx, &bankv1beta1.QuerySupplyOfRequest{Denom: c.denom})
	if totalSupplyErr != nil {
		slog.Error("Failed to query via gRPC", "query", "SupplyOf", "error", totalSupplyErr)
	}

	// Report 'up' metric based on query success
	upValue := 0.0
	if denomMetaErr == nil && totalSupplyErr == nil {
		upValue = 1.0
	}
	collectors.ReportUpMetric(ch, c.upDesc, upValue)

	c.collectDenomMetadata(ch, denomMetaResp, denomMetaErr)
	c.collectTotalSupply(ch, totalSupplyResp, totalSupplyErr)
}

func (c *DenomInfoCollector) collectDenomMetadata(ch chan<- prometheus.Metric, resp *bankv1beta1.QueryDenomMetadataResponse, queryErr error) {
	if queryErr != nil {
		collectors.ReportInvalidMetric(ch, c.denomMetaDesc, queryErr)
		return
	}
	if resp == nil {
		return
	}

	metadata := resp.Metadata

	if metadata != nil {
		metric, err := prometheus.NewConstMetric(
			c.denomMetaDesc,
			prometheus.GaugeValue,
			1, // Value is 1 to indicate presence/info
			metadata.Symbol,
			metadata.Base,
			metadata.Name,
			metadata.Display,
		)
		if err != nil {
			slog.Error("Failed to create denom metadata metric", "symbol", metadata.Symbol, "base", metadata.Base, "error", err)
		} else {
			ch <- metric
		}
	}
}

func (c *DenomInfoCollector) collectTotalSupply(ch chan<- prometheus.Metric, resp *bankv1beta1.QuerySupplyOfResponse, queryErr error) {
	if queryErr != nil {
		collectors.ReportInvalidMetric(ch, c.totalSupplyDesc, queryErr)
		return
	}
	if resp == nil {
		return
	}
	coin := resp.Amount
	if coin == nil {
		slog.Warn("Total supply response is nil")
		collectors.ReportInvalidMetric(ch, c.totalSupplyDesc, status.Error(codes.Internal, "total supply response is nil"))
		return
	}

	// *IMPORTANT*
	// The metric's metadata contains the supply of the token in the base denomination.
	// The gauge value is set to 1 to indicate the presence of the metric.
	metric, err := prometheus.NewConstMetric(
		c.totalSupplyDesc,
		prometheus.GaugeValue,
		1, // Let the client handle the metadata.
		coin.Denom,
		coin.Amount,
	)
	if err != nil {
		slog.Error("Failed to create total supply metric", "denom", coin.Denom, "error", err)
	} else {
		ch <- metric
	}
}

func init() {
	RegisterCollectorFactory("denom_metadata", func(grpcClient *client.GRPCClient, extra ...interface{}) prometheus.Collector {
		return NewDenomInfoCollector(grpcClient, "umfx")
	})
}
