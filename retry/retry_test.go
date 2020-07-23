// Copyright 2020 YourBase Inc.
// SPDX-License-Identifier: BSD-3-Clause

package retry

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"zombiezen.com/go/log/testlog"
)

func TestImmediateSuccess(t *testing.T) {
	ctx := testlog.WithTB(context.Background(), t)
	ncalls := 0
	f := func() error {
		ncalls++
		return nil
	}
	err := Do(ctx, "calling a function", constBackoff(0), f)
	if err != nil {
		t.Error("Do:", err)
	}
	if ncalls != 1 {
		t.Errorf("f called %d times; want 1 time", ncalls)
	}
}

func TestSecondTimeSuccess(t *testing.T) {
	ctx := testlog.WithTB(context.Background(), t)
	ncalls := 0
	f := func() error {
		ncalls++
		if ncalls == 1 {
			return errors.New("bork")
		}
		return nil
	}
	err := Do(ctx, "calling a function", constBackoff(0), f)
	if err != nil {
		t.Error("Do:", err)
	}
	if ncalls != 2 {
		t.Errorf("f called %d times; want 2 times", ncalls)
	}
}

func TestNoWaitLoop(t *testing.T) {
	t.Run("CanceledBeforeStart", func(t *testing.T) {
		ctx, cancel := context.WithCancel(testlog.WithTB(context.Background(), t))
		cancel()
		ncalls := 0
		want := errors.New("bork")
		f := func() error {
			ncalls++
			return want
		}
		got := Do(ctx, "calling a function", constBackoff(0), f)
		if !errors.Is(got, want) {
			t.Errorf("Do = %v; want %v", got, want)
		}
		if ncalls != 1 {
			t.Errorf("f called %d times; want 1 time", ncalls)
		}
	})

	t.Run("CanceledDuringFirstRun", func(t *testing.T) {
		ctx, cancel := context.WithCancel(testlog.WithTB(context.Background(), t))
		ncalls := 0
		want := errors.New("bork")
		f := func() error {
			ncalls++
			cancel()
			return want
		}
		got := Do(ctx, "calling a function", constBackoff(0), f)
		if !errors.Is(got, want) {
			t.Errorf("Do = %v; want %v", got, want)
		}
		if ncalls != 1 {
			t.Errorf("f called %d times; want 1 time", ncalls)
		}
	})

	t.Run("CanceledDuringSecondRun", func(t *testing.T) {
		ctx, cancel := context.WithCancel(testlog.WithTB(context.Background(), t))
		ncalls := 0
		want := errors.New("bork")
		f := func() error {
			ncalls++
			if ncalls >= 2 {
				cancel()
			}
			return want
		}
		got := Do(ctx, "calling a function", constBackoff(0), f)
		if !errors.Is(got, want) {
			t.Errorf("Do = %v; want %v", got, want)
		}
		if ncalls != 2 {
			t.Errorf("f called %d times; want 2 times", ncalls)
		}
	})
}

// TestSleepLoop exercises Do with non-zero sleeps between function calls.
func TestSleepLoop(t *testing.T) {
	// Must be non-zero, but low to avoid slow tests.
	const sleepInterval = 1 * time.Millisecond

	t.Run("CanceledBeforeStart", func(t *testing.T) {
		ctx, cancel := context.WithCancel(testlog.WithTB(context.Background(), t))
		cancel()
		ncalls := 0
		want := errors.New("bork")
		f := func() error {
			ncalls++
			return want
		}
		got := Do(ctx, "calling a function", constBackoff(sleepInterval), f)
		if !errors.Is(got, want) {
			t.Errorf("Do = %v; want %v", got, want)
		}
		if ncalls != 1 {
			t.Errorf("f called %d times; want 1 time", ncalls)
		}
	})

	t.Run("CanceledDuringFirstRun", func(t *testing.T) {
		ctx, cancel := context.WithCancel(testlog.WithTB(context.Background(), t))
		ncalls := 0
		want := errors.New("bork")
		f := func() error {
			ncalls++
			cancel()
			return want
		}
		got := Do(ctx, "calling a function", constBackoff(sleepInterval), f)
		if !errors.Is(got, want) {
			t.Errorf("Do = %v; want %v", got, want)
		}
		if ncalls != 1 {
			t.Errorf("f called %d times; want 1 time", ncalls)
		}
	})

	t.Run("CanceledDuringSecondRun", func(t *testing.T) {
		ctx, cancel := context.WithCancel(testlog.WithTB(context.Background(), t))
		ncalls := 0
		want := errors.New("bork")
		f := func() error {
			ncalls++
			if ncalls >= 2 {
				cancel()
			}
			return want
		}
		got := Do(ctx, "calling a function", constBackoff(sleepInterval), f)
		if !errors.Is(got, want) {
			t.Errorf("Do = %v; want %v", got, want)
		}
		if ncalls != 2 {
			t.Errorf("f called %d times; want 2 times", ncalls)
		}
	})
}

type constBackoff time.Duration

func (b constBackoff) Duration() time.Duration {
	return time.Duration(b)
}

func TestMain(m *testing.M) {
	testlog.Main(nil)
	os.Exit(m.Run())
}
