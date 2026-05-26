package service

import (
	"context"
	"testing"
	"time"

	"github.com/shamhi/top-search/internal/core/domain"
)

func BenchmarkIngest(b *testing.B) {
	repo := newStubRepo()
	svc := NewTrendService(repo, testLogger(b))
	now := time.Now().Unix()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = svc.Ingest(context.Background(), domain.SearchQueryEvent{
			Query: "bench", CreatedAt: now,
		})
	}
}

func BenchmarkGetTop(b *testing.B) {
	repo := newStubRepo()
	svc := NewTrendService(repo, testLogger(b))
	now := time.Now().Unix()

	for range 5000 {
		_ = svc.Ingest(context.Background(), domain.SearchQueryEvent{
			Query: "bench-query", CreatedAt: now,
		})
	}

	_, _ = svc.GetTop(context.Background(), 20)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, _ = svc.GetTop(context.Background(), 20)
	}
}

func BenchmarkNormalizeQuery(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		_ = normalizeQuery("IPHONE 15 Pro Max!")
	}
}
