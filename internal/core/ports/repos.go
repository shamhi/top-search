package ports

import (
	"context"

	"github.com/shamhi/top-search/internal/core/domain"
)

type TrendRepository interface {
	IncrementBucket(ctx context.Context, minute, query string) error
	AggregateTop(ctx context.Context, topKey string, minuteKeys []string) error
	GetTop(ctx context.Context, topKey string, limit int) ([]domain.TrendingQuery, error)

	CheckDedup(ctx context.Context, userID, query string) (bool, error)

	AddStopWord(ctx context.Context, word string) error
	RemoveStopWord(ctx context.Context, word string) error
	ListStopWords(ctx context.Context) ([]string, error)
}
