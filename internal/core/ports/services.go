package ports

import (
	"context"

	"github.com/shamhi/top-search/internal/core/domain"
)

type TrendService interface {
	Start(ctx context.Context) error
	Stop()

	Ingest(ctx context.Context, event domain.SearchQueryEvent) error
	GetTop(ctx context.Context, limit int) ([]domain.TrendingQuery, error)
	StreamTop(ctx context.Context, limit int, intervalSeconds uint32) (<-chan []domain.TrendingQuery, error)
	AddStopWord(ctx context.Context, word string) error
	RemoveStopWord(ctx context.Context, word string) error
	ListStopWords(ctx context.Context) ([]string, error)
}
