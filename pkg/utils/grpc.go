package utils

import (
	"context"
	"log/slog"
	"time"

	nodev1beta1 "cosmossdk.io/api/cosmos/base/node/v1beta1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func GetNodeStatus(target string) (*nodev1beta1.StatusResponse, error) {
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

	client := nodev1beta1.NewServiceClient(conn)

	resp, err := client.Status(ctx, &nodev1beta1.StatusRequest{})
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
	slog.Debug("gRPC port is ready and responding", "target", target, "status", resp)
	return true
}
