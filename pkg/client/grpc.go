package client

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	grpcInsecure "google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

var keepaliveParams = keepalive.ClientParameters{
	Time:                60 * time.Second,
	Timeout:             30 * time.Second,
	PermitWithoutStream: true,
}

type GRPCClient struct {
	Ctx  context.Context
	Conn *grpc.ClientConn
}

func NewGRPCClient(ctx context.Context, address string, insecure bool) (*GRPCClient, error) {
	slog.Info("Initializing gRPC client pool...")
	conn, err := dial(address, insecure)
	if err != nil {
		return nil, fmt.Errorf("unable to dial: %w", err)
	}

	return &GRPCClient{
		Ctx:  ctx,
		Conn: conn,
	}, nil
}

func dial(address string, insecure bool) (*grpc.ClientConn, error) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithKeepaliveParams(keepaliveParams))
	if insecure {
		opts = append(opts, grpc.WithTransportCredentials(grpcInsecure.NewCredentials()))
	} else {
		creds := credentials.NewClientTLSFromCert(nil, "")
		opts = append(opts, grpc.WithTransportCredentials(creds))
	}

	conn, err := grpc.NewClient(address, opts...)
	if err != nil {
		return nil, fmt.Errorf("unable to dial context: %w", err)
	}

	return conn, nil
}
