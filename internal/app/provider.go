package app

import (
	"context"

	gonats "github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	grpcAdapter "github.com/shamhi/top-search/internal/adapters/grpc"
	metricsAdapter "github.com/shamhi/top-search/internal/adapters/metrics"
	"github.com/shamhi/top-search/internal/adapters/nats"
	redisAdapter "github.com/shamhi/top-search/internal/adapters/redis"
	"github.com/shamhi/top-search/internal/core/config"
	"github.com/shamhi/top-search/internal/core/ports"
	"github.com/shamhi/top-search/internal/core/service"
	"github.com/shamhi/top-search/pkg/logger/types"
	pkgnats "github.com/shamhi/top-search/pkg/nats"
	pkgredis "github.com/shamhi/top-search/pkg/redis"
)

type appProvider struct {
	// Configuration
	cfg *config.Config
	log *types.Logger

	// Infra
	rdb      *pkgredis.Client
	natsConn *pkgnats.Conn
	natsJS   *pkgnats.JetStream

	// Adapters
	gRPCSrv *grpcAdapter.Server
	metrics *metricsAdapter.Server

	// Ports
	trendRepo   ports.TrendRepository
	trendSvc    ports.TrendService
	trendBroker ports.TrendBroker
}

func newAppProvider(cfg *config.Config, log *types.Logger) *appProvider {
	return &appProvider{
		cfg: cfg,
		log: log,
	}
}

func (p *appProvider) Cfg() *config.Config {
	return p.cfg
}

func (p *appProvider) Redis() *pkgredis.Client {
	if p.rdb != nil {
		return p.rdb
	}

	ctx, cancel := context.WithTimeout(context.Background(), p.cfg.Redis.ConnectTimeout())
	defer cancel()

	rd, err := pkgredis.New(ctx, pkgredis.Config{
		Addr:         p.cfg.Redis.Addr(),
		Username:     p.cfg.Redis.Username(),
		Password:     p.cfg.Redis.Password(),
		DB:           p.cfg.Redis.DB(),
		DialTimeout:  p.cfg.Redis.DialTimeout(),
		ReadTimeout:  p.cfg.Redis.ReadTimeout(),
		WriteTimeout: p.cfg.Redis.WriteTimeout(),
		PoolSize:     p.cfg.Redis.PoolSize(),
		MinIdleConns: p.cfg.Redis.MinIdleConns(),
	})
	if err != nil {
		p.log.Fatal("redis connect failed", zap.Error(err))
	}

	p.rdb = rd
	p.log.Debug("redis connected", zap.String("addr", p.cfg.Redis.Addr()))

	return rd
}

func (p *appProvider) NatsConn() *pkgnats.Conn {
	if p.natsConn != nil {
		return p.natsConn
	}

	nc, err := pkgnats.New(pkgnats.Config{
		URL:               p.cfg.Nats.URL(),
		Name:              p.cfg.Nats.Name(),
		ConnectTimeout:    p.cfg.Nats.ConnectTimeout(),
		ReconnectWait:     p.cfg.Nats.ReconnectWait(),
		MaxReconnect:      p.cfg.Nats.MaxReconnect(),
		RetryOnFailedConn: p.cfg.Nats.RetryOnFailedConnect(),
		ReconnectHandler: func(_ *gonats.Conn) {
			p.log.Info("nats reconnected")
		},
		DisconnectErrHandler: func(_ *gonats.Conn, err error) {
			p.log.Warn("nats disconnected", zap.Error(err))
		},
		CloseHandler: func(_ *gonats.Conn) {
			p.log.Warn("nats connection closed")
		},
	})
	if err != nil {
		p.log.Fatal("nats connect failed", zap.Error(err))
	}

	p.natsConn = nc
	p.log.Info("nats connected", zap.String("url", p.cfg.Nats.URL()))

	return nc
}

