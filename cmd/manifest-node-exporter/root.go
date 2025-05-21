package manifest_node_exporter

import (
	"github.com/liftedinit/manifest-node-exporter/cmd"
	"github.com/spf13/cobra"
)

var Version = "dev"

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:     "manifest-node-exporter",
	Short:   "Manifest Prometheus node exporter",
	Long:    `Export Prometheus metrics for the Manifest Network node.`,
	Version: Version,
}

func init() {
	cmd.BindGlobalFlags(RootCmd)
}

// Execute is called by main.main().
func Execute() {
	cmd.Execute(RootCmd, "manifest-node-exporter")
}
