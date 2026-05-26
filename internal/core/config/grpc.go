package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type GrpcConfig interface {
	Port() int
	Addr() string

	ConnectionTimeout() time.Duration
	ShutdownTimeout() time.Duration
	RequestTimeout() time.Duration

	MaxConnectionsAge() time.Duration
	MaxConcurrentStreams() uint32
	MaxReceiveMessageSize() int
	MaxSendMessageSize() int

	KeepAliveTime() time.Duration
	KeepAliveTimeout() time.Duration
}

type grpcConfig struct {
	port int

	connectionTimeout time.Duration
	shutdownTimeout   time.Duration
	requestTimeout    time.Duration

	maxConnectionsAge     time.Duration
	maxConcurrentStreams  uint32
	maxReceiveMessageSize int
	maxSendMessageSize    int

	keepAliveTime    time.Duration
	keepAliveTimeout time.Duration
}

func NewGrpcConfig() GrpcConfig {
	return &grpcConfig{
		port:                  viper.GetInt("grpc.port"),
		connectionTimeout:     viper.GetDuration("grpc.connection-timeout"),
		shutdownTimeout:       viper.GetDuration("grpc.shutdown-timeout"),
		requestTimeout:        viper.GetDuration("grpc.request-timeout"),
		maxConnectionsAge:     viper.GetDuration("grpc.max-connections-age"),
		maxConcurrentStreams:  viper.GetUint32("grpc.max-concurrent-streams"),
		maxReceiveMessageSize: viper.GetInt("grpc.max-receive-message-size"),
		maxSendMessageSize:    viper.GetInt("grpc.max-send-message-size"),
		keepAliveTime:         viper.GetDuration("grpc.keepalive-time"),
		keepAliveTimeout:      viper.GetDuration("grpc.keepalive-timeout"),
	}
}

func (c *grpcConfig) Port() int                        { return c.port }
func (c *grpcConfig) Addr() string                     { return fmt.Sprintf(":%d", c.port) }
func (c *grpcConfig) ConnectionTimeout() time.Duration { return c.connectionTimeout }
func (c *grpcConfig) ShutdownTimeout() time.Duration   { return c.shutdownTimeout }
func (c *grpcConfig) RequestTimeout() time.Duration    { return c.requestTimeout }
func (c *grpcConfig) MaxConnectionsAge() time.Duration { return c.maxConnectionsAge }
func (c *grpcConfig) MaxConcurrentStreams() uint32     { return c.maxConcurrentStreams }
func (c *grpcConfig) MaxReceiveMessageSize() int       { return c.maxReceiveMessageSize }
func (c *grpcConfig) MaxSendMessageSize() int          { return c.maxSendMessageSize }
func (c *grpcConfig) KeepAliveTime() time.Duration     { return c.keepAliveTime }
func (c *grpcConfig) KeepAliveTimeout() time.Duration  { return c.keepAliveTimeout }
