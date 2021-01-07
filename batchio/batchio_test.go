// Copyright 2020 YourBase Inc.
// SPDX-License-Identifier: BSD-3-Clause

package batchio

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestReader(t *testing.T) {
	tests := []struct {
		name  string
		size  int
		steps []readStep
		want  []string
	}{
		{
			name: "Empty",
			size: 64,
			want: []string{},
		},
		{
			name: "SingleBatch",
			size: 64,
			steps: []readStep{
				{data: "Hello, World!\n"},
			},
			want: []string{"Hello, World!\n"},
		},
		{
			name: "MultipleBatches",
			size: 5,
			steps: []readStep{
				{data: "Hello, World!\n"},
			},
			want: []string{
				"Hello",
				", Wor",
				"ld!\n",
			},
		},
		{
			name: "Timeout",
			size: 64,
			steps: []readStep{
				{data: "Hello, "},
				{waitBefore: true, data: "World!\n"},
			},
			want: []string{"Hello, ", "World!\n"},
		},
		{
			name: "LastBatch",
			size: 64,
			steps: []readStep{
				{data: "Last batch: "},
				{triggerCancel: true, waitBefore: true, data: "here ya go"},
			},
			want: []string{"Last batch: ", "here ya go"},
		},
	}

	ctx := context.Background()
	if d, ok := t.Deadline(); ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, d)
		t.Cleanup(cancel)
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(ctx)
			var cancelTriggered struct {
				mu  sync.Mutex
				val bool
			}
			r := &fakeReader{
				steps: test.steps,
				waits: make(chan struct{}, 1),
				cancel: func() {
					cancelTriggered.mu.Lock()
					cancelTriggered.val = true
					cancelTriggered.mu.Unlock()
					cancel()
				},
			}
			b := NewReader(r, test.size, 10*time.Millisecond)
			var got []string
			for {
				batch, err := b.Next(ctx)
				select {
				case r.waits <- struct{}{}:
				default:
				}
				if err != nil {
					if len(batch) > 0 {
						t.Errorf("Received a batch and an error after #%d", len(got))
					}
					switch {
					case errors.Is(err, io.EOF):
						// Expected
					case errors.Is(err, context.Canceled):
						// Accept context.Canceled if our test triggered a cancel.
						cancelTriggered.mu.Lock()
						ok := cancelTriggered.val
						cancelTriggered.mu.Unlock()
						if !ok {
							t.Errorf("Batch #%d error: %v (but did not cancel!)", len(got), err)
						}
					default:
						t.Errorf("Batch #%d error: %v", len(got), err)
					}
					break
				}
				if len(batch) == 0 {
					t.Errorf("Received an empty batch with no error after #%d", len(got))
					continue
				}
				got = append(got, string(batch))
			}
			last, err := b.Finish()
			if err != nil {
				t.Error("Finish:", err)
			}
			if len(last) > 0 {
				got = append(got, string(last))
			}
			if diff := cmp.Diff(test.want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("batches (-want +got):\n%s", diff)
			}
		})
	}

	// The batching in this test is non-deterministic because it occurs over
	// multiple reads. There is no guarantee the second read will complete before
	// the timeout. As such, we test for the full end string rather than
	// individual batches like in the above tests.
	t.Run("MultipleReads", func(t *testing.T) {
		const size = 64
		const want = "Hello, World!\n"
		r := &fakeReader{
			steps: []readStep{
				{data: "Hello, "},
				{data: "World!\n"},
			},
			waits: make(chan struct{}),
		}
		b := NewReader(r, size, 30*time.Second)
		buf := new(strings.Builder)
		batchCount := 0
		for {
			batch, err := b.Next(ctx)
			if err != nil {
				if len(batch) > 0 {
					t.Errorf("Received a batch and an error after #%d", batchCount)
				}
				if !errors.Is(err, io.EOF) {
					t.Errorf("Batch #%d error: %v", batchCount, err)
				}
				break
			}
			if len(batch) == 0 {
				t.Errorf("Received an empty batch with no error after #%d", batchCount)
				continue
			}
			buf.Write(batch)
			batchCount++
			if len(batch) > size {
				t.Errorf("Batch #%d has size %d (> %d) at position %d",
					batchCount, len(batch), size, buf.Len())
			}
		}
		t.Logf("Received %d batches", batchCount)
		last, err := b.Finish()
		if err != nil {
			t.Error("Finish:", err)
		}
		buf.Write(last)
		if got := buf.String(); got != want {
			t.Errorf("got %q; want %q", got, want)
		}
	})

	t.Run("NoProgress", func(t *testing.T) {
		b := NewReader(noProgressReader{}, 64, 30*time.Second)
		batch, err := b.Next(ctx)
		if len(batch) > 0 || !errors.Is(err, io.ErrNoProgress) {
			t.Errorf("b.Next(ctx) = %q, %v; want \"\", %v", batch, err, io.ErrNoProgress)
		}
		batch, err = b.Finish()
		if len(batch) > 0 || err != nil {
			t.Errorf("b.Finish() = %q, %v; want \"\", <nil>", batch, err)
		}
	})
}

type readStep struct {
	triggerCancel bool // close fakeReader.cancel at start of read
	waitBefore    bool // wait until Next returns before releasing bytes
	data          string
}

type fakeReader struct {
	remaining string
	steps     []readStep
	waits     chan struct{}
	cancel    context.CancelFunc
}

