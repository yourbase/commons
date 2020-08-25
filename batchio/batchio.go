// Copyright 2020 YourBase Inc.
// SPDX-License-Identifier: BSD-3-Clause

// Package batchio provides mechanisms for converting a stream of bytes into
// batches of approximately equal time and space. This provides a reasonable
// balance between throughput and latency.
package batchio

import (
	"context"
	"errors"
	"io"
	"time"
)

// A Reader buffers an io.Reader to produce a sequence of batches.
type Reader struct {
	r    io.ReadCloser
	tafb time.Duration

	buf   []byte
	nread int   // written by next() goroutine; read by Read goroutine
	err   error // written by Read goroutine

	read        chan int
	pendingRead bool
}

// NewReader returns a new Reader that reads batches from r. The batches will
// be no larger than the given size and will wait at most tafb after the first
// byte before returning.
//
// It must be safe to call r.Close concurrently with r.Read.
func NewReader(r io.ReadCloser, size int, tafb time.Duration) *Reader {
	if r == nil {
		panic("batchio.NewReader(nil, ...)")
	}
	if size <= 0 {
		panic("batchio.NewReader(..., <non-positive size>, ...)")
	}
	if tafb < 0 {
		panic("batchio.NewReader(..., <negative time-after-first-byte>)")
	}
	return &Reader{
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
func (r *Reader) Next(ctx context.Context) ([]byte, error) {
	// Wait on leftover read from last call.
	if r.pendingRead {
		select {
		case n := <-r.read:
			r.nread = copy(r.buf, r.buf[r.nread:r.nread+n])
			r.pendingRead = false
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	} else {
		r.nread = 0
	}

	var timeout <-chan time.Time
	for r.nread < len(r.buf) && r.err == nil {
		if timeout == nil && r.nread > 0 {
			timer := time.NewTimer(r.tafb)
			defer timer.Stop()
			timeout = timer.C
		}
		go func() {
			var n int
			for i := 0; i < 5; i++ {
				n, r.err = r.r.Read(r.buf[r.nread:])
				if n > 0 || r.err != nil {
					r.read <- n
					return
				}
			}
			r.err = io.ErrNoProgress
			r.read <- 0
		}()
		select {
		case n := <-r.read:
			r.nread += n
		case <-timeout:
			// Time After First Byte reached.
			r.pendingRead = true
			return r.buf[:r.nread:r.nread], nil
		case <-ctx.Done():
			r.pendingRead = true
			if r.nread == 0 {
				return nil, ctx.Err()
			}
			return r.buf[:r.nread:r.nread], nil
		}
	}
	if r.nread == 0 {
		return nil, r.err
	}
	return r.buf[:r.nread:r.nread], nil
}

// Finish closes the underlying reader and returns a final batch if a Read was
// pending. After the first call to Finish, it returns an error.
func (r *Reader) Finish() ([]byte, error) {
	if r.r == nil {
		return nil, errors.New("batchio.Reader.Finish called multiple times")
	}
	err := r.r.Close()
	if !r.pendingRead {
		return nil, err
	}
	n := <-r.read
	r.pendingRead = false
	r.r = nil
	return r.buf[r.nread : r.nread+n], err
}
