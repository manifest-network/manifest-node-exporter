package manifest_current_supply_exporter

import (
	"log/slog"

	_ "github.com/liftedinit/manifest-node-exporter/pkg/collectors/autodetect/manifestd" // RegisterMonitor the manifestd monitor (side-effect)
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve [flags]",
	Short: "Serve current supply metrics",
	RunE: func(cmd *cobra.Command, args []string) error {
		if parent := cmd.Parent(); parent != nil && parent.PreRunE != nil {
			if err := parent.PreRunE(parent, args); err != nil {
				return err
			}
		}
		slog.Info("Starting manifest-current-supply-exporter")

		return nil
	},
}

func init() {
	serveCmd.Flags().String("listen-address", "0.0.0.0:2112", "Address to listen on")

	if err := viper.BindPFlags(serveCmd.Flags()); err != nil {
		slog.Error("Failed to bind serveCmd flags", "error", err)
	}

	RootCmd.AddCommand(serveCmd)
}
