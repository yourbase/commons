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
