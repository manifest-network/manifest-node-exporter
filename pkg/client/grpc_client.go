package client

import (
	"context"
	"log/slog"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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
	conn := dial(ctx, address, insecure)

	return &GRPCClient{
		Ctx:  ctx,
		Conn: conn,
	}, nil
}

func dial(ctx context.Context, address string, insecure bool) *grpc.ClientConn {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithKeepaliveParams(keepaliveParams))
	if insecure {
		opts = append(opts, grpc.WithInsecure())
	} else {
		creds := credentials.NewClientTLSFromCert(nil, "")
		opts = append(opts, grpc.WithTransportCredentials(creds))
	}

	conn, err := grpc.DialContext(ctx, address, opts...)
	if err != nil {
		slog.Error("Failed to connect", "error", err)
		os.Exit(1)
	}

	return conn
}
