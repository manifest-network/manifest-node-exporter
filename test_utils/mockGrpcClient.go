package test_utils

import (
	"context"
	"net"
	"testing"

	"github.com/liftedinit/manifest-node-exporter/pkg"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func SetupMockGrpcClient(t *testing.T) *pkg.GRPCClient {
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()), // Use insecure for testing
	)
	require.NoError(t, err)
	return &pkg.GRPCClient{
		Ctx:  ctx,
		Conn: conn,
	}
}
