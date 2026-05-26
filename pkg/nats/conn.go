package nats

import (
	"fmt"
	"time"

	gonats "github.com/nats-io/nats.go"
)

type Config struct {
	URL                  string
	Name                 string
	ConnectTimeout       time.Duration
	ReconnectWait        time.Duration
	MaxReconnect         int
	RetryOnFailedConn    bool
	ReconnectHandler     gonats.ConnHandler
	DisconnectErrHandler gonats.ConnErrHandler
	CloseHandler         gonats.ConnHandler
}

type Conn struct {
	raw *gonats.Conn
}

func New(cfg Config) (*Conn, error) {
	nc, err := gonats.Connect(
		cfg.URL,
		gonats.Name(cfg.Name),
		gonats.Timeout(cfg.ConnectTimeout),
		gonats.RetryOnFailedConnect(cfg.RetryOnFailedConn),
		gonats.MaxReconnects(cfg.MaxReconnect),
		gonats.ReconnectWait(cfg.ReconnectWait),
		gonats.ReconnectHandler(cfg.ReconnectHandler),
		gonats.DisconnectErrHandler(cfg.DisconnectErrHandler),
		gonats.ClosedHandler(cfg.CloseHandler),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to nats: %w", err)
	}

	return &Conn{raw: nc}, nil
}

func (c *Conn) Drain() error {
	return c.raw.Drain()
}

func (c *Conn) Close() {
	c.raw.Close()
}
