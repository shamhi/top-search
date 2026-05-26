package grpc

import (
	"context"
	"fmt"
	"net"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	trendv1 "github.com/shamhi/top-search/api/gen/trend/v1"
	"github.com/shamhi/top-search/internal/adapters/grpc/interceptor"
	"github.com/shamhi/top-search/internal/core/ports"
	"github.com/shamhi/top-search/pkg/logger/types"
)

type Server struct {
	srv *grpc.Server
	lis net.Listener
	log *types.Logger
}

func NewServer(
	svc ports.TrendService,
	addr string,
	log *types.Logger,
	requestTimeout time.Duration,
	opts ...grpc.ServerOption,
) (*Server, error) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen %s: %w", addr, err)
	}

	unaryInterceptors := []grpc.UnaryServerInterceptor{
		interceptor.UnaryRecovery(log),
		interceptor.UnaryTimeout(requestTimeout),
		interceptor.UnaryMetrics(),
		interceptor.UnaryLogging(log),
	}
	streamInterceptors := []grpc.StreamServerInterceptor{
		interceptor.StreamRecovery(log),
		interceptor.StreamMetrics(),
		interceptor.StreamLogging(log),
	}

	srv := grpc.NewServer(
		append(
			opts,
			grpc.ChainUnaryInterceptor(unaryInterceptors...),
			grpc.ChainStreamInterceptor(streamInterceptors...),
		)...,
	)

	trendv1.RegisterTrendServiceServer(srv, &trendServer{svc: svc})
	reflection.Register(srv)

	hs := health.NewServer()
	hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(srv, hs)

	return &Server{srv: srv, lis: lis, log: log}, nil
}

func (s *Server) Run() error {
	s.log.Info("gRPC server starting", zap.String("addr", s.lis.Addr().String()))

	return s.srv.Serve(s.lis)
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.log.Info("gRPC server shutting down")

	done := make(chan struct{})

	go func() {
		s.srv.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		s.log.Info("gRPC server stopped gracefully")

		return nil
	case <-ctx.Done():
		s.log.Warn("gRPC graceful stop timed out, forcing")
		s.srv.Stop()

		return ctx.Err()
	}
}

type trendServer struct {
	trendv1.UnimplementedTrendServiceServer
	svc ports.TrendService
}

func (s *trendServer) GetTop(ctx context.Context, req *trendv1.GetTopRequest) (*trendv1.GetTopResponse, error) {
	entries, err := s.svc.GetTop(ctx, int(req.Limit))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get top: %v", err)
	}

	resp := &trendv1.GetTopResponse{
		Entries: make([]*trendv1.TrendingQuery, len(entries)),
	}

	for i, e := range entries {
		resp.Entries[i] = &trendv1.TrendingQuery{
			Query: e.Query,
			Score: e.Score,
		}
	}

	return resp, nil
}

func (s *trendServer) StreamTop(req *trendv1.StreamTopRequest, stream trendv1.TrendService_StreamTopServer) error {
	ch, err := s.svc.StreamTop(stream.Context(), int(req.Limit), req.UpdateIntervalSeconds)
	if err != nil {
		return status.Errorf(codes.Internal, "stream top: %v", err)
	}

	for batch := range ch {
		resp := &trendv1.GetTopResponse{
			Entries: make([]*trendv1.TrendingQuery, len(batch)),
		}

		for i, e := range batch {
			resp.Entries[i] = &trendv1.TrendingQuery{
				Query: e.Query,
				Score: e.Score,
			}
		}

		if err := stream.Send(resp); err != nil {
			return status.Errorf(codes.Internal, "stream send: %v", err)
		}
	}

	return nil
}

func (s *trendServer) AddStopWord(ctx context.Context, req *trendv1.AddStopWordRequest) (*trendv1.AddStopWordResponse, error) {
	if req.Word == "" {
		return nil, status.Error(codes.InvalidArgument, "word is empty")
	}

	if err := s.svc.AddStopWord(ctx, req.Word); err != nil {
		return nil, status.Errorf(codes.Internal, "add stop word: %v", err)
	}

	return &trendv1.AddStopWordResponse{}, nil
}

func (s *trendServer) RemoveStopWord(ctx context.Context, req *trendv1.RemoveStopWordRequest) (*trendv1.RemoveStopWordResponse, error) {
	if req.Word == "" {
		return nil, status.Error(codes.InvalidArgument, "word is empty")
	}

	if err := s.svc.RemoveStopWord(ctx, req.Word); err != nil {
		return nil, status.Errorf(codes.Internal, "remove stop word: %v", err)
	}

	return &trendv1.RemoveStopWordResponse{}, nil
}

func (s *trendServer) ListStopWords(ctx context.Context, req *trendv1.ListStopWordsRequest) (*trendv1.ListStopWordsResponse, error) {
	words, err := s.svc.ListStopWords(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list stop words: %v", err)
	}

	return &trendv1.ListStopWordsResponse{Words: words}, nil
}
