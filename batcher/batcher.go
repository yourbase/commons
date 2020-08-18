// Copyright 2020 YourBase Inc.
// SPDX-License-Identifier: BSD-3-Clause

// Package batcher provides a mechanism for converting a stream of bytes into
// batches of approximately equal time and space. This provides a reasonable
// balance between throughput and latency.
package batcher

import (
	"context"
	"errors"
	"io"
	"time"
)

// A Batcher buffers an io.Reader to produce a sequence of batches.
type Batcher struct {
	r    io.ReadCloser
	tafb time.Duration

	buf   []byte
	nread int   // written by next() goroutine; read by Read goroutine
	err   error // written by Read goroutine

	read        chan int
	pendingRead bool
}

// New returns a new batcher that reads batches from r. The batches will be no
// larger than the given size and will wait at most tafb after the first byte
// before returning.
//
// It must be safe to call r.Close concurrently with r.Read.
func New(r io.ReadCloser, size int, tafb time.Duration) *Batcher {
	if r == nil {
		panic("newChunker(nil, ...)")
	}
	if size <= 0 {
		panic("newChunker(..., <non-positive size>)")
	}
	return &Batcher{
		r:    r,
		buf:  make([]byte, size),
		tafb: tafb,
		read: make(chan int, 1),
	}
}

// Next reads the next batch from c's underlying reader. Next reads until its
// buffer is full, the duration after the first byte has elapsed, its underlying
// reader returns an error, or the Context is Done, whichever comes first.
// The returned batch is valid until the next call to Next.
//
// Next will return either a batch or an error. Once the underlying reader has
// returned an error, the Next will return the same error on subsequent calls.
func (b *Batcher) Next(ctx context.Context) ([]byte, error) {
	// Wait on leftover read from last call.
	if b.pendingRead {
		select {
		case n := <-b.read:
			b.nread = copy(b.buf, b.buf[b.nread:b.nread+n])
			b.pendingRead = false
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	} else {
		b.nread = 0
	}

	var timeout <-chan time.Time
	for b.nread < len(b.buf) && b.err == nil {
		if timeout == nil && b.nread > 0 {
			timer := time.NewTimer(b.tafb)
			defer timer.Stop()
			timeout = timer.C
		}
		go func() {
			var n int
			for i := 0; i < 5; i++ {
				n, b.err = b.r.Read(b.buf[b.nread:])
				if n > 0 || b.err != nil {
					b.read <- n
					return
				}
			}
			b.err = io.ErrNoProgress
			b.read <- 0
		}()
		select {
		case n := <-b.read:
			b.nread += n
		case <-timeout:
			// Time After First Byte reached.
			b.pendingRead = true
			return b.buf[:b.nread:b.nread], nil
		case <-ctx.Done():
			b.pendingRead = true
			return b.buf[:b.nread:b.nread], nil
		}
	}
	if b.nread == 0 {
		return nil, b.err
	}
	return b.buf[:b.nread:b.nread], nil
}

// Finish closes the underlying reader and returns a final batch if a Read was
// pending. After the first call to Finish, it returns an error.
func (b *Batcher) Finish() ([]byte, error) {
	if b.r == nil {
		return nil, errors.New("batcher finish called multiple times")
	}
	err := b.r.Close()
	if !b.pendingRead {
		return nil, err
	}
	n := <-b.read
	b.pendingRead = false
	b.r = nil
	return b.buf[b.nread : b.nread+n], err
}
