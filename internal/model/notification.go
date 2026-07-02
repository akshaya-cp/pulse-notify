package model

import (
	"time"

	"github.com/google/uuid"
)

// NotificationStatus tracks a notification through the async pipeline.
type NotificationStatus string

const (
	StatusPending NotificationStatus = "pending"
	StatusSent    NotificationStatus = "sent"
	StatusFailed  NotificationStatus = "failed"
)

// Notification is a delivery request stored in PostgreSQL. It is created in the
// "pending" state, published to Kafka, and updated by a worker once processed.
type Notification struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Channel   string
	Recipient string
	Subject   string
	Body      string
	Status    NotificationStatus
	Attempts  int
	LastError string
	CreatedAt time.Time
	UpdatedAt time.Time
	SentAt    *time.Time
}

// NotificationResponse is the public API shape.
type NotificationResponse struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Channel   string     `json:"channel"`
	Recipient string     `json:"recipient"`
	Subject   string     `json:"subject"`
	Body      string     `json:"body"`
	Status    string     `json:"status"`
	Attempts  int        `json:"attempts"`
	LastError string     `json:"last_error,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	SentAt    *time.Time `json:"sent_at,omitempty"`
}

func (n *Notification) ToResponse() NotificationResponse {
	return NotificationResponse{
		ID:        n.ID.String(),
		UserID:    n.UserID.String(),
		Channel:   n.Channel,
		Recipient: n.Recipient,
		Subject:   n.Subject,
		Body:      n.Body,
		Status:    string(n.Status),
		Attempts:  n.Attempts,
		LastError: n.LastError,
		CreatedAt: n.CreatedAt,
		SentAt:    n.SentAt,
	}
}
