package cmd_test

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/liftedinit/manifest-node-exporter/cmd"
)

func TestInvalidAddress(t *testing.T) {
	tests := []struct {
		name string
		addr string
		err  string
	}{
		{"empty address", "", "gRPC address cannot be empty"},
		{"missing port", "localhost", "expected host:port"},
		{"invalid port number", "localhost:99999", "expected a valid port number"},
		{"invalid format", "localhost:port", "expected a valid port number"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Using reflection to access the unexported function
			cmd.RootCmd.SetArgs([]string{"serve", tt.addr})
			err := cmd.RootCmd.Execute()
			require.Error(t, err)
			require.ErrorContains(t, err, tt.err)
		})
	}
}

func TestServeCommand_InvalidArgs(t *testing.T) {
	// Reset viper values after the test
	defer viper.Reset()

	cmd.RootCmd.SetArgs([]string{"serve"})
	err := cmd.RootCmd.Execute()
	require.Error(t, err, "Expected error when no gRPC address is provided")

	cmd.RootCmd.SetArgs([]string{"serve", "invalid-addr"})
	err = cmd.RootCmd.Execute()
	require.Error(t, err, "Expected error with invalid gRPC address")
}
