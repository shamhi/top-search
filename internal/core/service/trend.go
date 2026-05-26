package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"go.uber.org/zap"

	"github.com/shamhi/top-search/internal/core/domain"
	"github.com/shamhi/top-search/internal/core/ports"
	"github.com/shamhi/top-search/pkg/logger/types"
)

const (
	windowMinutes   = 5
	bucketFormat    = "200601021504"
	aggregatePeriod = 1 * time.Second
	maxTopSize      = 100
)

type TrendService struct {
	repo   ports.TrendRepository
	log    *types.Logger
	topKey string

	stopWords   map[string]struct{}
	stopWordsMu sync.RWMutex

	cachedTop   []domain.TrendingQuery
	cachedTopMu sync.RWMutex

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewTrendService(repo ports.TrendRepository, log *types.Logger) *TrendService {
	return &TrendService{
		repo:      repo,
		log:       log,
		topKey:    "trend:top",
		stopWords: make(map[string]struct{}),
		cachedTop: []domain.TrendingQuery{},
	}
}

func (s *TrendService) Start(ctx context.Context) error {
	if err := s.LoadStopWords(ctx); err != nil {
		s.log.Warn("stop words load failed", zap.Error(err))
	}

	ctx, s.cancel = context.WithCancel(ctx)

	s.wg.Add(1)
	go s.aggregateLoop(ctx)

	s.log.Info("aggregator started", zap.Duration("interval", aggregatePeriod))

	return nil
}

func (s *TrendService) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
	s.log.Info("aggregator stopped")
}

func (s *TrendService) Ingest(ctx context.Context, event domain.SearchQueryEvent) error {
	query := normalizeQuery(event.Query)
	if query == "" {
		EmptyQueryDropped.Inc()

		return nil
	}

	s.stopWordsMu.RLock()
	blocked := s.matchesStopWord(query)
	s.stopWordsMu.RUnlock()
	if blocked {
		StopwordBlocks.Inc()

		s.log.Debug(
			"event blocked by stopword",
			zap.String("query", query),
		)

		return nil
	}

	dedupID := event.UserID
	if dedupID == "" {
		dedupID = event.SessionID
	}
	if dedupID == "" {
		dedupID = event.DeviceID
	}

	if dedupID != "" {
		dup, err := s.repo.CheckDedup(ctx, dedupID, query)
		if err != nil {
			s.log.Warn("dedup check failed", zap.Error(err))
		} else if dup {
			DedupHits.Inc()

			s.log.Debug(
				"event deduplicated",
				zap.String("query", query),
				zap.String("dedup_id", dedupID),
			)

			return nil
		}
	}

	minute := time.Unix(event.CreatedAt, 0).UTC().Format(bucketFormat)

	if err := s.repo.IncrementBucket(ctx, minute, query); err != nil {
		return fmt.Errorf("increment bucket: %w", err)
	}

	EventsIngested.WithLabelValues().Inc()

	s.log.Debug(
		"event ingested",
		zap.String("query", query),
		zap.String("bucket", minute),
	)

	return nil
}

func (s *TrendService) GetTop(ctx context.Context, limit int) ([]domain.TrendingQuery, error) {
	s.cachedTopMu.RLock()
	cached := s.cachedTop
	s.cachedTopMu.RUnlock()

	if limit <= 0 {
		limit = 10
	}

	if len(cached) == 0 {
		CacheMisses.Inc()

		s.log.Debug("cache miss, refreshing top")

		var err error
		cached, err = s.refreshTop(ctx)
		if err != nil {
			return nil, err
		}
	} else {
		CacheHits.Inc()
	}

	s.stopWordsMu.RLock()
	filtered := make([]domain.TrendingQuery, 0, limit)
	for _, e := range cached {
		if s.matchesStopWord(e.Query) {
			continue
		}
		filtered = append(filtered, e)
		if len(filtered) >= limit {
			break
		}
	}
	s.stopWordsMu.RUnlock()

	s.log.Debug(
		"top cached",
		zap.Int("cached", len(cached)),
		zap.Int("filtered", len(filtered)),
		zap.Int("limit", limit),
	)

	return filtered, nil
}

