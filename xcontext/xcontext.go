// This file is derived from golang.org/x/tools/internal/xcontext at
// 5f9351755fc13ce6b9542113c6e61967e89215f6.
//
// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// SPDX-License-Identifier: BSD-3-Clause

// Package xcontext is a package to offer the extra functionality we need
// from contexts that is not available from the standard context package.
package xcontext

import (
	"context"
	"sync"
	"time"
)

// IgnoreDeadline returns a context that keeps all the values of its parent context
// but detaches from the cancellation and error handling.
func IgnoreDeadline(ctx context.Context) context.Context { return noDeadlineContext{ctx} }

type noDeadlineContext struct{ parent context.Context }

func (v noDeadlineContext) Deadline() (time.Time, bool)       { return time.Time{}, false }
func (v noDeadlineContext) Done() <-chan struct{}             { return nil }
func (v noDeadlineContext) Err() error                        { return nil }
func (v noDeadlineContext) Value(key interface{}) interface{} { return v.parent.Value(key) }

// KeepAlive returns a context that keeps all the values of its parent context
// and ensures that it is not marked Done for at least d time.
func KeepAlive(parent context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	// Compute deadline as close to start of function as possible.
	minDeadline := time.Now().Add(d)

	parentDone := parent.Done()
	if parentDone == nil {
		// Optimization: if the parent context can never be Done, then the only Done
		// signal can be from the returned cancel function.
		return context.WithCancel(parent)
	}
	select {
	case <-parentDone:
		// Optimization: if the parent context has already been canceled, then the
		// given duration is the deadline.
		return context.WithDeadline(IgnoreDeadline(parent), minDeadline)
	default:
	}

	// Otherwise, keep Done open until d has elapsed or the parent's Done channel
	// is closed, whichever comes last. Calling the returned cancel function
	// has priority over either condition.
	timer := time.NewTimer(time.Until(minDeadline))
	k := &keepAlive{
		parent: parent,
		done:   make(chan struct{}),
	}
	if k.deadline, k.hasDeadline = parent.Deadline(); k.hasDeadline {
		if k.deadline.Before(minDeadline) {
			k.deadline = minDeadline
		}
	}
	var cancelOnce sync.Once
	canceled := make(chan struct{})
	cancelerDone := make(chan struct{})
	go func() {
		defer close(cancelerDone)
		defer timer.Stop()

		select {
		case <-timer.C:
			// Waited the minimum time. Now propagate the parent Done.
			select {
			case <-parentDone:
				k.stop(parent.Err())
			case <-canceled:
				k.stop(context.Canceled)
			}
		case <-canceled:
			// Canceled before the minimum time elapsed.
			k.stop(context.Canceled)
		}
	}()
	return k, func() {
		cancelOnce.Do(func() {
			close(canceled)
			<-cancelerDone
		})
	}
}

type keepAlive struct {
	parent      context.Context
	deadline    time.Time
	hasDeadline bool
	done        chan struct{}

	mu  sync.RWMutex
	err error
}

func (k *keepAlive) Deadline() (time.Time, bool) {
	return k.deadline, k.hasDeadline
}

func (k *keepAlive) Done() <-chan struct{} {
	return k.done
}

func (k *keepAlive) Err() error {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return k.err
}

func (k *keepAlive) stop(e error) {
	if e == nil {
		panic("keepAlive.stop called with nil error")
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	k.err = e
	close(k.done)
}

func (k *keepAlive) Value(key interface{}) interface{} {
	return k.parent.Value(key)
}
