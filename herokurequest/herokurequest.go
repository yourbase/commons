// Copyright 2020 YourBase Inc.
// SPDX-License-Identifier: BSD-3-Clause

// Package herokurequest provides context information for the request ID.
// https://devcenter.heroku.com/articles/http-request-id
//
// Deprecated: Please import the new path, github.com/yourbase/commons/http/herokurequest.
package herokurequest

import (
	"context"

	"github.com/yourbase/commons/http/herokurequest"
)

// Middleware extracts the Heroku request ID from all incoming requests and
// sends the wrapped handler a request with a Context containing the request ID.
type Middleware = herokurequest.Middleware

// ContextID returns the Heroku request ID stored in the Context or the empty
// string if the Context did not come from Middleware.
func ContextID(ctx context.Context) string {
	return herokurequest.ContextID(ctx)
}
