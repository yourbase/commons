// Copyright 2020 YourBase Inc.
// SPDX-License-Identifier: BSD-3-Clause

// Package https provides middleware to redirect users to HTTPS if they connect
// via HTTP.
//
// Deprecated: Please import the new path, github.com/yourbase/commons/http/https.
package https

import (
	"net/http"

	"github.com/yourbase/commons/http/https"
)

// Force returns a handler that redirects any HTTP requests to HTTPS on the
// given host. HTTPS requests are passed through to the given handler. The host
// must not come from user input or else an attacker could send traffic to a
// different domain.
//
// In production, Heroku terminates HTTPS before it reaches us, but they place
// an X-Forwarded-Proto header in the forwarded request. If it's absent, we're
// probably on localhost, so allow it.
//
// See https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/X-Forwarded-Proto
// and https://help.heroku.com/J2R1S4T8/can-heroku-force-an-application-to-use-ssl-tls
// for more details.
func Force(host string, handler http.Handler) http.Handler {
	return https.Force(host, handler)
}