func (s *TrendService) StreamTop(ctx context.Context, limit int, intervalSeconds uint32) (<-chan []domain.TrendingQuery, error) {
	if limit <= 0 {
		limit = 10
	}
	if intervalSeconds == 0 {
		intervalSeconds = 1
	}

	ch := make(chan []domain.TrendingQuery, 4)
	interval := time.Duration(intervalSeconds) * time.Second

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer close(ch)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		sendTop := func() bool {
			entries, err := s.GetTop(ctx, limit)
			if err != nil {
				s.log.Warn("stream top failed", zap.Error(err))

				return true
			}

			select {
			case <-ctx.Done():
				return false
			case ch <- entries:
				return true
			}
		}

		if !sendTop() {
			return
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if !sendTop() {
					return
				}
			}
		}
	}()

	return ch, nil
}

func (s *TrendService) AddStopWord(ctx context.Context, word string) error {
	word = normalizeQuery(word)
	if word == "" {
		return nil
	}

	if err := s.repo.AddStopWord(ctx, word); err != nil {
		return err
	}

	s.stopWordsMu.Lock()
	s.stopWords[word] = struct{}{}
	StopWordsActive.Set(float64(len(s.stopWords)))
	s.stopWordsMu.Unlock()

	s.log.Debug("stop word added", zap.String("word", word))

	return nil
}

func (s *TrendService) RemoveStopWord(ctx context.Context, word string) error {
	word = normalizeQuery(word)
	if word == "" {
		return nil
	}

	if err := s.repo.RemoveStopWord(ctx, word); err != nil {
		return err
	}

	s.stopWordsMu.Lock()
	delete(s.stopWords, word)
	StopWordsActive.Set(float64(len(s.stopWords)))
	s.stopWordsMu.Unlock()

	s.log.Debug("stop word removed", zap.String("word", word))

	return nil
}

func (s *TrendService) ListStopWords(ctx context.Context) ([]string, error) {
	return s.repo.ListStopWords(ctx)
}

func (s *TrendService) LoadStopWords(ctx context.Context) error {
	words, err := s.repo.ListStopWords(ctx)
	if err != nil {
		return fmt.Errorf("list stop words: %w", err)
	}

	s.stopWordsMu.Lock()
	s.stopWords = make(map[string]struct{}, len(words))
	for _, w := range words {
		w = normalizeQuery(w)
		if w == "" {
			continue
		}
		s.stopWords[w] = struct{}{}
	}
	StopWordsActive.Set(float64(len(s.stopWords)))
	s.stopWordsMu.Unlock()

	s.log.Info("stop words loaded", zap.Int("count", len(words)))

	return nil
}

func (s *TrendService) aggregateLoop(ctx context.Context) {
	ticker := time.NewTicker(aggregatePeriod)
	defer ticker.Stop()
	defer s.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			AggregatorTicks.Inc()

			entries, err := s.refreshTop(ctx)
			if err != nil {
				s.log.Warn("refresh top failed", zap.Error(err))

				continue
			}

			s.cachedTopMu.Lock()
			s.cachedTop = entries
			s.cachedTopMu.Unlock()

			CachedTopEntries.Set(float64(len(entries)))

			s.log.Debug(
				"aggregator tick",
				zap.Int("top_entries", len(entries)),
			)
		}
	}
}

func (s *TrendService) refreshTop(ctx context.Context) ([]domain.TrendingQuery, error) {
	now := time.Now().UTC()

	var keys []string
	for i := 0; i < windowMinutes; i++ {
		t := now.Add(-time.Duration(i) * time.Minute)
		keys = append(keys, t.Format(bucketFormat))
	}
	sort.Strings(keys)

	if err := s.repo.AggregateTop(ctx, s.topKey, keys); err != nil {
		return nil, fmt.Errorf("aggregate: %w", err)
	}

	entries, err := s.repo.GetTop(ctx, s.topKey, maxTopSize)
	if err != nil {
		return nil, fmt.Errorf("get top: %w", err)
	}

	return entries, nil
}

func (s *TrendService) matchesStopWord(query string) bool {
	if _, ok := s.stopWords[query]; ok {
		return true
	}

	for _, token := range strings.Fields(query) {
		if _, ok := s.stopWords[token]; ok {
			return true
		}
	}

	return false
}

func normalizeQuery(q string) string {
	q = strings.TrimSpace(strings.ToLower(q))
	if q == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(q))

	for _, r := range q {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			b.WriteRune(r)
		}
	}

	result := strings.TrimSpace(b.String())

	return result
}
