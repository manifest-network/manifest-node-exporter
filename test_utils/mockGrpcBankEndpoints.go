package test_utils

import (
	"context"

	bankv1beta1 "cosmossdk.io/api/cosmos/bank/v1beta1"
	queryv1beta1 "cosmossdk.io/api/cosmos/base/query/v1beta1"
	basev1beta1 "cosmossdk.io/api/cosmos/base/v1beta1"
)

// mockBankEndpoints is a mock implementation of the bankv1beta1.QueryServer interface.
type mockBankEndpoints struct {
	bankv1beta1.UnimplementedQueryServer
}

// DenomsMetadata returns a mock response for the cosmos.bank.v1beta1.DenomsMetadata gRPC endpoint.
func (s *mockBankEndpoints) DenomsMetadata(_ context.Context, _ *bankv1beta1.QueryDenomsMetadataRequest) (*bankv1beta1.QueryDenomsMetadataResponse, error) {
	return &bankv1beta1.QueryDenomsMetadataResponse{Pagination: &queryv1beta1.PageResponse{Total: 102}}, nil
}

// SupplyOf returns a mock response for the cosmos.bank.v1beta1.SupplyOf gRPC endpoint.
func (s *mockBankEndpoints) SupplyOf(_ context.Context, _ *bankv1beta1.QuerySupplyOfRequest) (*bankv1beta1.QuerySupplyOfResponse, error) {
	return &bankv1beta1.QuerySupplyOfResponse{Amount: &basev1beta1.Coin{
		Denom:  "udummy",
		Amount: "10",
	}}, nil
}

// DenomMetadata returns a mock response for the cosmos.bank.v1beta1.DenomMetadata gRPC endpoint.
func (s *mockBankEndpoints) DenomMetadata(_ context.Context, _ *bankv1beta1.QueryDenomMetadataRequest) (*bankv1beta1.QueryDenomMetadataResponse, error) {
	return &bankv1beta1.QueryDenomMetadataResponse{Metadata: &bankv1beta1.Metadata{
		Description: "Dummy_Description",
		DenomUnits: []*bankv1beta1.DenomUnit{
			{Denom: "udummy", Exponent: 0},
			{Denom: "DUMMY", Exponent: 6},
		},
		Base:    "udummy",
		Display: "DUMMY_DISPLAY",
		Name:    "Dummy_Name",
		Symbol:  "Dummy_Symbol",
		Uri:     "Dummy_Uri",
		UriHash: "Dummy_Uri_Hash",
	}}, nil
}
