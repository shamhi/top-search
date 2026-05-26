package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

type JetStream struct {
	js jetstream.JetStream
}

func NewJetStream(conn *Conn) (*JetStream, error) {
	js, err := jetstream.New(conn.raw)
	if err != nil {
		return nil, fmt.Errorf("create jetstream: %w", err)
	}

	return &JetStream{js: js}, nil
}

func (j *JetStream) EnsureStream(ctx context.Context, streamName, subject string) error {
	_, err := j.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:              streamName,
		Subjects:          []string{subject},
		Retention:         jetstream.LimitsPolicy,
		Discard:           jetstream.DiscardOld,
		MaxConsumers:      -1,
		MaxMsgs:           -1,
		MaxBytes:          -1,
		MaxAge:            10 * time.Minute,
		MaxMsgsPerSubject: -1,
		MaxMsgSize:        1 << 20,
		Storage:           jetstream.FileStorage,
		Replicas:          1,
		Duplicates:        2 * time.Minute,
	})
	if err != nil {
		return fmt.Errorf("ensure stream: %w", err)
	}

	return nil
}

func (j *JetStream) EnsureConsumer(ctx context.Context, streamName, consumerName, subject string) error {
	_, err := j.js.CreateOrUpdateConsumer(ctx, streamName, jetstream.ConsumerConfig{
		Name:          consumerName,
		Durable:       consumerName,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       30 * time.Second,
		MaxDeliver:    3,
		FilterSubject: subject,
		MaxAckPending: 1024,
	})
	if err != nil {
		return fmt.Errorf("ensure consumer: %w", err)
	}

	return nil
}