func (p *appProvider) NatsJS() *pkgnats.JetStream {
	if p.natsJS != nil {
		return p.natsJS
	}

	js, err := pkgnats.NewJetStream(p.NatsConn())
	if err != nil {
		p.log.Fatal("jetstream create failed", zap.Error(err))
	}

	p.natsJS = js
	p.log.Info("jetstream ready")

	return js
}

func (p *appProvider) TrendRepository() ports.TrendRepository {
	if p.trendRepo != nil {
		return p.trendRepo
	}

	p.trendRepo = redisAdapter.NewTrendRepository(p.Redis())

	return p.trendRepo
}

func (p *appProvider) TrendService() ports.TrendService {
	if p.trendSvc != nil {
		return p.trendSvc
	}

	p.trendSvc = service.NewTrendService(
		p.TrendRepository(),
		p.log,
	)

	return p.trendSvc
}

func (p *appProvider) TrendBroker() ports.TrendBroker {
	if p.trendBroker != nil {
		return p.trendBroker
	}

	js := p.NatsJS()

	ctx, cancel := context.WithTimeout(context.Background(), p.cfg.Nats.ConnectTimeout())
	defer cancel()

	if err := js.EnsureStream(ctx, p.cfg.Nats.StreamName(), p.cfg.Nats.Subject()); err != nil {
		p.log.Panic("ensure stream", zap.Error(err))
	}

	if err := js.EnsureConsumer(
		ctx,
		p.cfg.Nats.StreamName(),
		p.cfg.Nats.ConsumerName(),
		p.cfg.Nats.Subject(),
	); err != nil {
		p.log.Panic("ensure consumer", zap.Error(err))
	}

	p.log.Info(
		"broker stream ready",
		zap.String("stream", p.cfg.Nats.StreamName()),
		zap.String("consumer", p.cfg.Nats.ConsumerName()),
		zap.String("subject", p.cfg.Nats.Subject()),
	)

	consumer := pkgnats.NewConsumer(p.NatsJS())

	p.trendBroker = nats.NewBroker(
		p.NatsConn(),
		consumer,
		p.cfg.Nats.StreamName(),
		p.cfg.Nats.ConsumerName(),
		p.log,
	)

	return p.trendBroker
}

func (p *appProvider) GrpcServer() *grpcAdapter.Server {
	if p.gRPCSrv != nil {
		return p.gRPCSrv
	}

	opts := []grpc.ServerOption{
		grpc.ConnectionTimeout(p.cfg.Grpc.ConnectionTimeout()),
		grpc.MaxConcurrentStreams(p.cfg.Grpc.MaxConcurrentStreams()),
		grpc.MaxRecvMsgSize(p.cfg.Grpc.MaxReceiveMessageSize()),
		grpc.MaxSendMsgSize(p.cfg.Grpc.MaxSendMessageSize()),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionAge: p.cfg.Grpc.MaxConnectionsAge(),
			Time:             p.cfg.Grpc.KeepAliveTime(),
			Timeout:          p.cfg.Grpc.KeepAliveTimeout(),
		}),
	}

	srv, err := grpcAdapter.NewServer(
		p.TrendService(),
		p.cfg.Grpc.Addr(),
		p.log,
		p.cfg.Grpc.RequestTimeout(),
		opts...,
	)
	if err != nil {
		p.log.Fatal("grpc server create failed", zap.Error(err))
	}

	p.gRPCSrv = srv

	return srv
}

func (p *appProvider) MetricsServer() *metricsAdapter.Server {
	if p.metrics != nil {
		return p.metrics
	}

	p.metrics = metricsAdapter.NewServer(
		p.cfg.Metrics.Addr(),
		p.log,
	)

	return p.metrics
}

func (p *appProvider) Close() {
	if p.natsConn != nil {
		p.natsConn.Close()
		p.log.Info("nats closed")
	}

	if p.rdb != nil {
		_ = p.rdb.Close()
		p.log.Info("redis closed")
	}
}
