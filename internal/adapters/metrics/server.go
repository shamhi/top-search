package metrics

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/shamhi/top-search/pkg/logger/types"
)

type Server struct {
	srv *http.Server
	log *types.Logger
}

func NewServer(addr string, log *types.Logger) *Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	return &Server{
		srv: &http.Server{
			Addr:              addr,
			Handler:           mux,
			ReadHeaderTimeout: 3 * time.Second,
		},
		log: log,
	}
}

func (s *Server) Run() error {
	s.log.Info("metrics server starting", zap.String("addr", s.srv.Addr))

	if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("metrics serve: %w", err)
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.log.Info("metrics server shutting down")

	if err := s.srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("metrics shutdown: %w", err)
	}

	return nil
}
