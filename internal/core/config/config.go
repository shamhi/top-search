package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Logger  LoggerConfig
	Grpc    GrpcConfig
	Nats    NatsConfig
	Redis   RedisConfig
	Metrics MetricsConfig
}

func NewConfig() (*Config, error) {
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()
	setDefaults()

	viper.SetConfigType("yaml")
	viper.SetConfigName("config")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/config")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := errors.AsType[viper.ConfigFileNotFoundError](err); !ok {
			return nil, fmt.Errorf("load config: %w", err)
		}

		viper.SetConfigName("config.local")
		if err := viper.ReadInConfig(); err != nil {
			if _, ok := errors.AsType[viper.ConfigFileNotFoundError](err); !ok {
				return nil, fmt.Errorf("load config: %w", err)
			}
		}
	}

	return &Config{
		Logger:  NewLoggerConfig(),
		Grpc:    NewGrpcConfig(),
		Nats:    NewNatsConfig(),
		Redis:   NewRedisConfig(),
		Metrics: NewMetricsConfig(),
	}, nil
}

func setDefaults() {
	viper.SetDefault("logger.debug", false)
	viper.SetDefault("logger.log-to-file", false)
	viper.SetDefault("logger.logs-dir", "logs/")
	viper.SetDefault("logger.timezone", "Europe/Moscow")

	viper.SetDefault("grpc.port", 50051)
	viper.SetDefault("grpc.connection-timeout", "5s")
	viper.SetDefault("grpc.shutdown-timeout", "15s")
	viper.SetDefault("grpc.request-timeout", "3s")
	viper.SetDefault("grpc.max-connections-age", "5m")
	viper.SetDefault("grpc.max-concurrent-streams", 256)
	viper.SetDefault("grpc.max-receive-message-size", 4<<20)
	viper.SetDefault("grpc.max-send-message-size", 4<<20)
	viper.SetDefault("grpc.keepalive-time", "30s")
	viper.SetDefault("grpc.keepalive-timeout", "10s")

	viper.SetDefault("metrics.enabled", true)
	viper.SetDefault("metrics.port", 2112)
	viper.SetDefault("metrics.shutdown-timeout", "5s")

	viper.SetDefault("nats.url", "nats://localhost:4222")
	viper.SetDefault("nats.name", "top-search")
	viper.SetDefault("nats.connect-timeout", "10s")
	viper.SetDefault("nats.reconnect-wait", "2s")
	viper.SetDefault("nats.max-reconnect", 60)
	viper.SetDefault("nats.retry-on-failed-connect", true)
	viper.SetDefault("nats.stream-name", "SEARCH_QUERIES")
	viper.SetDefault("nats.consumer-name", "top-search-consumer")
	viper.SetDefault("nats.subject", "search.query.created")

	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.username", "")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("redis.connect-timeout", "5s")
	viper.SetDefault("redis.dial-timeout", "3s")
	viper.SetDefault("redis.read-timeout", "2s")
	viper.SetDefault("redis.write-timeout", "2s")
	viper.SetDefault("redis.pool-size", 32)
	viper.SetDefault("redis.min-idle-conns", 8)
}
