package worker

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/akshaya-cp/golang_project/internal/events"
	"github.com/akshaya-cp/golang_project/internal/repository"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// Pool is a fan-out worker pool. A single dispatcher goroutine fetches messages
// from Kafka and hands them to a bounded set of worker goroutines over a
// channel. Each worker processes a message (with retries + backoff), persists
// the terminal state to Postgres, then commits the offset — yielding
// at-least-once processing with bounded concurrency.
type Pool struct {
	consumer    *events.Consumer
	repo        *repository.NotificationRepository
	notifier    Notifier
	log         *slog.Logger
	concurrency int
	maxRetries  int
}

func NewPool(consumer *events.Consumer, repo *repository.NotificationRepository, notifier Notifier, log *slog.Logger, concurrency, maxRetries int) *Pool {
	if concurrency < 1 {
		concurrency = 1
	}
	return &Pool{
		consumer:    consumer,
		repo:        repo,
		notifier:    notifier,
		log:         log,
		concurrency: concurrency,
		maxRetries:  maxRetries,
	}
}

// Run blocks until ctx is cancelled, draining in-flight work before returning.
func (p *Pool) Run(ctx context.Context) error {
	jobs := make(chan kafka.Message)

	var wg sync.WaitGroup
	for i := 0; i < p.concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			p.worker(ctx, id, jobs)
		}(i + 1)
	}

	p.log.Info("worker pool started", "workers", p.concurrency, "max_retries", p.maxRetries)

	// Dispatcher: fetch messages and fan them out to workers.
	var dispatchErr error
	for {
		msg, err := p.consumer.Fetch(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || ctx.Err() != nil {
				break
			}
			p.log.Error("kafka fetch failed", "error", err)
			// Brief pause before retrying so a broker hiccup doesn't hot-loop.
			select {
			case <-ctx.Done():
			case <-time.After(time.Second):
			}
			continue
		}

		select {
		case jobs <- msg:
		case <-ctx.Done():
			dispatchErr = ctx.Err()
		}
		if dispatchErr != nil {
			break
		}
	}

	close(jobs)
	wg.Wait()
	p.log.Info("worker pool stopped")
	return nil
}

func (p *Pool) worker(ctx context.Context, id int, jobs <-chan kafka.Message) {
	for msg := range jobs {
		p.handle(ctx, id, msg)
	}
}

func (p *Pool) handle(ctx context.Context, workerID int, msg kafka.Message) {
	var evt events.NotificationEvent
	if err := json.Unmarshal(msg.Value, &evt); err != nil {
		// Poison message: we can't process it, so log and commit to skip it.
		p.log.Error("skipping malformed message", "worker", workerID, "error", err, "offset", msg.Offset)
		p.commit(ctx, msg)
		return
	}

	log := p.log.With("worker", workerID, "notification_id", evt.NotificationID, "channel", evt.Channel)

	notifID, err := uuid.Parse(evt.NotificationID)
	if err != nil {
		log.Error("invalid notification id in event", "error", err)
		p.commit(ctx, msg)
		return
	}

	attempts, sendErr := p.deliverWithRetry(ctx, log, evt)

	// Persist terminal state. Use a detached context so a shutdown mid-flight
	// still records the result before we exit.
	dbCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if sendErr != nil {
		log.Warn("notification failed after retries", "attempts", attempts, "error", sendErr)
		if err := p.repo.MarkFailed(dbCtx, notifID, attempts, sendErr.Error()); err != nil {
			log.Error("failed to persist failed status", "error", err)
		}
	} else {
		log.Info("notification delivered", "attempts", attempts)
		if err := p.repo.MarkSent(dbCtx, notifID, attempts); err != nil {
			log.Error("failed to persist sent status", "error", err)
		}
	}

	p.commit(ctx, msg)
}

// deliverWithRetry attempts delivery up to maxRetries+1 times with exponential
// backoff, returning the number of attempts made and the final error (nil on
// success).
func (p *Pool) deliverWithRetry(ctx context.Context, log *slog.Logger, evt events.NotificationEvent) (int, error) {
	var lastErr error
	for attempt := 1; attempt <= p.maxRetries+1; attempt++ {
		if ctx.Err() != nil {
			return attempt - 1, ctx.Err()
		}

		lastErr = p.notifier.Send(ctx, evt)
		if lastErr == nil {
			return attempt, nil
		}

		log.Debug("delivery attempt failed", "attempt", attempt, "error", lastErr)

		if attempt <= p.maxRetries {
			backoff := time.Duration(1<<uint(attempt-1)) * 100 * time.Millisecond
			select {
			case <-ctx.Done():
				return attempt, ctx.Err()
			case <-time.After(backoff):
			}
		}
	}
	return p.maxRetries + 1, lastErr
}

func (p *Pool) commit(ctx context.Context, msg kafka.Message) {
	commitCtx := ctx
	if ctx.Err() != nil {
		// Still commit processed work during shutdown.
		var cancel context.CancelFunc
		commitCtx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
	}
	if err := p.consumer.Commit(commitCtx, msg); err != nil {
		p.log.Error("failed to commit offset", "error", err, "offset", msg.Offset)
	}
}
