package acme

import (
	"bytes"
	"context"
	"sync/atomic"
	"testing"
	"time"
)

type fakeFailureNotifier struct {
	events []failureEvent
}

func (n *fakeFailureNotifier) Notify(event failureEvent) error {
	n.events = append(n.events, event)
	return nil
}

func TestRunLoopRunsImmediatelyAndStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var runs int32
	err := runLoop(ctx, 10*time.Millisecond, true, func() error {
		if atomic.AddInt32(&runs, 1) == 2 {
			cancel()
		}
		return nil
	})
	if err != nil {
		t.Fatalf("runLoop: %v", err)
	}
	if got := atomic.LoadInt32(&runs); got != 2 {
		t.Fatalf("runs: got %d want 2", got)
	}
}

func TestRunLoopWaitsForFirstTickWhenImmediateDisabled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var runs int32
	err := runLoop(ctx, 10*time.Millisecond, false, func() error {
		atomic.AddInt32(&runs, 1)
		cancel()
		return nil
	})
	if err != nil {
		t.Fatalf("runLoop: %v", err)
	}
	if got := atomic.LoadInt32(&runs); got != 1 {
		t.Fatalf("runs: got %d want 1", got)
	}
}

func TestRunRenewCycleRetriesWithBackoffAndRecovers(t *testing.T) {
	ctx := context.Background()
	var waits []time.Duration
	notifier := &fakeFailureNotifier{}
	var out bytes.Buffer
	attempts := 0

	err := runRenewCycle(
		ctx,
		"example",
		24*time.Hour,
		retryPolicy{initialBackoff: time.Second, maxBackoff: 4 * time.Second, maxRetryAttempts: 3},
		notifier,
		func(_ context.Context, delay time.Duration) error {
			waits = append(waits, delay)
			return nil
		},
		&out,
		func() error {
			attempts++
			if attempts < 3 {
				return context.DeadlineExceeded
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("runRenewCycle: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts: got %d want 3", attempts)
	}
	if len(waits) != 2 || waits[0] != time.Second || waits[1] != 2*time.Second {
		t.Fatalf("waits: got %v", waits)
	}
	if len(notifier.events) != 2 {
		t.Fatalf("notifications: got %d want 2", len(notifier.events))
	}
	if notifier.events[1].finalFailure {
		t.Fatal("expected second notification to be retryable, not final")
	}
}

func TestRunRenewCycleStopsAfterMaxRetriesAndKeepsLoopAlive(t *testing.T) {
	ctx := context.Background()
	var waits []time.Duration
	notifier := &fakeFailureNotifier{}
	var out bytes.Buffer
	attempts := 0

	err := runRenewCycle(
		ctx,
		"all-certificates",
		24*time.Hour,
		retryPolicy{initialBackoff: time.Second, maxBackoff: 2 * time.Second, maxRetryAttempts: 2},
		notifier,
		func(_ context.Context, delay time.Duration) error {
			waits = append(waits, delay)
			return nil
		},
		&out,
		func() error {
			attempts++
			return context.DeadlineExceeded
		},
	)
	if err != nil {
		t.Fatalf("runRenewCycle: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts: got %d want 3", attempts)
	}
	if len(waits) != 2 || waits[0] != time.Second || waits[1] != 2*time.Second {
		t.Fatalf("waits: got %v", waits)
	}
	if len(notifier.events) != 3 {
		t.Fatalf("notifications: got %d want 3", len(notifier.events))
	}
	if !notifier.events[2].finalFailure {
		t.Fatal("expected final notification after retries are exhausted")
	}
}
