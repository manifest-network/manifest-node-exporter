package test_utils

import (
	"context"
	"net"
	"testing"

	"github.com/liftedinit/manifest-node-exporter/pkg/client"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func SetupMockGrpcClient(t *testing.T) *client.GRPCClient {
	ctx := context.Background()
	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	return &client.GRPCClient{
		Ctx:  ctx,
		Conn: conn,
	}
}
