package acme

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/neko233-com/acme-go/internal/config"
	"github.com/neko233-com/acme-go/internal/hook"
)

// AutoRenewOptions controls the periodic renewal loop.
type AutoRenewOptions struct {
	Name           string
	Force          bool
	Interval       time.Duration
	RunImmediately bool
	Out            io.Writer
}

type retryPolicy struct {
	initialBackoff   time.Duration
	maxBackoff       time.Duration
	maxRetryAttempts int
}

type failureEvent struct {
	target           string
	attempt          int
	maxRetryAttempts int
	nextRetry        time.Duration
	interval         time.Duration
	finalFailure     bool
	err              error
	occurredAt       time.Time
}

type failureNotifier interface {
	Notify(event failureEvent) error
}

type commandFailureNotifier struct {
	commands []string
	out      io.Writer
}

func (n commandFailureNotifier) Notify(event failureEvent) error {
	if len(n.commands) == 0 {
		return nil
	}
	return hook.Run(n.commands, event.Env(), n.out)
}

func (e failureEvent) Env() map[string]string {
	return map[string]string{
		"ACME_LOOP_TARGET":             e.target,
		"ACME_LOOP_ATTEMPT":            strconv.Itoa(e.attempt),
		"ACME_LOOP_MAX_RETRY_ATTEMPTS": strconv.Itoa(e.maxRetryAttempts),
		"ACME_LOOP_NEXT_RETRY":         e.nextRetry.String(),
		"ACME_LOOP_INTERVAL":           e.interval.String(),
		"ACME_LOOP_FINAL_FAILURE":      strconv.FormatBool(e.finalFailure),
		"ACME_LOOP_ERROR":              e.err.Error(),
		"ACME_LOOP_OCCURRED_AT":        e.occurredAt.Format(time.RFC3339),
	}
}

// AutoRenewLoop runs renew cycles until the context is canceled.
//
// If options.Interval is zero, the value is loaded from cfg.Automation.
func AutoRenewLoop(ctx context.Context, cfg *config.Config, options AutoRenewOptions) error {
	if options.Out == nil {
		options.Out = io.Discard
	}

	interval := options.Interval
	if interval <= 0 {
		parsed, err := cfg.Automation.RenewIntervalDuration()
		if err != nil {
			return err
		}
		interval = parsed
	}
	if interval <= 0 {
		return fmt.Errorf("auto renew interval is required; set automation.renew_interval or pass an explicit interval")
	}

	retryBackoff, err := cfg.Automation.RetryBackoffDuration()
	if err != nil {
		return err
	}
	maxRetryBackoff, err := cfg.Automation.MaxRetryBackoffDuration()
	if err != nil {
		return err
	}
	policy := retryPolicy{
		initialBackoff:   retryBackoff,
		maxBackoff:       maxRetryBackoff,
		maxRetryAttempts: *cfg.Automation.MaxRetryAttempts,
	}
	notifier := commandFailureNotifier{commands: cfg.Automation.FailureCommands, out: options.Out}
	target := options.Name
	if target == "" {
		target = "all-certificates"
	}

	runOnce := func() error {
		_, err := Run(cfg, Options{
			Name:  options.Name,
			Force: options.Force,
			Mode:  ModeRenew,
			Out:   options.Out,
		})
		return err
	}

	cycle := func() error {
		return runRenewCycle(ctx, target, interval, policy, notifier, waitForDelay, options.Out, runOnce)
	}

	return runLoop(ctx, interval, options.RunImmediately, cycle)
}

func runLoop(ctx context.Context, interval time.Duration, runImmediately bool, runOnce func() error) error {
	if runImmediately {
		if err := runOnce(); err != nil {
			return err
		}
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := runOnce(); err != nil {
				return err
			}
		}
	}
}

func runRenewCycle(
	ctx context.Context,
	target string,
	interval time.Duration,
	policy retryPolicy,
	notifier failureNotifier,
	wait func(context.Context, time.Duration) error,
	out io.Writer,
	runOnce func() error,
) error {
	for attempt := 1; ; attempt++ {
		err := runOnce()
		if err == nil {
			if attempt > 1 {
				fmt.Fprintf(out, "[auto-renew] recovered after %d attempt(s) for %s\n", attempt, target)
			}
			return nil
		}

		finalFailure := attempt > policy.maxRetryAttempts
		nextRetry := time.Duration(0)
		if !finalFailure {
			nextRetry = retryDelay(policy.initialBackoff, policy.maxBackoff, attempt-1)
		}
		event := failureEvent{
			target:           target,
			attempt:          attempt,
			maxRetryAttempts: policy.maxRetryAttempts,
			nextRetry:        nextRetry,
			interval:         interval,
			finalFailure:     finalFailure,
			err:              err,
			occurredAt:       time.Now(),
		}
		fmt.Fprintf(out, "[auto-renew] attempt %d for %s failed: %v\n", attempt, target, err)
		if notifyErr := notifier.Notify(event); notifyErr != nil {
			fmt.Fprintf(out, "[auto-renew] failure notification error: %v\n", notifyErr)
		}
		if finalFailure {
			fmt.Fprintf(out, "[auto-renew] giving up for now; next scheduled run for %s remains in %s\n", target, interval)
			return nil
		}
		fmt.Fprintf(out, "[auto-renew] retrying %s in %s (%d/%d)\n", target, nextRetry, attempt, policy.maxRetryAttempts)
		if err := wait(ctx, nextRetry); err != nil {
			return err
		}
	}
}

func retryDelay(initial, max time.Duration, retryIndex int) time.Duration {
	if initial <= 0 {
		return 0
	}
	delay := initial
	for step := 0; step < retryIndex; step++ {
		delay *= 2
		if max > 0 && delay >= max {
			return max
		}
	}
	if max > 0 && delay > max {
		return max
	}
	return delay
}

func waitForDelay(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
