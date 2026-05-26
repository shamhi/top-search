package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

type Client struct {
	raw *goredis.Client
}

func New(ctx context.Context, cfg Config) (*Client, error) {
	rdb := goredis.NewClient(&goredis.Options{
		Addr:         cfg.Addr,
		Username:     cfg.Username,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return &Client{raw: rdb}, nil
}

func (c *Client) SAdd(ctx context.Context, key string, members ...any) *goredis.IntCmd {
	return c.raw.SAdd(ctx, key, members...)
}

func (c *Client) SRem(ctx context.Context, key string, members ...any) *goredis.IntCmd {
	return c.raw.SRem(ctx, key, members...)
}

func (c *Client) SMembers(ctx context.Context, key string) *goredis.StringSliceCmd {
	return c.raw.SMembers(ctx, key)
}

func (c *Client) ZIncrBy(ctx context.Context, key string, increment float64, member string) *goredis.FloatCmd {
	return c.raw.ZIncrBy(ctx, key, increment, member)
}

func (c *Client) ZUnionStore(ctx context.Context, dest string, store *goredis.ZStore) *goredis.IntCmd {
	return c.raw.ZUnionStore(ctx, dest, store)
}

func (c *Client) ZRevRangeWithScores(ctx context.Context, key string, start, stop int64) *goredis.ZSliceCmd {
	return c.raw.ZRevRangeWithScores(ctx, key, start, stop)
}

func (c *Client) Incr(ctx context.Context, key string) *goredis.IntCmd {
	return c.raw.Incr(ctx, key)
}

func (c *Client) Expire(ctx context.Context, key string, expiration time.Duration) *goredis.BoolCmd {
	return c.raw.Expire(ctx, key, expiration)
}

func (c *Client) Close() error {
	return c.raw.Close()
}
