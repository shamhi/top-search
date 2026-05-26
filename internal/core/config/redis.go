package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type RedisConfig interface {
	Host() string
	Port() int
	Addr() string
	Username() string
	Password() string
	DB() int

	ConnectTimeout() time.Duration
	DialTimeout() time.Duration
	ReadTimeout() time.Duration
	WriteTimeout() time.Duration

	PoolSize() int
	MinIdleConns() int
}

type redisConfig struct {
	host     string
	port     int
	username string
	password string
	db       int

	connectTimeout time.Duration
	dialTimeout    time.Duration
	readTimeout    time.Duration
	writeTimeout   time.Duration

	poolSize     int
	minIdleConns int
}

func NewRedisConfig() RedisConfig {
	return &redisConfig{
		host:           viper.GetString("redis.host"),
		port:           viper.GetInt("redis.port"),
		username:       viper.GetString("redis.username"),
		password:       viper.GetString("redis.password"),
		db:             viper.GetInt("redis.db"),
		connectTimeout: viper.GetDuration("redis.connect-timeout"),
		dialTimeout:    viper.GetDuration("redis.dial-timeout"),
		readTimeout:    viper.GetDuration("redis.read-timeout"),
		writeTimeout:   viper.GetDuration("redis.write-timeout"),
		poolSize:       viper.GetInt("redis.pool-size"),
		minIdleConns:   viper.GetInt("redis.min-idle-conns"),
	}
}

func (rc *redisConfig) Host() string                  { return rc.host }
func (rc *redisConfig) Port() int                     { return rc.port }
func (rc *redisConfig) Addr() string                  { return fmt.Sprintf("%s:%d", rc.host, rc.port) }
func (rc *redisConfig) Username() string              { return rc.username }
func (rc *redisConfig) Password() string              { return rc.password }
func (rc *redisConfig) DB() int                       { return rc.db }
func (rc *redisConfig) ConnectTimeout() time.Duration { return rc.connectTimeout }
func (rc *redisConfig) DialTimeout() time.Duration    { return rc.dialTimeout }
func (rc *redisConfig) ReadTimeout() time.Duration    { return rc.readTimeout }
func (rc *redisConfig) WriteTimeout() time.Duration   { return rc.writeTimeout }
func (rc *redisConfig) PoolSize() int                 { return rc.poolSize }
func (rc *redisConfig) MinIdleConns() int             { return rc.minIdleConns }
