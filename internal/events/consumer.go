package events

import (
	"context"
	"time"

	"github.com/segmentio/kafka-go"
)

// Consumer wraps a Kafka consumer-group reader. Offsets are committed manually
// after a message is successfully processed, giving at-least-once delivery.
type Consumer struct {
	reader *kafka.Reader
}

func NewConsumer(brokers []string, topic, groupID string) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: 0, // commit explicitly via CommitMessages
		MaxWait:        500 * time.Millisecond,
	})
	return &Consumer{reader: reader}
}

// Fetch returns the next message without committing its offset.
func (c *Consumer) Fetch(ctx context.Context) (kafka.Message, error) {
	return c.reader.FetchMessage(ctx)
}

// Commit marks the given messages as processed.
func (c *Consumer) Commit(ctx context.Context, msgs ...kafka.Message) error {
	return c.reader.CommitMessages(ctx, msgs...)
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
