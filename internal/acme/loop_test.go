package acme

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

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
