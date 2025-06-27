package utils

import (
	"context"
	"log/slog"
	"time"

	tmv1beta1 "cosmossdk.io/api/cosmos/base/tendermint/v1beta1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func GetNodeStatus(target string) (*tmv1beta1.GetNodeInfoResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := tmv1beta1.NewServiceClient(conn)

	resp, err := client.GetNodeInfo(ctx, &tmv1beta1.GetNodeInfoRequest{})
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func IsGrpcPort(target string) bool {
	resp, err := GetNodeStatus(target)
	if err != nil || resp == nil {
		return false
	}
	if resp.ApplicationVersion == nil {
		slog.Warn("gRPC port is not responding as expected", "target", target, "error", "ApplicationVersion is nil")
		return false
	}

	cosmosSdkVersion := resp.ApplicationVersion.CosmosSdkVersion
	if cosmosSdkVersion == "" {
		slog.Warn("gRPC port is not responding as expected", "target", target, "error", "Cosmos SDK version is empty")
		return false
	}

	slog.Debug("gRPC port is ready and responding", "target", target, "status", resp, "cosmosSdkVersion", cosmosSdkVersion)
	return true
}
