// Copyright 2020 YourBase Inc.
// SPDX-License-Identifier: BSD-3-Clause

// Package retry provides a function for retrying an operation.
package retry

import (
	"context"
	"time"

	"zombiezen.com/go/log"
)

// A BackoffStrategy can be called repeatedly to obtain (presumably) increasing
// durations to wait between retries.
type BackoffStrategy interface {
	Duration() time.Duration
}

// Do calls a function repeatedly with exponential backoff until it returns a
// nil error. Do returns an error only if the passed-in function does not return
// nil before the Context is Done. The function is guaranteed to be called at
// least once.
//
// The operation should be a verb phrase like "talking to Alice" for logging.
func Do(ctx context.Context, operation string, strategy BackoffStrategy, f func() error) error {
	var t *time.Timer
	for {
		err := f()
		if err == nil {
			return nil
		}
		d := strategy.Duration()
		if d > 0 {
			log.Warnf(ctx, "Error %s (will retry in %v): %v", operation, d, err)
			if t == nil {
				t = time.NewTimer(d)
				defer t.Stop()
			} else {
				t.Reset(d)
			}
			select {
			case <-t.C:
			case <-ctx.Done():
				return err
			}
		} else {
			log.Warnf(ctx, "Error %s (will retry): %v", operation, d, err)
			select {
			case <-ctx.Done():
				return err
			default:
			}
		}
	}
}
