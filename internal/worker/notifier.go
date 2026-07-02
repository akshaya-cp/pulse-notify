package worker

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/akshaya-cp/golang_project/internal/events"
)

// Notifier delivers a notification over its channel. In a real system this
// would call an email/SMS/push provider; here it is simulated.
type Notifier interface {
	Send(ctx context.Context, evt events.NotificationEvent) error
}

// SimulatedNotifier mimics a downstream provider: it adds a little latency and
// fails intermittently so the retry and dead-letter paths are observable.
type SimulatedNotifier struct {
	failureRate float64
}

func NewSimulatedNotifier(failureRate float64) *SimulatedNotifier {
	return &SimulatedNotifier{failureRate: failureRate}
}

func (n *SimulatedNotifier) Send(ctx context.Context, evt events.NotificationEvent) error {
	// Simulate network/provider latency, respecting cancellation.
	delay := time.Duration(50+rand.Intn(150)) * time.Millisecond
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(delay):
	}

	if rand.Float64() < n.failureRate {
		return fmt.Errorf("provider rejected %s delivery to %s", evt.Channel, evt.Recipient)
	}
	return nil
}
