package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/akshaya-cp/golang_project/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotificationNotFound = errors.New("notification not found")

type NotificationRepository struct {
	db *pgxpool.Pool
}

func NewNotificationRepository(db *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) Create(ctx context.Context, userID uuid.UUID, channel, recipient, subject, body string) (*model.Notification, error) {
	const q = `
		INSERT INTO notifications (user_id, channel, recipient, subject, body, status)
		VALUES ($1, $2, $3, $4, $5, 'pending')
		RETURNING id, user_id, channel, recipient, subject, body, status, attempts, last_error, created_at, updated_at, sent_at
	`
	row := r.db.QueryRow(ctx, q, userID, channel, recipient, subject, body)
	n, err := scanNotification(row)
	if err != nil {
		return nil, fmt.Errorf("insert notification: %w", err)
	}
	return n, nil
}

func (r *NotificationRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Notification, error) {
	const q = `
		SELECT id, user_id, channel, recipient, subject, body, status, attempts, last_error, created_at, updated_at, sent_at
		FROM notifications WHERE id = $1
	`
	row := r.db.QueryRow(ctx, q, id)
	n, err := scanNotification(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotificationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find notification by id: %w", err)
	}
	return n, nil
}

// ListByUser returns a user's notifications, newest first.
func (r *NotificationRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit int) ([]model.Notification, error) {
	const q = `
		SELECT id, user_id, channel, recipient, subject, body, status, attempts, last_error, created_at, updated_at, sent_at
		FROM notifications WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	return r.queryList(ctx, q, userID, limit)
}

// ListAll returns notifications across all users (admin view), newest first.
func (r *NotificationRepository) ListAll(ctx context.Context, limit int) ([]model.Notification, error) {
	const q = `
		SELECT id, user_id, channel, recipient, subject, body, status, attempts, last_error, created_at, updated_at, sent_at
		FROM notifications
		ORDER BY created_at DESC
		LIMIT $1
	`
	return r.queryList(ctx, q, limit)
}

// MarkSent transitions a notification to the sent state.
func (r *NotificationRepository) MarkSent(ctx context.Context, id uuid.UUID, attempts int) error {
	const q = `
		UPDATE notifications
		SET status = 'sent', attempts = $2, last_error = '', sent_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, q, id, attempts)
	if err != nil {
		return fmt.Errorf("mark sent: %w", err)
	}
	return nil
}

// MarkFailed records a terminal failure after retries are exhausted.
func (r *NotificationRepository) MarkFailed(ctx context.Context, id uuid.UUID, attempts int, lastErr string) error {
	const q = `
		UPDATE notifications
		SET status = 'failed', attempts = $2, last_error = $3, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, q, id, attempts, lastErr)
	if err != nil {
		return fmt.Errorf("mark failed: %w", err)
	}
	return nil
}

func (r *NotificationRepository) queryList(ctx context.Context, q string, args ...any) ([]model.Notification, error) {
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	var out []model.Notification
	for rows.Next() {
		n, err := scanNotification(rows)
		if err != nil {
			return nil, fmt.Errorf("scan notification: %w", err)
		}
		out = append(out, *n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func scanNotification(row scannable) (*model.Notification, error) {
	var (
		n      model.Notification
		sentAt *time.Time
	)
	err := row.Scan(
		&n.ID,
		&n.UserID,
		&n.Channel,
		&n.Recipient,
		&n.Subject,
		&n.Body,
		&n.Status,
		&n.Attempts,
		&n.LastError,
		&n.CreatedAt,
		&n.UpdatedAt,
		&sentAt,
	)
	if err != nil {
		return nil, err
	}
	n.SentAt = sentAt
	return &n, nil
}
