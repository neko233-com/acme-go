package acme

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/neko233-com/acme-go/internal/config"
)

// AutoRenewOptions controls the periodic renewal loop.
type AutoRenewOptions struct {
	Name           string
	Force          bool
	Interval       time.Duration
	RunImmediately bool
	Out            io.Writer
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

	runOnce := func() error {
		_, err := Run(cfg, Options{
			Name:  options.Name,
			Force: options.Force,
			Mode:  ModeRenew,
			Out:   options.Out,
		})
		return err
	}

	return runLoop(ctx, interval, options.RunImmediately, runOnce)
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
