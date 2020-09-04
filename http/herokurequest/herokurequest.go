// Copyright 2020 YourBase Inc.
// SPDX-License-Identifier: BSD-3-Clause

// Package herokurequest provides context information for the request ID.
// https://devcenter.heroku.com/articles/http-request-id
package herokurequest

import (
	"context"
	"net/http"
)

type contextKey struct{}

// Middleware extracts the Heroku request ID from all incoming requests and
// sends the wrapped handler a request with a Context containing the request ID.
type Middleware struct {
	Wrap http.Handler
}

func (m *Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := r.Header.Get("X-Request-ID")
	if id != "" {
		r = r.WithContext(context.WithValue(r.Context(), contextKey{}, id))
	}
	m.Wrap.ServeHTTP(w, r)
}

// ContextID returns the Heroku request ID stored in the Context or the empty
// string if the Context did not come from Middleware.
func ContextID(ctx context.Context) string {
	id, _ := ctx.Value(contextKey{}).(string)
	return id
}
