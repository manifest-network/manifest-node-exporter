//go:build manifest_node_exporter
// +build manifest_node_exporter

package manifestd

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	distributionv1beta1 "cosmossdk.io/api/cosmos/distribution/v1beta1"
	stakingv1beta1 "cosmossdk.io/api/cosmos/staking/v1beta1"
	"cosmossdk.io/math"
	"github.com/manifest-network/manifest-node-exporter/pkg/client"
	"github.com/manifest-network/manifest-node-exporter/pkg/collectors"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type FeesCollector struct {
	grpcClient   *client.GRPCClient
	feesDesc     *prometheus.Desc
	upDesc       *prometheus.Desc
	errorsDesc   *prometheus.Desc
	denom        string
	initialError error

	cache     map[string]math.Int
	cacheMu   sync.RWMutex
	stateFile string

	validatorErrsTotal uint64
}

type feesState map[string]string

func defaultFeesStatePath(denom string) string {
	base, err := os.UserCacheDir()
	if err != nil || base == "" {
		home, _ := os.UserHomeDir()
		if home == "" {
			base = "."
		} else {
			base = filepath.Join(home, ".cache")
		}
	}
	return filepath.Join(base, "manifest-node-exporter", "fees_"+denom+".json")
}

func NewFeesCollector(client *client.GRPCClient, denom string) *FeesCollector {
	var initialError error
	if client == nil {
		initialError = status.Error(codes.Internal, "gRPC client is nil")
	} else if client.Conn == nil {
		initialError = status.Error(codes.Internal, "gRPC client connection is nil")
	}

	c := &FeesCollector{
		grpcClient:         client,
		initialError:       initialError,
		validatorErrsTotal: 0,
		denom:              denom,
		cache:              make(map[string]math.Int),
		stateFile:          defaultFeesStatePath(denom),
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
		errorsDesc: prometheus.NewDesc(
			prometheus.BuildFQName("manifest", "tokenomics", "fees_validator_failures_total"),
			"Total per-validator gRPC failures since process start.",
			nil,
			prometheus.Labels{"source": "grpc", "denom": denom},
		),
	}

	if err := c.loadState(); err != nil {
		slog.Warn("failed to load fees cache", "path", c.stateFile, "error", err)
	}

	return c
}

func (c *FeesCollector) loadState() error {
	b, err := os.ReadFile(c.stateFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	var st feesState
	if err := json.Unmarshal(b, &st); err != nil {
		return err
	}
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	for addr, s := range st {
		if v, ok := math.NewIntFromString(s); ok {
			c.cache[addr] = v
		}
	}
	return nil
}

func (c *FeesCollector) saveState() error {
	// snapshot cache under read lock
	c.cacheMu.RLock()
	st := make(feesState, len(c.cache))
	for addr, v := range c.cache {
		st[addr] = v.String()
	}
	c.cacheMu.RUnlock()

	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(c.stateFile), 0o755); err != nil {
		return err
	}
	tmp := c.stateFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, c.stateFile)
}

func (c *FeesCollector) emitUpAndErrors(ch chan<- prometheus.Metric, up float64) {
	collectors.ReportUpMetric(ch, c.upDesc, up)

	failures := atomic.LoadUint64(&c.validatorErrsTotal)
	errMetric, err := prometheus.NewConstMetric(c.errorsDesc, prometheus.CounterValue, float64(failures))
	if err != nil {
		slog.Error("failed to create validator failure metric", "error", err)
		collectors.ReportInvalidMetric(ch, c.errorsDesc, err)
		return
	}
	ch <- errMetric
}

func (c *FeesCollector) emitFromDisk(ch chan<- prometheus.Metric, up float64) {
	c.emitUpAndErrors(ch, up)

	b, err := os.ReadFile(c.stateFile)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			slog.Warn("failed to read fees cache from disk", "path", c.stateFile, "error", err)
		}
		return
	}

	var st feesState
	if err := json.Unmarshal(b, &st); err != nil {
		slog.Warn("failed to unmarshal fees cache from disk", "path", c.stateFile, "error", err)
		return
	}

	total := math.ZeroInt()
	for _, s := range st {
		if v, ok := math.NewIntFromString(s); ok {
			total = total.Add(v)
		}
	}

	m, err := prometheus.NewConstMetric(c.feesDesc, prometheus.GaugeValue, 1, total.String(), c.denom)
	if err != nil {
		slog.Error("failed to create fees metric", "error", err)
		collectors.ReportInvalidMetric(ch, c.feesDesc, err)
		return
	}
	ch <- m
}

func (c *FeesCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.feesDesc
	ch <- c.upDesc
	ch <- c.errorsDesc
}

