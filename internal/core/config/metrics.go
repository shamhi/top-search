package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type MetricsConfig interface {
	Enabled() bool
	Port() int
	Addr() string
	ShutdownTimeout() time.Duration
}

type metricsConfig struct {
	enabled         bool
	port            int
	shutdownTimeout time.Duration
}

func NewMetricsConfig() MetricsConfig {
	return &metricsConfig{
		enabled:         viper.GetBool("metrics.enabled"),
		port:            viper.GetInt("metrics.port"),
		shutdownTimeout: viper.GetDuration("metrics.shutdown-timeout"),
	}
}

func (c *metricsConfig) Enabled() bool                  { return c.enabled }
func (c *metricsConfig) Port() int                      { return c.port }
func (c *metricsConfig) Addr() string                   { return fmt.Sprintf(":%d", c.port) }
func (c *metricsConfig) ShutdownTimeout() time.Duration { return c.shutdownTimeout }
