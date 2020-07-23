// Copyright 2020 YourBase Inc.
// SPDX-License-Identifier: BSD-3-Clause

// Package ctxwebsocket provides context-aware I/O functions on WebSockets.
package ctxwebsocket

import (
	"context"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

// ReadMessage reads the next message from the connection.
func ReadMessage(ctx context.Context, conn *websocket.Conn) (messageType int, p []byte, err error) {
	ctxDone := ctx.Done()
	if ctxDone == nil {
		return conn.ReadMessage()
	}
	select {
	case <-ctxDone:
		return 0, nil, fmt.Errorf("read websocket message: %w", ctx.Err())
	default:
	}
	read := make(chan struct{})
	watchDone := make(chan struct{})
	go func() {
		close(watchDone)
		select {
		case <-read:
		case <-ctxDone:
			conn.SetReadDeadline(time.Now())
		}
	}()
	messageType, p, err = conn.ReadMessage()
	close(read)
	<-watchDone
	return
}

// WriteMessage writes a message to the connection.
func WriteMessage(ctx context.Context, conn *websocket.Conn, messageType int, data []byte) error {
	ctxDone := ctx.Done()
	if ctxDone == nil {
		return conn.WriteMessage(messageType, data)
	}
	select {
	case <-ctxDone:
		return fmt.Errorf("write websocket message: %w", ctx.Err())
	default:
	}
	written := make(chan struct{})
	watchDone := make(chan struct{})
	go func() {
		close(watchDone)
		select {
		case <-written:
		case <-ctxDone:
			// XXX This is racy because WriteMessage will unconditionally call
			// SetWriteDeadline.
			conn.UnderlyingConn().SetWriteDeadline(time.Now())
		}
	}()
	err := conn.WriteMessage(messageType, data)
	close(written)
	<-watchDone
	return err
}

// Ping writes a ping message to the connection. It is safe to call concurrently
// with WriteMessage on the same connection.
func Ping(ctx context.Context, conn *websocket.Conn, data []byte) error {
	ctxDone := ctx.Done()
	if ctxDone == nil {
		return conn.WriteControl(websocket.PingMessage, data, time.Time{})
	}
	select {
	case <-ctxDone:
		return fmt.Errorf("ping websocket: %w", ctx.Err())
	default:
	}
	written := make(chan struct{})
	watchDone := make(chan struct{})
	go func() {
		close(watchDone)
		select {
		case <-written:
		case <-ctxDone:
			// XXX This is racy because WriteControl will unconditionally call
			// SetWriteDeadline.
			conn.UnderlyingConn().SetWriteDeadline(time.Now())
		}
	}()
	err := conn.WriteControl(websocket.PingMessage, data, time.Time{})
	close(written)
	<-watchDone
	return err
}
