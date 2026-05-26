package nats

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	trendv1 "github.com/shamhi/top-search/api/gen/trend/v1"
	"github.com/shamhi/top-search/internal/core/domain"
	"github.com/shamhi/top-search/internal/core/ports"
	"github.com/shamhi/top-search/pkg/logger/types"
	pkgnats "github.com/shamhi/top-search/pkg/nats"
)

type Broker struct {
	conn         *pkgnats.Conn
	consumer     *pkgnats.Consumer
	streamName   string
	consumerName string
	log          *types.Logger

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewBroker(
	conn *pkgnats.Conn,
	consumer *pkgnats.Consumer,
	streamName string,
	consumerName string,
	log *types.Logger,
) *Broker {
	return &Broker{
		conn:         conn,
		consumer:     consumer,
		streamName:   streamName,
		consumerName: consumerName,
		log:          log,
	}
}

func (b *Broker) Subscribe(ctx context.Context, handler ports.SearchEventHandler) error {
	ctx, b.cancel = context.WithCancel(ctx)

	b.log.Info(
		"starting broker subscription",
		zap.String("stream", b.streamName),
		zap.String("consumer", b.consumerName),
	)

	b.wg.Add(1)

	go func() {
		defer b.wg.Done()

		defer b.log.Info("broker subscription stopped")

		if err := b.consumer.Consume(
			ctx, b.streamName, b.consumerName,
			func(ctx context.Context, payload []byte) error {
				var event trendv1.SearchEvent
				if err := proto.Unmarshal(payload, &event); err != nil {
					b.log.Warn(
						"failed to unmarshal search event",
						zap.Error(err),
					)

					return nil
				}

				createdAt := time.Now().Unix()
				if event.CreatedAt != nil {
					createdAt = event.CreatedAt.GetSeconds()
				}

				domainEvent := domain.SearchQueryEvent{
					EventID:   event.EventId,
					Query:     event.Query,
					UserID:    event.UserId,
					SessionID: event.SessionId,
					DeviceID:  event.DeviceId,
					Locale:    event.Locale,
					Platform:  event.Platform,
					CreatedAt: createdAt,
				}

				if err := handler(ctx, domainEvent); err != nil {
					b.log.Warn(
						"handler failed for search event",
						zap.String("query", event.Query),
						zap.Error(err),
					)

					return err
				}

				b.log.Debug(
					"search event processed",
					zap.String("query", event.Query),
				)

				return nil
			},
		); err != nil {
			b.log.Error("consumer failed", zap.Error(err))
		}
	}()

	return nil
}

func (b *Broker) Stop() error {
	b.log.Info("stopping broker")

	if b.cancel != nil {
		b.cancel()
	}

	b.wg.Wait()

	if err := b.conn.Drain(); err != nil {
		return fmt.Errorf("drain nats: %w", err)
	}

	return nil
}