func (c *FeesCollector) Collect(ch chan<- prometheus.Metric) {
	// Check for initialization or connection errors first.
	if err := collectors.ValidateClient(c.grpcClient, c.initialError); err != nil {
		c.emitFromDisk(ch, 0)
		return
	}

	stakingQueryClient := stakingv1beta1.NewQueryClient(c.grpcClient.Conn)
	validatorsResp, validatorsErr := stakingQueryClient.Validators(c.grpcClient.Ctx, &stakingv1beta1.QueryValidatorsRequest{})
	if validatorsErr != nil {
		slog.Error("Failed to query via gRPC", "query", "Validators", "error", validatorsErr)
		c.emitFromDisk(ch, 0)
		return
	}

	if validatorsResp == nil || validatorsResp.Validators == nil {
		slog.Error("Validators response is nil")
		c.emitFromDisk(ch, 0)
		return
	}

	if len(validatorsResp.Validators) == 0 {
		slog.Warn("Validators response has no validators")
		c.emitFromDisk(ch, 1)
		return
	}

	distributionQueryClient := distributionv1beta1.NewQueryClient(c.grpcClient.Conn)
	const rpcTimeout = 2 * time.Second

	var (
		eg, egCtx  = errgroup.WithContext(c.grpcClient.Ctx)
		mu         sync.Mutex
		updates    = make(map[string]math.Int)
		scrapeErrs uint64
	)

	for _, val := range validatorsResp.Validators {
		val := val
		eg.Go(func() error {
			callCtx, callCancel := context.WithTimeout(egCtx, rpcTimeout)
			defer callCancel()

			feesResp, feesErr := distributionQueryClient.ValidatorOutstandingRewards(callCtx, &distributionv1beta1.QueryValidatorOutstandingRewardsRequest{ValidatorAddress: val.OperatorAddress})
			if feesErr != nil {
				slog.Error("Failed to query via gRPC", "query", "ValidatorOutstandingRewards", "validator", val.OperatorAddress, "error", feesErr)
				atomic.AddUint64(&scrapeErrs, 1)
				return nil
			}
			if feesResp == nil || feesResp.Rewards == nil {
				slog.Error("ValidatorOutstandingRewards response is nil or empty", "validator", val.OperatorAddress)
				atomic.AddUint64(&scrapeErrs, 1)
				return nil
			}
			if len(feesResp.Rewards.Rewards) != 1 {
				slog.Warn("ValidatorOutstandingRewards response has no rewards or too many rewards", "validator", val.OperatorAddress)
				return nil
			}

			denom := feesResp.Rewards.Rewards[0].Denom
			if denom != c.denom {
				slog.Warn("ValidatorOutstandingRewards response has different denom", "validator", val.OperatorAddress, "expected", c.denom, "got", denom)
				atomic.AddUint64(&scrapeErrs, 1)
				return nil
			}

			// Convert the amount to a big.Int
			amount, ok := new(big.Int).SetString(feesResp.Rewards.Rewards[0].Amount, 10)
			if !ok {
				slog.Error("Failed to parse coin amount", "validator", val.OperatorAddress, "amount", feesResp.Rewards.Rewards[0].Amount)
				atomic.AddUint64(&scrapeErrs, 1)
				return nil
			}

			// And create a LegacyDec from it, using the LegacyPrecision
			legacyAmount := math.LegacyNewDecFromBigIntWithPrec(amount, math.LegacyPrecision)

			// And only keep the integer part
			truncatedAmount := legacyAmount.TruncateInt()
			mu.Lock()
			updates[val.OperatorAddress] = truncatedAmount
			mu.Unlock()

			return nil
		})
	}

	_ = eg.Wait()

	// Accumulate any scrape errors into the total counter
	if n := atomic.LoadUint64(&scrapeErrs); n > 0 {
		atomic.AddUint64(&c.validatorErrsTotal, n)
	}

	// Check if any updates differ from the cache
	var changed bool
	c.cacheMu.Lock()
	for addr, amt := range updates {
		prev, ok := c.cache[addr]
		if !ok || !prev.Equal(amt) {
			c.cache[addr] = amt
			changed = true
		}
	}
	c.cacheMu.Unlock()

	if changed {
		if err := c.saveState(); err != nil {
			slog.Warn("failed to persist fees cache", "path", c.stateFile, "error", err)
		}
	}

	c.emitFromDisk(ch, 1)
}

func init() {
	RegisterCollectorFactory("fees", func(client *client.GRPCClient, extra ...interface{}) prometheus.Collector {
		return NewFeesCollector(client, "umfx")
	})
}