func (r *fakeReader) Read(p []byte) (n int, err error) {
	if len(r.remaining) > 0 {
		n = copy(p, r.remaining)
		r.remaining = r.remaining[n:]
		if len(r.remaining) == 0 && len(r.steps) == 0 {
			err = io.EOF
		}
		return
	}
	if len(r.steps) == 0 {
		return 0, io.EOF
	}
	curr := r.steps[0]
	r.steps = r.steps[1:]
	if curr.triggerCancel {
		r.cancel()
	}
	if curr.waitBefore {
		<-r.waits
	}
	n = copy(p, curr.data)
	r.remaining = curr.data[n:]
	if len(r.remaining) == 0 && len(r.steps) == 0 {
		err = io.EOF
	}
	return
}

func (r *fakeReader) Close() error {
	close(r.waits)
	return nil
}

type noProgressReader struct{}

func (noProgressReader) Read(p []byte) (n int, err error) {
	return 0, nil
}

func (noProgressReader) Close() error {
	return nil
}

func TestWriter(t *testing.T) {
	const tafb = 10 * time.Millisecond

	t.Run("SingleBatch", func(t *testing.T) {
		rec := new(batchRecorder)
		w := NewWriter(rec, 64, tafb)
		const want = "Hello, World!\n"
		writeStrings(t, w, want)
		if err := w.Flush(); err != nil {
			t.Error("w.Flush():", err)
		}
		got := rec.get()
		if diff := cmp.Diff([]string{want}, got); diff != "" {
			t.Errorf("batches (-want +got):\n%s", diff)
		}
	})

	t.Run("ExactSizeOfBatch", func(t *testing.T) {
		rec := new(batchRecorder)
		const want = "Hello, World!\n"
		w := NewWriter(rec, len(want), tafb)
		writeStrings(t, w, want)
		if err := w.Flush(); err != nil {
			t.Error("w.Flush():", err)
		}
		got := rec.get()
		if diff := cmp.Diff([]string{want}, got); diff != "" {
			t.Errorf("batches (-want +got):\n%s", diff)
		}
	})

	t.Run("MultipleWritesLessThanBatchSize", func(t *testing.T) {
		rec := new(batchRecorder)
		const batchSize = 64
		w := NewWriter(rec, batchSize, tafb)
		const want = "Hello, World!\n"
		writeStrings(t, w, "Hello", ", ", "World!\n")
		if err := w.Flush(); err != nil {
			t.Error("w.Flush():", err)
		}
		// We can't guarantee the exact batching because it's dependent on timing.
		if got := rec.get(); !isBatchingValid(got, want, batchSize) {
			t.Errorf("bad batching for %q, batch size = %d: %q", want, batchSize, got)
		}
	})

	t.Run("SecondWriteLongerThanBatchSize", func(t *testing.T) {
		rec := new(batchRecorder)
		const batchSize = 5
		w := NewWriter(rec, batchSize, tafb)
		const want = "Hello, World!\n"
		writeStrings(t, w, "He", "llo, World!\n")
		if err := w.Flush(); err != nil {
			t.Error("w.Flush():", err)
		}
		// We can't guarantee the exact batching because it's dependent on timing.
		if got := rec.get(); !isBatchingValid(got, want, batchSize) {
			t.Errorf("bad batching for %q, batch size = %d: %q", want, batchSize, got)
		}
	})

	t.Run("MultipleBatches", func(t *testing.T) {
		rec := new(batchRecorder)
		w := NewWriter(rec, 5, tafb)
		writeStrings(t, w, "Hello, World!\n")
		if err := w.Flush(); err != nil {
			t.Error("w.Flush():", err)
		}
		got := rec.get()
		if diff := cmp.Diff([]string{"Hello", ", Wor", "ld!\n"}, got); diff != "" {
			t.Errorf("batches (-want +got):\n%s", diff)
		}
	})

	t.Run("Timeout", func(t *testing.T) {
		rec := new(batchRecorder)
		w := NewWriter(rec, 64, tafb)
		const want = "Hello, World!\n"
		writeStrings(t, w, want)
		if t.Failed() {
			t.Fatal("Test already failed: skipping wait for results.")
		}
		rec.waitForBytes(1)
		got := rec.get()
		if diff := cmp.Diff([]string{want}, got); diff != "" {
			t.Errorf("batches (-want +got):\n%s", diff)
		}
	})
}

func writeStrings(t *testing.T, w io.Writer, s ...string) {
	for _, data := range s {
		n, err := io.WriteString(w, data)
		if n != len(data) || err != nil {
			t.Errorf("w.Write([]byte(%q)) = %d, %v; want %d, <nil>", data, n, err, len(data))
		}
	}
}

func isBatchingValid(batches []string, want string, batchSize int) bool {
	for _, batch := range batches {
		if len(batch) == 0 || len(batch) > batchSize {
			return false
		}
	}
	return strings.Join(batches, "") == want
}

// batchRecorder is an io.Writer that saves the individual Write calls it receives.
// It is safe to use concurrently.
type batchRecorder struct {
	mu      sync.Mutex
	cond    chan struct{}
	batches []string
}

func (rec *batchRecorder) Write(p []byte) (int, error) {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	rec.batches = append(rec.batches, string(p))
	if len(p) > 0 && rec.cond != nil {
		close(rec.cond)
		rec.cond = nil
	}
	return len(p), nil
}

func (rec *batchRecorder) get() []string {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	return append([]string(nil), rec.batches...)
}

// waitForBytes waits until the given number of bytes have been written to
// the writer.
func (rec *batchRecorder) waitForBytes(n int) {
	for {
		rec.mu.Lock()
		current := 0
		for _, b := range rec.batches {
			current += len(b)
		}
		if current >= n {
			rec.mu.Unlock()
			return
		}

		// Not enough bytes. Wait for the channel to notify.
		if rec.cond == nil {
			rec.cond = make(chan struct{})
		}
		c := rec.cond
		rec.mu.Unlock()
		<-c
	}
}
