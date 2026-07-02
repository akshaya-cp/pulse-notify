package events

import "time"

// NotificationEvent is the message published to Kafka when a notification is
// enqueued. Workers consume it and perform the actual (simulated) delivery.
type NotificationEvent struct {
	NotificationID string    `json:"notification_id"`
	UserID         string    `json:"user_id"`
	Channel        string    `json:"channel"`
	Recipient      string    `json:"recipient"`
	Subject        string    `json:"subject"`
	Body           string    `json:"body"`
	EnqueuedAt     time.Time `json:"enqueued_at"`
}
