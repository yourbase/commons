// Copyright 2020 YourBase Inc.
// SPDX-License-Identifier: BSD-3-Clause

package ctxwebsocket

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/websocket"
)

func TestReadMessage(t *testing.T) {
	t.Run("Background", func(t *testing.T) {
		c1, c2, err := pipe(t)
		if err != nil {
			t.Fatal(err)
		}
		const message = "Hello, World!\n"
		if err := c1.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
			t.Fatal(err)
		}
		typ, got, err := ReadMessage(context.Background(), c2)
		if err != nil {
			t.Fatal("ReadMessage:", err)
		}
		if typ != websocket.TextMessage || string(got) != message {
			t.Errorf("ReadMessage(ctx, c2) = %d, %q, <nil>; want %d, %q, <nil>", typ, got, websocket.TextMessage, message)
		}
	})
	t.Run("Canceled", func(t *testing.T) {
		c, _, err := pipe(t)
		if err != nil {
			t.Fatal(err)
		}
		_, _, err = ReadMessage(canceledContext(), c)
		if err == nil {
			t.Error("ReadMessage did not return error")
		}
	})
}

func TestWriteMessage(t *testing.T) {
	t.Run("Background", func(t *testing.T) {
		c1, c2, err := pipe(t)
		if err != nil {
			t.Fatal(err)
		}
		const message = "Hello, World!\n"
		if err := WriteMessage(context.Background(), c1, websocket.TextMessage, []byte(message)); err != nil {
			t.Errorf("WriteMessage(ctx, c1, websocket.TextMessage, %q): %v", message, err)
		}
		typ, got, err := c2.ReadMessage()
		if err != nil {
			t.Fatal("c2.ReadMessage:", err)
		}
		if typ != websocket.TextMessage || string(got) != message {
			t.Errorf("c2.ReadMessage() = %d, %q, <nil>; want %d, %q, <nil>", typ, got, websocket.TextMessage, message)
		}
	})
	t.Run("Canceled", func(t *testing.T) {
		c, _, err := pipe(t)
		if err != nil {
			t.Fatal(err)
		}
		const message = "Hello, World!\n"
		if err := WriteMessage(canceledContext(), c, websocket.TextMessage, []byte(message)); err == nil {
			t.Error("WriteMessage did not return error")
		}
	})
}

func TestPing(t *testing.T) {
	t.Run("Background", func(t *testing.T) {
		c1, c2, err := pipe(t)
		if err != nil {
			t.Fatal(err)
		}
		pingChan := make(chan string, 1)
		c2.SetPingHandler(func(data string) error {
			pingChan <- data
			return nil
		})

		const message = "ping"
		if err := Ping(context.Background(), c1, []byte(message)); err != nil {
			t.Errorf("Ping(ctx, c1, %q): %v", message, err)
		}
		c1.Close()
		c2.ReadMessage()
		got := <-pingChan
		if string(got) != message {
			t.Errorf("c2 ping message = %q; want %q", got, message)
		}
	})
	t.Run("Canceled", func(t *testing.T) {
		c, _, err := pipe(t)
		if err != nil {
			t.Fatal(err)
		}
		const message = "ping"
		if err := Ping(canceledContext(), c, []byte(message)); err == nil {
			t.Error("Ping did not return error")
		}
	})
}

func pipe(c cleanuper) (conn1, conn2 *websocket.Conn, err error) {
	type upgradeResult struct {
		conn *websocket.Conn
		err  error
	}
	ch := make(chan upgradeResult, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := new(websocket.Upgrader).Upgrade(w, r, nil)
		ch <- upgradeResult{conn, err}
	}))
	u := "ws" + srv.URL[len("http"):]
	conn1, _, err = websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		srv.Close()
		return nil, nil, err
	}
	result := <-ch
	if result.err != nil {
		conn1.Close()
		srv.Close()
		return nil, nil, err
	}
	c.Cleanup(func() {
		conn1.Close()
		result.conn.Close()
		srv.Close()
	})
	return conn1, result.conn, nil
}

type cleanuper interface {
	Cleanup(f func())
}

func canceledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}
