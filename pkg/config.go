package pkg

import (
	"fmt"
	"net"
	"strconv"

	"github.com/spf13/viper"
)

type ServeConfig struct {
	ListenAddress string `mapstructure:"listen_address"`
}

func (c ServeConfig) Validate() error {
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
		ListenAddress: viper.GetString("listen-address"),
	}
}
