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
	"sync"
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
// be no larger than the given size and will wait at most the given time after
// the first byte before returning.
//
// It must be safe to call r.Close concurrently with r.Read.
func NewReader(r io.ReadCloser, size int, timeAfterFirstByte time.Duration) *Reader {
	if r == nil {
		panic("batchio.NewReader(nil, ...)")
	}
	if size <= 0 {
		panic("batchio.NewReader(..., <non-positive size>, ...)")
	}
	if timeAfterFirstByte < 0 {
		panic("batchio.NewReader(..., <negative time-after-first-byte>)")
	}
	return &Reader{
		r:    r,
		buf:  make([]byte, size),
		tafb: timeAfterFirstByte,
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

// A Writer is a buffered io.Writer that writes batches to an underlying
// io.Writer object. If an error occurs writing to a Writer, no more data will
// be accepted and all subsequent writes, and Flush, will return the error.
// After all data has been written, the client should call the Flush method to
// guarantee all data has been forwarded to the underlying io.Writer object.
type Writer struct {
	w         io.Writer
	tafb      time.Duration
	timerDone chan struct{} // sent to when the AfterFunc has completed

	mu        sync.Mutex
	buf       []byte // a writer goroutine is running iff len(buf) > 0
	err       error
	flushChan chan struct{} // signal to the writer goroutine to start (has a buffer of 1)
	timer     *time.Timer   // return value of AfterFunc that trigger a flush
	writeDone chan struct{} // closed when the writer goroutine returns
}

// NewWriter returns a new Writer that writes batches to w. The batches will
// be no larger than the given size and will wait at most the given time after
// the first byte in a batch before writing the whole batch.
func NewWriter(w io.Writer, size int, timeAfterFirstByte time.Duration) *Writer {
	if w == nil {
		panic("batchio.NewWriter(nil, ...)")
	}
	if size <= 0 {
		panic("batchio.NewWriter(..., <non-positive size>, ...)")
	}
	if timeAfterFirstByte < 0 {
		panic("batchio.NewWriter(..., <negative time-after-first-byte>)")
	}
	return &Writer{
		w:         w,
		buf:       make([]byte, 0, size),
		tafb:      timeAfterFirstByte,
		timerDone: make(chan struct{}),
	}
}

// Write writes the contents of p into the buffer. It returns the number of
// bytes written. If n < len(p), it also returns an error explaining why the
// write is short.
func (w *Writer) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.err != nil {
		return 0, w.err
	}
	if len(w.buf) > 0 {
		// Goroutine has started, but is waiting for flush.
		// Append data to buffer without exceeding capacity.
		n = copy(w.buf[len(w.buf):cap(w.buf)], p)
		w.buf = w.buf[:len(w.buf)+n]
		p = p[n:]
		if len(w.buf) < cap(w.buf) {
			// Not enough data to trigger a flush.
			return n, nil
		}
		w.flushLocked()
		if w.err != nil {
			return n, w.err
		}
	}
	// No goroutine running. First, synchronously batch any data from the
	// beginning of the current write until the remaining data is less than the
	// buffer size.
	for len(p) >= cap(w.buf) {
		var nn int
		nn, w.err = w.w.Write(p[:cap(w.buf)])
		n += nn
		if err != nil {
			w.err = err
			return n, w.err
		}
		p = p[nn:]
	}
	// Now the rest of the current write will fit inside the buffer.
	w.buf = append(w.buf, p...)
	n += len(p)
	// If the buffer has data, then we need to kick off a goroutine to write it.
	if len(w.buf) == 0 {
		return n, nil
	}
	flushChan := make(chan struct{}, 1) // variable captured for AfterFunc
	w.flushChan = flushChan
	w.timer = time.AfterFunc(w.tafb, func() {
		select {
		case flushChan <- struct{}{}:
		default:
			// Already signaled.
		}
	})
	w.writeDone = make(chan struct{})
	go w.backgroundWrite()
	return n, nil
}

func (w *Writer) backgroundWrite() {
	// Wait for first of:
	// a) buffer is full
	// b) timer has expired
	<-w.flushChan

	// Holding onto the lock while writing avoids having to communicate to the
	// main goroutine how much of the buffer we wrote.
	w.mu.Lock()
	defer w.mu.Unlock()
	_, w.err = w.w.Write(w.buf)

	// Reset for the next background write.
	// We don't need to synchronize with the AfterFunc because it doesn't block.
	w.buf = w.buf[:0]
	w.flushChan = nil
	w.timer.Stop()
	w.timer = nil
	close(w.writeDone)
	w.writeDone = nil
}

// Flush writes any buffered data to the underlying io.Writer.
func (w *Writer) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	for len(w.buf) > 0 {
		w.flushLocked()
	}
	return w.err
}

// flushLocked signals to the writer goroutine that it should proceed with the
// write and waits for it to finish. The caller must be holding onto w.mu and
// should always check w.err afterward.
func (w *Writer) flushLocked() {
	select {
	case w.flushChan <- struct{}{}:
	default:
		// Already signaled.
	}
	done := w.writeDone
	w.mu.Unlock()
	<-done
	w.mu.Lock()
}
