// Copyright 2020 YourBase Inc.
// SPDX-License-Identifier: BSD-3-Clause

package batcher_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/yourbase/commons/batcher"
)

func Example() {
	ctx := context.Background()

	// The stream can be any io.ReadCloser that supports calling Close
	// concurrently with Read. Common examples include *os.File, net.Conn,
	// and net/http.Request.Body.
	stream := ioutil.NopCloser(strings.NewReader("Hello, World!"))

	// Set parameters for your batcher.
	const maxBatchSize = 5
	const timeAfterFirstByte = 10 * time.Second
	b := batcher.New(stream, maxBatchSize, timeAfterFirstByte)

	// Always call Finish to close the stream and read any buffered data.
	defer func() {
		last, err := b.Finish()
		if len(last) > 0 {
			fmt.Printf("%s\n", last)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "Finish error:", err)
			return
		}
	}()

	// Loop until stream encounters an error.
	for {
		batch, err := b.Next(ctx)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				fmt.Fprintln(os.Stderr, "Error:", err)
			}
			break
		}
		fmt.Printf("%s\n", batch)
	}

	// Output:
	// Hello
	// , Wor
	// ld!
}
