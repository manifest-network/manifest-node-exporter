//go:build manifest_node_exporter
// +build manifest_node_exporter

package ghostcloudd

import (
	"log/slog"

	gcv1beta1 "github.com/liftedinit/ghostcloud/x/ghostcloud/types"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/manifest-network/manifest-node-exporter/pkg/client"
	"github.com/manifest-network/manifest-node-exporter/pkg/collectors"
)

// WebsiteCountCollector collects the total number of deployed website from Ghostcloud.
type WebsiteCountCollector struct {
	grpcClient       *client.GRPCClient
	websiteCountDesc *prometheus.Desc // Website count
	upDesc           *prometheus.Desc // gRPC query success
	initialError     error
}

// NewWebsiteCountCollector creates a new WebsiteCountCollector.
// It requires a gRPC client connection to query the bank module.
func NewWebsiteCountCollector(client *client.GRPCClient) *WebsiteCountCollector {
	var initialError error
	if client == nil {
		initialError = status.Error(codes.Internal, "gRPC client is nil")
	} else if client.Conn == nil {
		initialError = status.Error(codes.Internal, "gRPC client connection is nil")
	}

	return &WebsiteCountCollector{
		grpcClient:   client,
		initialError: initialError,
		websiteCountDesc: prometheus.NewDesc(
			prometheus.BuildFQName("ghostcloud", "deployment", "website_count"),
			"Total number of deployed websites.",
			[]string{},
			prometheus.Labels{"source": "grpc"},
		),
		upDesc: prometheus.NewDesc(
			prometheus.BuildFQName("ghostcloud", "deployment", "count_grpc_up"),
			"Whether the gRPC query was successful.",
			nil,
			prometheus.Labels{"source": "grpc", "queries": "QueryMetas"},
		),
	}
}

// Describe implements the prometheus.Collector interface.
func (c *WebsiteCountCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.websiteCountDesc
	ch <- c.upDesc
}

// Collect implements the prometheus.Collector interface.
func (c *WebsiteCountCollector) Collect(ch chan<- prometheus.Metric) {
	// Check for initialization or connection errors first.
	if err := collectors.ValidateClient(c.grpcClient, c.initialError); err != nil {
		collectors.ReportUpMetric(ch, c.upDesc, 0) // Report gRPC down
		collectors.ReportInvalidMetric(ch, c.websiteCountDesc, err)
		return
	}

	gcQueryClient := gcv1beta1.NewQueryClient(c.grpcClient.Conn)
	metaResp, metaErr := gcQueryClient.Metas(c.grpcClient.Ctx, &gcv1beta1.QueryMetasRequest{})
	if metaErr != nil {
		slog.Error("Failed to query via gRPC", "query", "metadata", "error", metaErr)
	}

	// Report 'up' metric based on query success
	upValue := 0.0
	if metaErr == nil {
		upValue = 1.0
	}
	collectors.ReportUpMetric(ch, c.upDesc, upValue)

	if metaResp == nil {
		collectors.ReportInvalidMetric(ch, c.websiteCountDesc, status.Error(codes.Internal, "QueryMetas response is nil"))
		return
	}

	if metaResp.Pagination == nil {
		collectors.ReportInvalidMetric(ch, c.websiteCountDesc, status.Error(codes.Internal, "Pagination response is nil"))
		return
	}

	metric, err := prometheus.NewConstMetric(
		c.websiteCountDesc,
		prometheus.GaugeValue,
		float64(metaResp.Pagination.Total),
	)
	if err != nil {
		slog.Error("Failed to create total website count metric", "error", err)
	} else {
		ch <- metric
	}
}

func init() {
	RegisterCollectorFactory("website_count", func(grpcClient *client.GRPCClient, extra ...interface{}) prometheus.Collector {
		return NewWebsiteCountCollector(grpcClient)
	})
}
