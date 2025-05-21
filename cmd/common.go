package cmd

import (
	"fmt"
	"log/slog"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	validLogLevels = map[string]slog.Level{
		"debug": slog.LevelDebug,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
	}
	validLogLevelsStr = strings.Join(slices.Sorted(maps.Keys(validLogLevels)), "|")
)

// BindGlobalFlags attaches common flags to a cobra command.
func BindGlobalFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringP("logLevel", "l", "info",
		fmt.Sprintf("set log level (%s)", validLogLevelsStr))
	if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
		slog.Error("Failed to bind flags", "error", err)
	}
}

// InitConfig loads config file and environment settings.
func InitConfig(appName string) {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/." + appName)
	viper.AddConfigPath("/etc/" + appName)

	viper.SetEnvPrefix(strings.ToUpper(appName))
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		slog.Info("Using config file", "file", viper.ConfigFileUsed())
	}
}

// PreRunLogLevel enforces the selected log level before command execution.
func PreRunLogLevel(cmd *cobra.Command, args []string) error {
	levelName := viper.GetString("logLevel")
	level, ok := validLogLevels[levelName]
	if !ok {
		return fmt.Errorf("invalid log level: %s. valid levels: %s", levelName, validLogLevelsStr)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)
	return nil
}

// Execute wraps cobra.Execute with PreRun and exit handling.
func Execute(root *cobra.Command, appName string) {
	root.PreRunE = PreRunLogLevel
	InitConfig(appName)
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
