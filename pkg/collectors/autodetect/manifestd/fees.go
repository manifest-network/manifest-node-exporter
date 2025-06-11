//go:build manifest_node_exporter
// +build manifest_node_exporter

package manifestd

import (
	"context"
	"log/slog"
	"math/big"
	"time"

	distributionv1beta1 "cosmossdk.io/api/cosmos/distribution/v1beta1"
	stakingv1beta1 "cosmossdk.io/api/cosmos/staking/v1beta1"
	"cosmossdk.io/math"
	"github.com/liftedinit/manifest-node-exporter/pkg/client"
	"github.com/liftedinit/manifest-node-exporter/pkg/collectors"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type FeesCollector struct {
	grpcClient   *client.GRPCClient
	feesDesc     *prometheus.Desc
	upDesc       *prometheus.Desc
	denom        string
	initialError error
}

func NewFeesCollector(client *client.GRPCClient, denom string) *FeesCollector {
	var initialError error
	if client == nil {
		initialError = status.Error(codes.Internal, "gRPC client is nil")
	} else if client.Conn == nil {
		initialError = status.Error(codes.Internal, "gRPC client connection is nil")
	}

	return &FeesCollector{
		grpcClient:   client,
		initialError: initialError,
		denom:        denom,
		feesDesc: prometheus.NewDesc(
			prometheus.BuildFQName("manifest", "tokenomics", "fees"),
			"Transaction fees locked in validators.",
			[]string{"amount", "denom"},
			prometheus.Labels{"source": "grpc"},
		),
		upDesc: prometheus.NewDesc(
			prometheus.BuildFQName("manifest", "tokenomics", "fees_grpc_up"),
			"Whether the gRPC query was successful.",
			nil,
			prometheus.Labels{"source": "grpc", "queries": "Validators, ValidatorOutstandingRewards"},
		),
	}
}

func (c *FeesCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.feesDesc
	ch <- c.upDesc
}

func (c *FeesCollector) Collect(ch chan<- prometheus.Metric) {
	// Check for initialization or connection errors first.
	if err := collectors.ValidateClient(c.grpcClient, c.initialError); err != nil {
		collectors.ReportUpMetric(ch, c.upDesc, 0) // Report gRPC down
		collectors.ReportInvalidMetric(ch, c.feesDesc, err)
		return
	}

	stakingQueryClient := stakingv1beta1.NewQueryClient(c.grpcClient.Conn)
	validatorsResp, validatorsErr := stakingQueryClient.Validators(c.grpcClient.Ctx, &stakingv1beta1.QueryValidatorsRequest{})
	if validatorsErr != nil {
		slog.Error("Failed to query via gRPC", "query", "Validators", "error", validatorsErr)
		collectors.ReportUpMetric(ch, c.upDesc, 0)
		collectors.ReportInvalidMetric(ch, c.feesDesc, validatorsErr)
		return
	}

	if validatorsResp == nil || validatorsResp.Validators == nil || len(validatorsResp.Validators) == 0 {
		collectors.ReportUpMetric(ch, c.upDesc, 0)
		collectors.ReportInvalidMetric(ch, c.feesDesc, status.Error(codes.Internal, "Validators response is nil or empty"))
		return
	}

	distributionQueryClient := distributionv1beta1.NewQueryClient(c.grpcClient.Conn)
	const rpcTimeout = 2 * time.Second
	eg, egCtx := errgroup.WithContext(c.grpcClient.Ctx)
	results := make(chan math.Int, len(validatorsResp.Validators))
	for _, val := range validatorsResp.Validators {
		val := val
		eg.Go(func() error {
			callCtx, callCancel := context.WithTimeout(egCtx, rpcTimeout)
			defer callCancel()

			feesResp, feesErr := distributionQueryClient.ValidatorOutstandingRewards(callCtx, &distributionv1beta1.QueryValidatorOutstandingRewardsRequest{ValidatorAddress: val.OperatorAddress})
			if feesErr != nil {
				slog.Error("Failed to query via gRPC", "query", "ValidatorOutstandingRewards", "validator", val.OperatorAddress, "error", feesErr)
				return feesErr
			}
			if feesResp == nil || feesResp.Rewards == nil {
				slog.Error("ValidatorOutstandingRewards response is nil or empty", "validator", val.OperatorAddress)
				return status.Error(codes.Internal, "ValidatorOutstandingRewards response is nil or empty")
			}
			if len(feesResp.Rewards.Rewards) != 1 {
				slog.Warn("ValidatorOutstandingRewards response has no rewards or too many rewards", "validator", val.OperatorAddress)
				return nil
			}
			denom := feesResp.Rewards.Rewards[0].Denom
			if denom != c.denom {
				slog.Warn("ValidatorOutstandingRewards response has different denom", "validator", val.OperatorAddress, "expected", c.denom, "got", denom)
				return status.Error(codes.InvalidArgument, "denom mismatch for validator "+val.OperatorAddress+": expected "+c.denom+", got "+denom)
			}

			// Convert the amount to a big.Int
			amount, ok := new(big.Int).SetString(feesResp.Rewards.Rewards[0].Amount, 10)
			if !ok {
				slog.Error("Failed to parse coin amount", "validator", val.OperatorAddress, "amount", feesResp.Rewards.Rewards[0].Amount)
				return status.Error(codes.Internal, "invalid coin amount for validator "+val.OperatorAddress+": "+feesResp.Rewards.Rewards[0].Amount)
			}

			// And create a LegacyDec from it, using the LegacyPrecision
			legacyAmount := math.LegacyNewDecFromBigIntWithPrec(amount, math.LegacyPrecision)

			// And only keep the integer part
			truncatedAmount := legacyAmount.TruncateInt()
			results <- truncatedAmount

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		collectors.ReportUpMetric(ch, c.upDesc, 0)
		collectors.ReportInvalidMetric(ch, c.feesDesc, err)
		close(results)
		return
	}
	close(results)

	total := math.ZeroInt()
	for v := range results {
		total = total.Add(v)
	}

	collectors.ReportUpMetric(ch, c.upDesc, 1)
	m, err := prometheus.NewConstMetric(c.feesDesc, prometheus.GaugeValue, 1, total.String(), c.denom)
	if err != nil {
		slog.Error("Failed to create fees metric", "error", err)
		collectors.ReportInvalidMetric(ch, c.feesDesc, err)
	} else {
		ch <- m
	}
}

func init() {
	RegisterCollectorFactory("fees", func(client *client.GRPCClient, extra ...interface{}) prometheus.Collector {
		return NewFeesCollector(client, "umfx")
	})
}
