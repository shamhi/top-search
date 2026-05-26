package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/shamhi/top-search/internal/core/domain"
	pkredis "github.com/shamhi/top-search/pkg/redis"
)

const (
	bucketTTL   = 10 * time.Minute
	dedupWindow = 30 * time.Second
)

type TrendRepository struct {
	rdb *pkredis.Client
}

func NewTrendRepository(rdb *pkredis.Client) *TrendRepository {
	return &TrendRepository{rdb: rdb}
}

func (r *TrendRepository) IncrementBucket(ctx context.Context, minute, query string) error {
	key := bucketKeyStr(minute)

	if err := r.rdb.ZIncrBy(ctx, key, 1, query).Err(); err != nil {
		return fmt.Errorf("zincrby %s: %w", key, err)
	}

	if err := r.rdb.Expire(ctx, key, bucketTTL).Err(); err != nil {
		return fmt.Errorf("expire %s: %w", key, err)
	}

	return nil
}

func (r *TrendRepository) AggregateTop(ctx context.Context, topKey string, minuteKeys []string) error {
	if len(minuteKeys) == 0 {
		return nil
	}

	keys := make([]string, 0, len(minuteKeys))
	for _, minute := range minuteKeys {
		keys = append(keys, bucketKeyStr(minute))
	}

	store := &goredis.ZStore{
		Keys:      keys,
		Aggregate: "SUM",
	}

	if err := r.rdb.ZUnionStore(ctx, topKey, store).Err(); err != nil {
		return fmt.Errorf("zunionstore: %w", err)
	}

	if err := r.rdb.Expire(ctx, topKey, 2*time.Minute).Err(); err != nil {
		return fmt.Errorf("expire %s: %w", topKey, err)
	}

	return nil
}

func (r *TrendRepository) GetTop(ctx context.Context, topKey string, limit int) ([]domain.TrendingQuery, error) {
	result, err := r.rdb.ZRevRangeWithScores(ctx, topKey, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("zrevrange: %w", err)
	}

	entries := make([]domain.TrendingQuery, 0, len(result))

	for _, z := range result {
		query, ok := z.Member.(string)
		if !ok {
			query = fmt.Sprint(z.Member)
		}

		var score uint64
		if z.Score > 0 {
			score = uint64(z.Score)
		}

		entries = append(entries, domain.TrendingQuery{
			Query: query,
			Score: score,
		})
	}

	return entries, nil
}

func (r *TrendRepository) CheckDedup(ctx context.Context, userID, query string) (bool, error) {
	key := dedupKey(userID, query)

	n, err := r.rdb.Incr(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("incr dedup: %w", err)
	}

	if err := r.rdb.Expire(ctx, key, dedupWindow).Err(); err != nil {
		return false, fmt.Errorf("expire dedup: %w", err)
	}

	return n > 1, nil
}

func (r *TrendRepository) AddStopWord(ctx context.Context, word string) error {
	return r.rdb.SAdd(ctx, stopWordsKey, word).Err()
}

func (r *TrendRepository) RemoveStopWord(ctx context.Context, word string) error {
	return r.rdb.SRem(ctx, stopWordsKey, word).Err()
}

func (r *TrendRepository) ListStopWords(ctx context.Context) ([]string, error) {
	return r.rdb.SMembers(ctx, stopWordsKey).Result()
}

func bucketKeyStr(minute string) string {
	return fmt.Sprintf("trend:bucket:%s", minute)
}
