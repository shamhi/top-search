package app

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/shamhi/top-search/internal/core/config"
	"github.com/shamhi/top-search/pkg/logger"
	"github.com/shamhi/top-search/pkg/logger/types"
)

type App struct {
	provider *appProvider
	log      *types.Logger
}

func New(ctx context.Context) (*App, error) {
	a := &App{}

	if err := a.init(ctx); err != nil {
		return nil, fmt.Errorf("init app: %w", err)
	}

	return a, nil
}

func (a *App) init(ctx context.Context) error {
	cfg, err := config.NewConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if err := logger.Init(logger.Config{
		Debug:        cfg.Logger.Debug(),
		LogToFile:    cfg.Logger.LogToFile(),
		LogsDir:      cfg.Logger.LogsDir(),
		TimeLocation: cfg.Logger.Timezone(),
	}); err != nil {
		return fmt.Errorf("init logger: %w", err)
	}

	a.log = logger.Log
	a.log.Debug("logger initialized", zap.Bool("debug", cfg.Logger.Debug()))
	a.provider = newAppProvider(cfg, logger.Log)

	return nil
}

func (a *App) Run() {
	defer a.gracefulShutdown()
	defer func() {
		if r := recover(); r != nil {
			a.log.Error("panic in run", zap.Any("panic", r))
		}
	}()

	sigCtx, sigCancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer sigCancel()

	if err := a.startComponents(context.Background()); err != nil {
		a.log.Error("start components failed", zap.Error(err))

		return
	}

	a.log.Info(
		"app started",
		zap.Int("grpc_port", a.provider.Cfg().Grpc.Port()),
	)

	<-sigCtx.Done()
	a.log.Info("shutdown signal received")
}

func (a *App) startComponents(ctx context.Context) error {
	if err := a.provider.TrendService().Start(ctx); err != nil {
		return fmt.Errorf("start trend service: %w", err)
	}

	if err := a.provider.SetupBroker(ctx); err != nil {
		return fmt.Errorf("setup broker: %w", err)
	}

	if err := a.provider.TrendBroker().Subscribe(ctx, a.provider.TrendService().Ingest); err != nil {
		return fmt.Errorf("subscribe broker: %w", err)
	}

	grpcErrCh := a.startGrpc()

	var metricsErrCh chan error
	if a.provider.Cfg().Metrics.Enabled() {
		metricsErrCh = a.startMetrics()
	}

	go func() {
		select {
		case err := <-grpcErrCh:
			if err != nil {
				panic("grpc server error: " + err.Error())
			}
		case err := <-metricsErrCh:
			if err != nil {
				panic("metrics server error: " + err.Error())
			}
		}
	}()

	return nil
}

func (a *App) startGrpc() chan error {
	errCh := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				errCh <- fmt.Errorf("panic in grpc: %v", r)
			}
		}()

		grpcSrv := a.provider.GrpcServer()
		errCh <- grpcSrv.Run()
	}()

	return errCh
}

func (a *App) startMetrics() chan error {
	errCh := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				errCh <- fmt.Errorf("panic in metrics: %v", r)
			}
		}()

		metricsSrv := a.provider.MetricsServer()
		errCh <- metricsSrv.Run()
	}()

	return errCh
}

func (a *App) gracefulShutdown() {
	a.log.Info("gracefully shutting down app...")

	if a.provider == nil {
		return
	}

	ctx := context.Background()

	if brk := a.provider.TrendBroker(); brk != nil {
		if err := brk.Stop(); err != nil {
			a.log.Error("stop broker", zap.Error(err))
		} else {
			a.log.Info("broker stopped")
		}
	}

	if svc := a.provider.TrendService(); svc != nil {
		svc.Stop()
		a.log.Info("trend service stopped")
	}

	if srv := a.provider.GrpcServer(); srv != nil {
		grpcCtx, grpcCancel := context.WithTimeout(ctx, a.provider.Cfg().Grpc.ShutdownTimeout())
		defer grpcCancel()

		if err := srv.Shutdown(grpcCtx); err != nil {
			a.log.Error("shutdown grpc", zap.Error(err))
		} else {
			a.log.Info("grpc server stopped")
		}
	}

	if a.provider.Cfg().Metrics.Enabled() {
		if srv := a.provider.MetricsServer(); srv != nil {
			metricsCtx, metricsCancel := context.WithTimeout(ctx, a.provider.Cfg().Metrics.ShutdownTimeout())
			defer metricsCancel()

			if err := srv.Shutdown(metricsCtx); err != nil {
				a.log.Error("shutdown metrics", zap.Error(err))
			} else {
				a.log.Info("metrics server stopped")
			}
		}
	}

	a.provider.Close()

	a.log.Info("gracefully stopping completed")
}
