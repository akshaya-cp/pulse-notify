package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

// Producer publishes notification events to Kafka.
type Producer struct {
	writer *kafka.Writer
	topic  string
}

// NewProducer builds a Kafka writer. The writer manages its own connection
// pool and load-balances across brokers, so a single instance is shared.
func NewProducer(brokers []string, topic string) *Producer {
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  topic,
		Balancer:               &kafka.Hash{},
		RequiredAcks:           kafka.RequireAll,
		AllowAutoTopicCreation: true,
		BatchTimeout:           50 * time.Millisecond,
	}
	return &Producer{writer: writer, topic: topic}
}

// Publish serializes and writes a notification event. The event's UserID is
// used as the partition key so all of a user's notifications keep ordering.
func (p *Producer) Publish(ctx context.Context, evt NotificationEvent) error {
	payload, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(evt.UserID),
		Value: payload,
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("write kafka message: %w", err)
	}
	return nil
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
