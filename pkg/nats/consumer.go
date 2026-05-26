package nats

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"
)

type MessageHandler func(ctx context.Context, payload []byte) error

type Consumer struct {
	js *JetStream
}

func NewConsumer(js *JetStream) *Consumer {
	return &Consumer{js: js}
}

func (c *Consumer) Consume(ctx context.Context, streamName, consumerName string, handler MessageHandler) error {
	stream, err := c.js.js.Stream(ctx, streamName)
	if err != nil {
		return fmt.Errorf("get stream: %w", err)
	}

	cons, err := stream.Consumer(ctx, consumerName)
	if err != nil {
		return fmt.Errorf("get consumer: %w", err)
	}

	cc, err := cons.Consume(func(msg jetstream.Msg) {
		if err = handler(ctx, msg.Data()); err != nil {
			msg.Nak()
			return
		}

		msg.Ack()
	})
	if err != nil {
		return fmt.Errorf("consume messages: %w", err)
	}

	<-ctx.Done()

	cc.Stop()

	return nil
}
