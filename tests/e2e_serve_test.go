package tests

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/liftedinit/manifest-node-exporter/pkg"
	"github.com/liftedinit/manifest-node-exporter/pkg/collectors/grpc"
	"github.com/liftedinit/manifest-node-exporter/test_utils"
)

// TestE2EServe tests the end-to-end functionality of the server and Prometheus collectors.
func TestE2EServe(t *testing.T) {
	mockServer := test_utils.SetupMockGrpcServer()
	defer mockServer.Stop()

	mockClient := test_utils.SetupMockGrpcClient(t)
	defer mockClient.Conn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	collectors, err := grpc.RegisterCollectors(mockClient)
	require.NoError(t, err)
	require.NotEmpty(t, collectors)

	httpServer := pkg.NewMetricsServer("localhost:2112")
	defer httpServer.Shutdown(ctx)

	httpServer.Start()
	test_utils.WaitForServerReady(t, "localhost:2112", 5*time.Second)

	resp, err := http.Get("http://localhost:2112/metrics")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	cancel()

	body, err := io.ReadAll(bufio.NewReader(resp.Body))
	require.NoError(t, err)
	require.NotEmpty(t, body)

	cases := []struct {
		Name   string
		Metric string
	}{
		{"manifest_tokenomics_denom_info", `{denom="udummy",display="DUMMY_DISPLAY",name="Dummy_Name",source="grpc",symbol="Dummy_Symbol"}`},
		{"manifest_tokenomics_total_supply", `{denom="udummy",source="grpc"} 10`},
		{"manifest_tokenomics_token_number", `{source="grpc"} 102`},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			require.Contains(t, string(body), c.Metric)
		})
	}
}
