//go:build manifest_current_supply_exporter
// +build manifest_current_supply_exporter

package manifestd

import (
	"log/slog"
	"math/big"

	bankv1beta1 "cosmossdk.io/api/cosmos/bank/v1beta1"
	"github.com/liftedinit/manifest-node-exporter/pkg"
	"github.com/liftedinit/manifest-node-exporter/pkg/client"
	"github.com/liftedinit/manifest-node-exporter/pkg/collectors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"resty.dev/v3"
)

type ExcludedSupplyCollector struct {
	grpcClient         *client.GRPCClient
	addrsEndpoint      string
	restyClient        *resty.Client
	excludedSupplyDesc *prometheus.Desc
	upDesc             *prometheus.Desc
	denom              string
	initialError       error
}

func NewExcludedSupplyCollector(c *client.GRPCClient, endpoint, denom string) *ExcludedSupplyCollector {
	var err error
	if c == nil || c.Conn == nil {
		err = status.Error(codes.Internal, "gRPC client or connection is nil")
	}

	return &ExcludedSupplyCollector{
		grpcClient:    c,
		addrsEndpoint: endpoint,
		restyClient:   resty.New().SetHeader("Accept", "application/json").SetTimeout(pkg.ClientTimeout).SetRetryCount(pkg.ClientRetry),
		initialError:  err,
		denom:         denom,
		excludedSupplyDesc: prometheus.NewDesc(
			prometheus.BuildFQName("manifest", "tokenomics", "excluded_supply"),
			"Token supply to exclude from total supply to obtain circulating supply",
			[]string{"excluded_supply", "denom"},
			prometheus.Labels{"source": "grpc"},
		),
		upDesc: prometheus.NewDesc(
			prometheus.BuildFQName("manifest", "tokenomics", "balance_grpc_up"),
			"Whether the gRPC queries succeeded.",
			nil,
			prometheus.Labels{"source": "grpc", "query": "Balance"},
		),
	}
}

func (c *ExcludedSupplyCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.excludedSupplyDesc
	ch <- c.upDesc
}

func (c *ExcludedSupplyCollector) Collect(ch chan<- prometheus.Metric) {
	if err := collectors.ValidateClient(c.grpcClient, c.initialError); err != nil {
		collectors.ReportUpMetric(ch, c.upDesc, 0)
		collectors.ReportInvalidMetric(ch, c.excludedSupplyDesc, err)
		return
	}

	var addrs []string
	resp, err := c.restyClient.R().
		SetResult(&addrs).
		Get(c.addrsEndpoint)
	if err != nil || resp.IsError() {
		slog.Error("Failed to fetch addresses", "endpoint", c.addrsEndpoint, "error", err)
		collectors.ReportUpMetric(ch, c.upDesc, 0)
		collectors.ReportInvalidMetric(ch, c.excludedSupplyDesc, err)
		return
	}

	bankClient := bankv1beta1.NewQueryClient(c.grpcClient.Conn)
	total := big.NewInt(0)
	success := true

	for _, addr := range addrs {
		resp, err := bankClient.Balance(c.grpcClient.Ctx, &bankv1beta1.QueryBalanceRequest{
			Address: addr,
			Denom:   c.denom,
		})
		if err != nil {
			slog.Error("Failed to query Balance", "address", addr, "error", err)
			success = false
			continue
		}
		if v, ok := new(big.Int).SetString(resp.Balance.Amount, 10); ok {
			total.Add(total, v)
		} else {
			slog.Error("Invalid coin amount", "address", addr, "denom", resp.Balance.Denom, "amount", resp.Balance.Amount)
			success = false
		}
	}

	upVal := 0.0
	if success {
		upVal = 1.0
	}
	collectors.ReportUpMetric(ch, c.upDesc, upVal)

	m, err := prometheus.NewConstMetric(c.excludedSupplyDesc, prometheus.GaugeValue, 1, total.String(), c.denom)
	if err != nil {
		slog.Error("Failed to create excluded supply metric", "error", err)
	} else {
		ch <- m
	}
}

func init() {
	RegisterCollectorFactory("account_balance", func(grpcClient *client.GRPCClient, extra ...interface{}) prometheus.Collector {
		endpoint := viper.GetString("addrs-endpoint")
		return NewExcludedSupplyCollector(grpcClient, endpoint, "umfx")
	})
}
