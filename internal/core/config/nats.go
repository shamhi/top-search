package config

import (
	"time"

	"github.com/spf13/viper"
)

type NatsConfig interface {
	URL() string
	Name() string
	ConnectTimeout() time.Duration
	ReconnectWait() time.Duration
	MaxReconnect() int
	RetryOnFailedConnect() bool
	StreamName() string
	ConsumerName() string
	Subject() string
}

type natsConfig struct {
	url            string
	name           string
	connectTimeout time.Duration
	reconnectWait  time.Duration
	maxReconnect   int
	retryOnFailed  bool
	streamName     string
	consumerName   string
	subject        string
}

func NewNatsConfig() NatsConfig {
	return &natsConfig{
		url:            viper.GetString("nats.url"),
		name:           viper.GetString("nats.name"),
		connectTimeout: viper.GetDuration("nats.connect-timeout"),
		reconnectWait:  viper.GetDuration("nats.reconnect-wait"),
		maxReconnect:   viper.GetInt("nats.max-reconnect"),
		retryOnFailed:  viper.GetBool("nats.retry-on-failed-connect"),
		streamName:     viper.GetString("nats.stream-name"),
		consumerName:   viper.GetString("nats.consumer-name"),
		subject:        viper.GetString("nats.subject"),
	}
}

func (c *natsConfig) URL() string                   { return c.url }
func (c *natsConfig) Name() string                  { return c.name }
func (c *natsConfig) ConnectTimeout() time.Duration { return c.connectTimeout }
func (c *natsConfig) ReconnectWait() time.Duration  { return c.reconnectWait }
func (c *natsConfig) MaxReconnect() int             { return c.maxReconnect }
func (c *natsConfig) RetryOnFailedConnect() bool    { return c.retryOnFailed }
func (c *natsConfig) StreamName() string            { return c.streamName }
func (c *natsConfig) ConsumerName() string          { return c.consumerName }
func (c *natsConfig) Subject() string               { return c.subject }
