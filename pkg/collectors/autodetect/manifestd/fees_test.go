//go:build manifest_node_exporter
// +build manifest_node_exporter

package manifestd

import (
	"testing"

	basev1beta1 "cosmossdk.io/api/cosmos/base/v1beta1"
	"cosmossdk.io/math"
)

// decCoin builds a reward coin. Outstanding-reward amounts are LegacyDec values
// serialized as integers scaled by 10^18 (math.LegacyPrecision), matching what
// the distribution module returns for ValidatorOutstandingRewards.
func decCoin(denom, amount string) *basev1beta1.DecCoin {
	return &basev1beta1.DecCoin{Denom: denom, Amount: amount}
}

const (
	testDenomUMFX = "umfx"
	// Representative billing-v2 PWR denom that appears alongside umfx.
	testDenomUPWR = "factory/manifest1afk9zr2hn2jsac63h4hm60vl9z3e5u69gndzf7c99cqge3vzwjzsfmy9qj/upwr"
)

func TestExtractDenomRewards(t *testing.T) {
	// 320 and 100 expressed as LegacyDec (scaled by 10^18).
	const umfxAmount = "320000000000000000000"
	const upwrAmount = "100000000000000000000"

	tests := []struct {
		name    string
		rewards []*basev1beta1.DecCoin
		want    math.Int
		wantErr bool
	}{
		{
			name:    "multi-denom umfx and upwr returns umfx amount",
			rewards: []*basev1beta1.DecCoin{decCoin(testDenomUMFX, umfxAmount), decCoin(testDenomUPWR, upwrAmount)},
			want:    math.NewInt(320),
		},
		{
			name:    "upwr before umfx still returns umfx amount",
			rewards: []*basev1beta1.DecCoin{decCoin(testDenomUPWR, upwrAmount), decCoin(testDenomUMFX, umfxAmount)},
			want:    math.NewInt(320),
		},
		{
			name:    "single umfx returns umfx amount",
			rewards: []*basev1beta1.DecCoin{decCoin(testDenomUMFX, umfxAmount)},
			want:    math.NewInt(320),
		},
		{
			name:    "fractional umfx is truncated",
			rewards: []*basev1beta1.DecCoin{decCoin(testDenomUMFX, "320900000000000000000")},
			want:    math.NewInt(320),
		},
		{
			name:    "single upwr contributes zero",
			rewards: []*basev1beta1.DecCoin{decCoin(testDenomUPWR, upwrAmount)},
			want:    math.ZeroInt(),
		},
		{
			name:    "empty rewards contributes zero",
			rewards: nil,
			want:    math.ZeroInt(),
		},
		{
			name:    "invalid umfx amount errors",
			rewards: []*basev1beta1.DecCoin{decCoin(testDenomUMFX, "not-a-number")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractDenomRewards(tt.rewards, testDenomUMFX, "manifestvaloper1test")
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !got.Equal(tt.want) {
				t.Fatalf("got %s, want %s", got.String(), tt.want.String())
			}
		})
	}
}
