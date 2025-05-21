package manifest_current_supply_exporter

import (
	"github.com/spf13/cobra"

	"github.com/liftedinit/manifest-node-exporter/cmd"
)

var Version = "dev"

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:     "manifest-current-supply-exporter",
	Short:   "Manifest Prometheus current supply exporter",
	Long:    `Export Prometheus current supply.`,
	Version: Version,
}

func init() {
	cmd.BindGlobalFlags(RootCmd)
}

// Execute is called by main.main().
func Execute() {
	cmd.Execute(RootCmd, "manifest-current-supply-exporter")
}
