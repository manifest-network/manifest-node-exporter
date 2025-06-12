//go:build manifest_excluded_supply_exporter
// +build manifest_excluded_supply_exporter

package manifestd

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	"golang.org/x/sync/errgroup"

	bankv1beta1 "cosmossdk.io/api/cosmos/bank/v1beta1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"resty.dev/v3"

	"github.com/liftedinit/manifest-node-exporter/pkg"
	"github.com/liftedinit/manifest-node-exporter/pkg/client"
	"github.com/liftedinit/manifest-node-exporter/pkg/collectors"
	"github.com/liftedinit/manifest-node-exporter/pkg/utils"
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

func NewExcludedSupplyCollector(client *client.GRPCClient, endpoint, denom string) *ExcludedSupplyCollector {
	var initialError error
	if client == nil {
		initialError = status.Error(codes.Internal, "gRPC client is nil")
	} else if client.Conn == nil {
		initialError = status.Error(codes.Internal, "gRPC client connection is nil")
	}

	return &ExcludedSupplyCollector{
		grpcClient:    client,
		addrsEndpoint: endpoint,
		restyClient:   resty.New().SetHeader("Accept", "application/json").SetTimeout(pkg.ClientTimeout).SetRetryCount(pkg.ClientRetry),
		initialError:  initialError,
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
	if err := utils.DoJSONRequest(c.restyClient, c.addrsEndpoint, &addrs); err != nil {
		slog.Error("Failed to fetch addresses", "endpoint", c.addrsEndpoint, "error", err)
		collectors.ReportUpMetric(ch, c.upDesc, 0)
		collectors.ReportInvalidMetric(ch, c.excludedSupplyDesc, err)
		return
	}

	const rpcTimeout = 2 * time.Second
	eg, egCtx := errgroup.WithContext(c.grpcClient.Ctx)
	results := make(chan *big.Int, len(addrs))

	bankClient := bankv1beta1.NewQueryClient(c.grpcClient.Conn)
	for _, addr := range addrs {
		addr := addr
		eg.Go(func() error {
			callCtx, cancel := context.WithTimeout(egCtx, rpcTimeout)
			defer cancel()

			resp, err := bankClient.Balance(callCtx, &bankv1beta1.QueryBalanceRequest{
				Address: addr,
				Denom:   c.denom,
			})
			if err != nil {
				slog.Error("Failed to query balance", "address", addr, "error", err)
				return err
			}
			if v, ok := new(big.Int).SetString(resp.Balance.Amount, 10); ok {
				results <- v
			} else {
				return fmt.Errorf("invalid coin amount for address %s: %s", addr, resp.Balance.Amount)
			}

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		collectors.ReportUpMetric(ch, c.upDesc, 0)
		collectors.ReportInvalidMetric(ch, c.excludedSupplyDesc, err)
		close(results)
		return
	}
	close(results)

	total := new(big.Int)
	for v := range results {
		total.Add(total, v)
	}

	collectors.ReportUpMetric(ch, c.upDesc, 1)
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
