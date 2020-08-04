// Copyright 2020 YourBase Inc.
// SPDX-License-Identifier: BSD-3-Clause

package https

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestForce(t *testing.T) {
	tests := []struct {
		name         string
		host         string
		proto        string
		url          string
		wantRedirect bool
		wantLocation string
	}{
		{
			name:         "ForwardedHTTP",
			host:         "example.com",
			proto:        "http",
			url:          "http://example.com/foo",
			wantRedirect: true,
			wantLocation: "https://example.com/foo",
		},
		{
			name:         "HostSpoofHTTP",
			host:         "example.com",
			proto:        "http",
			url:          "http://hacker.example.com/foo",
			wantRedirect: true,
			wantLocation: "https://example.com/foo",
		},
		{
			name:         "ForwardedHTTPS",
			host:         "example.com",
			proto:        "https",
			url:          "http://example.com/foo",
			wantRedirect: false,
		},
		{
			name:         "Localhost",
			host:         "example.com",
			url:          "http://localhost:8080/foo",
			wantRedirect: false,
		},
		{
			name:         "BogusProtocol",
			host:         "example.com",
			proto:        "bogus",
			url:          "http://example.com/foo",
			wantRedirect: true,
			wantLocation: "https://example.com/foo",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var handler mockHandler
			req := httptest.NewRequest(http.MethodGet, test.url, nil)
			if test.proto != "" {
				req.Header.Set("X-Forwarded-Proto", test.proto)
			}

			rec := new(httptest.ResponseRecorder)
			Force(test.host, &handler).ServeHTTP(rec, req)
			resp := rec.Result()

			if test.wantRedirect {
				if got, want := resp.StatusCode, http.StatusPermanentRedirect; got != want {
					t.Errorf("status = %d (%s); want %d", got, http.StatusText(got), want)
				}
				if got, want := resp.Header.Get("Location"), test.wantLocation; got != want {
					t.Errorf("Location = %q; want %q", got, want)
				}
				if handler.called {
					t.Error("Handler called")
				}
			} else {
				if got, want := resp.StatusCode, http.StatusOK; got != want {
					t.Errorf("status = %d (%s); want %d", got, http.StatusText(got), want)
				}
				if !handler.called {
					t.Error("Handler not called")
				}
			}
		})
	}
}

type mockHandler struct {
	called bool
}

func (h *mockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.called = true
}
