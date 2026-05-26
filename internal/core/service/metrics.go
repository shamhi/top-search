package service

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	EventsIngested = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trend_events_ingested_total",
			Help: "Total search events successfully ingested.",
		},
		[]string{},
	)

	DedupHits = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "trend_dedup_hits_total",
			Help: "Total events blocked by deduplication.",
		},
	)

	StopwordBlocks = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "trend_stopword_blocks_total",
			Help: "Total events blocked by stop-word filter.",
		},
	)

	EmptyQueryDropped = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "trend_empty_query_dropped_total",
			Help: "Total events dropped due to empty/normalized query.",
		},
	)

	CacheMisses = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "trend_cache_misses_total",
			Help: "Total GetTop calls that missed in-memory cache and triggered refresh.",
		},
	)

	CacheHits = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "trend_cache_hits_total",
			Help: "Total GetTop calls served from in-memory cache.",
		},
	)

	StopWordsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "trend_stopwords_active",
			Help: "Current number of active stop-words.",
		},
	)

	AggregatorTicks = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "trend_aggregator_ticks_total",
			Help: "Total aggregator refresh cycles executed.",
		},
	)

	CachedTopEntries = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "trend_cached_top_entries",
			Help: "Current number of entries in the cached top-N set.",
		},
	)
)
