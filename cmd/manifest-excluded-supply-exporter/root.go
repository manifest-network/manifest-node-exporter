package manifest_excluded_supply_exporter

import (
	"github.com/spf13/cobra"

	"github.com/liftedinit/manifest-node-exporter/cmd"
)

var Version = "dev"

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:     "manifest-excluded-supply-exporter",
	Short:   "Manifest Prometheus excluded supply exporter",
	Long:    `Export Prometheus excluded supply. The excluded supply is subtracted from the total supply to obtain the circulating supply`,
	Version: Version,
}

func init() {
	cmd.BindGlobalFlags(RootCmd)
}

// Execute is called by main.main().
func Execute() {
	cmd.Execute(RootCmd, "manifest-excluded-supply-exporter")
}
