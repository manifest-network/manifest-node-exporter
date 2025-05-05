package pkg

import (
	"fmt"
	"net"
	"strconv"

	"github.com/spf13/viper"
)

type ServeConfig struct {
	ListenAddress  string `mapstructure:"listen_address"`
	Insecure       bool   `mapstructure:"insecure"`
	MaxConcurrency uint   `mapstructure:"max_concurrency"`
	MaxRetries     uint   `mapstructure:"max_retries"`
}

func (c ServeConfig) Validate() error {
	if c.MaxConcurrency == 0 {
		return fmt.Errorf("max-concurrency must be greater than 0")
	}
	if c.MaxRetries == 0 {
		return fmt.Errorf("max-retries must be greater than 0")
	}

	host, port, err := net.SplitHostPort(c.ListenAddress)
	if err != nil {
		return fmt.Errorf("invalid prometheus-addr format, expected host:port: %w", err)
	}
	if _, err := strconv.Atoi(port); err != nil {
		return fmt.Errorf("invalid port in prometheus-addr: %w", err)
	}

	if host != "" && host != "0.0.0.0" && host != "localhost" && net.ParseIP(host) == nil {
		return fmt.Errorf("invalid host in prometheus-addr: %s", host)
	}

	return nil
}

func LoadServeConfig() ServeConfig {
	return ServeConfig{
		ListenAddress:  viper.GetString("listen-address"),
		Insecure:       viper.GetBool("insecure"),
		MaxConcurrency: viper.GetUint("max-concurrency"),
		MaxRetries:     viper.GetUint("max-retries"),
	}
}
