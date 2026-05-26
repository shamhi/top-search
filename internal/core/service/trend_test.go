package service

import (
	"context"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/shamhi/top-search/internal/core/domain"
	"github.com/shamhi/top-search/pkg/logger/types"
)

func testLogger(t zaptest.TestingT) *types.Logger {
	return &types.Logger{Logger: zaptest.NewLogger(t)}
}

type stubRepo struct {
	mu     sync.Mutex
	top    []domain.TrendingQuery
	bucket map[string]map[string]int64
	dedup  map[string]int64
	sw     map[string]struct{}
}

func newStubRepo() *stubRepo {
	return &stubRepo{
		bucket: make(map[string]map[string]int64),
		dedup:  make(map[string]int64),
		sw:     make(map[string]struct{}),
	}
}

func (r *stubRepo) IncrementBucket(ctx context.Context, minute, query string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.bucket[minute] == nil {
		r.bucket[minute] = make(map[string]int64)
	}
	r.bucket[minute][query]++

	return nil
}

func (r *stubRepo) AggregateTop(ctx context.Context, topKey string, minuteKeys []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	merged := make(map[string]int64)
	for _, k := range minuteKeys {
		for q, c := range r.bucket[k] {
			merged[q] += c
		}
	}

	r.top = r.top[:0]
	for q, c := range merged {
		r.top = append(r.top, domain.TrendingQuery{Query: q, Score: uint64(c)})
	}
	sort.Slice(r.top, func(i, j int) bool {
		return r.top[i].Score > r.top[j].Score
	})

	return nil
}

func (r *stubRepo) GetTop(ctx context.Context, topKey string, limit int) ([]domain.TrendingQuery, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if limit <= 0 || limit > len(r.top) {
		limit = len(r.top)
	}

	return r.top[:limit], nil
}

func (r *stubRepo) CheckDedup(ctx context.Context, userID, query string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := userID + ":" + query
	r.dedup[key]++

	return r.dedup[key] > 1, nil
}

func (r *stubRepo) AddStopWord(ctx context.Context, word string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sw[word] = struct{}{}

	return nil
}

func (r *stubRepo) RemoveStopWord(ctx context.Context, word string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sw, word)

	return nil
}

func (r *stubRepo) ListStopWords(ctx context.Context) ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var words []string
	for w := range r.sw {
		words = append(words, w)
	}

	return words, nil
}

func TestNormalizeQuery(t *testing.T) {
	is := assert.New(t)

	is.Equal("iphone", normalizeQuery("IPHONE!!!"))
	is.Equal("airpods pro", normalizeQuery("  AirPods Pro!  "))
	is.Equal("", normalizeQuery("!@#$%"))
	is.Equal("macbook", normalizeQuery("MacBook"))
}

func TestIngestIncrementsBucket(t *testing.T) {
	repo := newStubRepo()
	svc := NewTrendService(repo, testLogger(t))

	now := time.Now().Unix()

	err := svc.Ingest(context.Background(), domain.SearchQueryEvent{
		Query:     "iphone",
		CreatedAt: now,
	})
	require.NoError(t, err)

	minute := time.Unix(now, 0).UTC().Format(bucketFormat)
	assert.Equal(t, int64(1), repo.bucket[minute]["iphone"])
}

func TestIngestBlocksDedup(t *testing.T) {
	repo := newStubRepo()
	svc := NewTrendService(repo, testLogger(t))

	now := time.Now().Unix()

	_ = svc.Ingest(context.Background(), domain.SearchQueryEvent{
		Query: "spam", UserID: "bot-1", CreatedAt: now,
	})

	err := svc.Ingest(context.Background(), domain.SearchQueryEvent{
		Query: "spam", UserID: "bot-1", CreatedAt: now,
	})
	require.NoError(t, err)

	minute := time.Unix(now, 0).UTC().Format(bucketFormat)
	assert.Equal(t, int64(1), repo.bucket[minute]["spam"])
}

func TestIngestDedupUsesSessionWhenUserMissing(t *testing.T) {
	repo := newStubRepo()
	svc := NewTrendService(repo, testLogger(t))
	now := time.Now().Unix()

	for range 2 {
		_ = svc.Ingest(context.Background(), domain.SearchQueryEvent{
			Query:     "spam",
			SessionID: "session-1",
			CreatedAt: now,
		})
	}

	minute := time.Unix(now, 0).UTC().Format(bucketFormat)
	assert.Equal(t, int64(1), repo.bucket[minute]["spam"])
}

func TestIngestFiltersStopWords(t *testing.T) {
	repo := newStubRepo()
	svc := NewTrendService(repo, testLogger(t))

	_ = svc.AddStopWord(context.Background(), "BadWord!")
	now := time.Now().Unix()

	err := svc.Ingest(context.Background(), domain.SearchQueryEvent{
		Query: "badword", CreatedAt: now,
	})
	require.NoError(t, err)

	minute := time.Unix(now, 0).UTC().Format(bucketFormat)
	assert.Nil(t, repo.bucket[minute])
}

func TestGetTopReturnsAggregated(t *testing.T) {
	repo := newStubRepo()
	svc := NewTrendService(repo, testLogger(t))

	now := time.Now().Unix()

	for range 100 {
		_ = svc.Ingest(context.Background(), domain.SearchQueryEvent{
			Query: "iphone", CreatedAt: now,
		})
	}

	for range 50 {
		_ = svc.Ingest(context.Background(), domain.SearchQueryEvent{
			Query: "samsung", CreatedAt: now,
		})
	}

	entries, err := svc.GetTop(context.Background(), 5)
	require.NoError(t, err)
	require.Len(t, entries, 2)

	assert.Equal(t, "iphone", entries[0].Query)
	assert.Equal(t, uint64(100), entries[0].Score)
	assert.Equal(t, "samsung", entries[1].Query)
	assert.Equal(t, uint64(50), entries[1].Score)
}

func TestGetTopFiltersStopWords(t *testing.T) {
	repo := newStubRepo()
	svc := NewTrendService(repo, testLogger(t))

	now := time.Now().Unix()

	_ = svc.Ingest(context.Background(), domain.SearchQueryEvent{
		Query: "badword", CreatedAt: now,
	})
	_ = svc.Ingest(context.Background(), domain.SearchQueryEvent{
		Query: "goodword", CreatedAt: now,
	})

	_ = svc.AddStopWord(context.Background(), "badword")

	entries, err := svc.GetTop(context.Background(), 5)
	require.NoError(t, err)

	for _, e := range entries {
		assert.NotEqual(t, "badword", e.Query)
	}

	found := false
	for _, e := range entries {
		if e.Query == "goodword" {
			found = true
		}
	}
	assert.True(t, found)
}

func TestGetTopRespectsLimit(t *testing.T) {
	repo := newStubRepo()
	svc := NewTrendService(repo, testLogger(t))

	now := time.Now().Unix()

	for i, q := range []string{"a", "b", "c", "d", "e", "f", "g"} {
		for range 10 - i {
			_ = svc.Ingest(context.Background(), domain.SearchQueryEvent{
				Query: q, CreatedAt: now,
			})
		}
	}

	entries, err := svc.GetTop(context.Background(), 3)
	require.NoError(t, err)
	assert.Len(t, entries, 3)
}

func TestEmptyQueryIgnored(t *testing.T) {
	repo := newStubRepo()
	svc := NewTrendService(repo, testLogger(t))

	err := svc.Ingest(context.Background(), domain.SearchQueryEvent{
		Query: "", CreatedAt: time.Now().Unix(),
	})
	require.NoError(t, err)

	assert.Empty(t, repo.bucket)
}
