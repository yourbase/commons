// Copyright 2020 YourBase Inc.
// SPDX-License-Identifier: BSD-3-Clause

// Package https provides middleware to redirect users to HTTPS if they connect
// via HTTP.
package https

import (
	"net/http"
)

type middleware struct {
	host string
	wrap http.Handler
}

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
	return middleware{host, handler}
}

func (m middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "https" && proto != "" {
		u := *r.URL
		u.Scheme = "https"
		u.Host = m.host
		// "Permanent Redirect" instructs the client to retry with the same method.
		// https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/308
		http.Redirect(w, r, u.String(), http.StatusPermanentRedirect)
		return
	}
	m.wrap.ServeHTTP(w, r)
}
