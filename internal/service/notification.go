package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/akshaya-cp/golang_project/internal/cache"
	"github.com/akshaya-cp/golang_project/internal/events"
	"github.com/akshaya-cp/golang_project/internal/model"
	"github.com/akshaya-cp/golang_project/internal/repository"
	"github.com/google/uuid"
)

// allowedChannels defines the delivery channels the platform accepts.
var allowedChannels = map[string]bool{
	"email": true,
	"sms":   true,
	"push":  true,
}

// NotificationProducer is the subset of the Kafka producer the service needs.
type NotificationProducer interface {
	Publish(ctx context.Context, evt events.NotificationEvent) error
}

type NotificationService struct {
	repo     *repository.NotificationRepository
	producer NotificationProducer
	cache    *cache.Client
	log      *slog.Logger
}

func NewNotificationService(repo *repository.NotificationRepository, producer NotificationProducer, c *cache.Client, log *slog.Logger) *NotificationService {
	return &NotificationService{repo: repo, producer: producer, cache: c, log: log}
}

// Enqueue validates the request, persists a pending notification, and publishes
// an event to Kafka for asynchronous processing by the worker pool.
func (s *NotificationService) Enqueue(ctx context.Context, userID uuid.UUID, channel, recipient, subject, body string) (*model.NotificationResponse, error) {
	channel = strings.ToLower(strings.TrimSpace(channel))
	if !allowedChannels[channel] {
		return nil, fmt.Errorf("unsupported channel %q: must be one of email, sms, push", channel)
	}

	n, err := s.repo.Create(ctx, userID, channel, recipient, subject, body)
	if err != nil {
		return nil, err
	}

	evt := events.NotificationEvent{
		NotificationID: n.ID.String(),
		UserID:         n.UserID.String(),
		Channel:        n.Channel,
		Recipient:      n.Recipient,
		Subject:        n.Subject,
		Body:           n.Body,
		EnqueuedAt:     time.Now().UTC(),
	}

	if err := s.producer.Publish(ctx, evt); err != nil {
		// The row is persisted as pending; surface the publish failure so the
		// caller knows the event did not reach the queue.
		return nil, fmt.Errorf("publish notification event: %w", err)
	}

	s.invalidateUserCache(ctx, userID)

	resp := n.ToResponse()
	return &resp, nil
}

func (s *NotificationService) Get(ctx context.Context, id uuid.UUID) (*model.NotificationResponse, error) {
	n, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := n.ToResponse()
	return &resp, nil
}

func (s *NotificationService) ListForUser(ctx context.Context, userID uuid.UUID, limit int) ([]model.NotificationResponse, error) {
	limit = normalizeLimit(limit)
	cacheKey := fmt.Sprintf("notif:list:%s:%d", userID.String(), limit)

	if s.cache != nil {
		var cached []model.NotificationResponse
		if hit, err := s.cache.GetJSON(ctx, cacheKey, &cached); err == nil && hit {
			return cached, nil
		}
	}

	items, err := s.repo.ListByUser(ctx, userID, limit)
	if err != nil {
		return nil, err
	}
	resp := toResponses(items)

	if s.cache != nil {
		_ = s.cache.SetJSON(ctx, cacheKey, resp, 15*time.Second)
	}
	return resp, nil
}

func (s *NotificationService) ListAll(ctx context.Context, limit int) ([]model.NotificationResponse, error) {
	limit = normalizeLimit(limit)
	items, err := s.repo.ListAll(ctx, limit)
	if err != nil {
		return nil, err
	}
	return toResponses(items), nil
}

func (s *NotificationService) invalidateUserCache(ctx context.Context, userID uuid.UUID) {
	if s.cache == nil {
		return
	}
	// The list cache is short-lived; deleting known limits keeps reads fresh.
	for _, limit := range []int{20, 50, 100} {
		_ = s.cache.Delete(ctx, fmt.Sprintf("notif:list:%s:%d", userID.String(), limit))
	}
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func toResponses(items []model.Notification) []model.NotificationResponse {
	out := make([]model.NotificationResponse, 0, len(items))
	for i := range items {
		out = append(out, items[i].ToResponse())
	}
	return out
}
