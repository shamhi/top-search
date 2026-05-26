package ports

import (
	"context"

	"github.com/shamhi/top-search/internal/core/domain"
)

type SearchEventHandler func(ctx context.Context, event domain.SearchQueryEvent) error

type TrendBroker interface {
	Subscribe(ctx context.Context, handler SearchEventHandler) error
	Stop() error
}
