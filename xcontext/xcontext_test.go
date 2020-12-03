// Copyright 2020 YourBase Inc.
// SPDX-License-Identifier: BSD-3-Clause

package xcontext

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestKeepAlive(t *testing.T) {
	const keepAlive = 10 * time.Millisecond

	t.Run("ParentBackground", func(t *testing.T) {
		k, cancelK := KeepAlive(context.Background(), keepAlive)
		defer cancelK()
		time.Sleep(keepAlive)
		select {
		case <-k.Done():
			t.Errorf("KeepAlive(ctx, %v).Done() closed before cancel", keepAlive)
		default:
		}

		cancelK()
		<-k.Done()
		if got := k.Err(); !errors.Is(got, context.Canceled) {
			t.Errorf("KeepAlive(ctx, %v).Err() = %v; want %v", keepAlive, got, context.Canceled)
		}
	})

	t.Run("CancelParentBeforeWait", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		start := time.Now()
		k, cancelK := KeepAlive(ctx, keepAlive)
		defer cancelK()

		<-k.Done()
		if got := time.Since(start); got < keepAlive {
			t.Errorf("KeepAlive(ctx, %v).Done() closed after %v; want at least %v",
				keepAlive, got, keepAlive)
		}
		if got := k.Err(); !errors.Is(got, context.DeadlineExceeded) {
			t.Errorf("KeepAlive(ctx, %v).Err() = %v; want %v", keepAlive, got, context.DeadlineExceeded)
		}
	})

	t.Run("CancelParentDuringWait", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		start := time.Now()
		k, cancelK := KeepAlive(ctx, keepAlive)
		defer cancelK()

		cancel()
		<-k.Done()
		if got := time.Since(start); got < keepAlive {
			t.Errorf("KeepAlive(ctx, %v).Done() closed after %v; want at least %v",
				keepAlive, got, keepAlive)
		}
		if got := k.Err(); !errors.Is(got, context.Canceled) {
			t.Errorf("KeepAlive(ctx, %v).Err() = %v; want %v", keepAlive, got, context.Canceled)
		}
	})

	t.Run("CancelChildDuringWait", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		k, cancelK := KeepAlive(ctx, keepAlive)
		defer cancelK()

		cancelK()
		<-k.Done()
		if got := k.Err(); !errors.Is(got, context.Canceled) {
			t.Errorf("KeepAlive(ctx, %v).Err() = %v; want %v", keepAlive, got, context.Canceled)
		}
	})

	t.Run("CancelChildAfterWait", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		k, cancelK := KeepAlive(ctx, keepAlive)
		defer cancelK()

		time.Sleep(keepAlive)
		cancelK()
		<-k.Done()
		if got := k.Err(); !errors.Is(got, context.Canceled) {
			t.Errorf("KeepAlive(ctx, %v).Err() = %v; want %v", keepAlive, got, context.Canceled)
		}
	})

	t.Run("ExtendDeadline", func(t *testing.T) {
		testStart := time.Now()
		ctx, cancel := context.WithDeadline(context.Background(), testStart.Add(keepAlive-1*time.Millisecond))
		defer cancel()

		start := time.Now()
		k, cancelK := KeepAlive(ctx, keepAlive)
		defer cancelK()

		want := start.Add(keepAlive)
		if got, ok := k.Deadline(); !ok || got.Before(want) {
			t.Errorf("KeepAlive(ctx, %v).Deadline() = %v, %t; want >=%v, true", keepAlive, got, ok, want)
		}
	})

	t.Run("LongDeadline", func(t *testing.T) {
		// This test can fail if executed too slowly, but we intentionally pick a
		// really long timeout we hope to never see.
		want := time.Now().Add(9000 * time.Hour)
		ctx, cancel := context.WithDeadline(context.Background(), want)
		defer cancel()

		k, cancelK := KeepAlive(ctx, keepAlive)
		defer cancelK()

		if got, ok := k.Deadline(); !ok || !got.Equal(want) {
			t.Errorf("KeepAlive(ctx, %v).Deadline() = %v, %t; want %v, true", keepAlive, got, ok, want)
		}
	})
}
