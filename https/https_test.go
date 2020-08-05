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
		forceHost    string
		method       string
		proto        string
		url          string
		wantCode     int
		wantLocation string
	}{
		{
			name:         "ForwardedHTTP/Get",
			forceHost:    "example.com",
			method:       http.MethodGet,
			proto:        "http",
			url:          "http://example.com/foo",
			wantCode:     http.StatusMovedPermanently,
			wantLocation: "https://example.com/foo",
		},
		{
			name:         "ForwardedHTTP/Head",
			forceHost:    "example.com",
			method:       http.MethodHead,
			proto:        "http",
			url:          "http://example.com/foo",
			wantCode:     http.StatusMovedPermanently,
			wantLocation: "https://example.com/foo",
		},
		{
			name:      "ForwardedHTTP/Post",
			forceHost: "example.com",
			method:    http.MethodPost,
			proto:     "http",
			url:       "http://example.com/foo",
			wantCode:  http.StatusGone,
		},
		{
			name:         "HostSpoofHTTP",
			forceHost:    "example.com",
			method:       http.MethodGet,
			proto:        "http",
			url:          "http://hacker.example.com/foo",
			wantCode:     http.StatusMovedPermanently,
			wantLocation: "https://example.com/foo",
		},
		{
			name:      "ForwardedHTTPS",
			forceHost: "example.com",
			method:    http.MethodGet,
			proto:     "https",
			url:       "http://example.com/foo",
			wantCode:  http.StatusOK,
		},
		{
			name:      "Localhost",
			forceHost: "example.com",
			method:    http.MethodGet,
			url:       "http://localhost:8080/foo",
			wantCode:  http.StatusOK,
		},
		{
			name:         "BogusProtocol",
			forceHost:    "example.com",
			method:       http.MethodGet,
			proto:        "bogus",
			url:          "http://example.com/foo",
			wantCode:     http.StatusMovedPermanently,
			wantLocation: "https://example.com/foo",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var handler mockHandler
			req := httptest.NewRequest(test.method, test.url, nil)
			if test.proto != "" {
				req.Header.Set("X-Forwarded-Proto", test.proto)
			}

			rec := new(httptest.ResponseRecorder)
			Force(test.forceHost, &handler).ServeHTTP(rec, req)
			resp := rec.Result()

			if got, want := resp.StatusCode, test.wantCode; got != want {
				t.Errorf("status = %d (%s); want %d", got, http.StatusText(got), want)
			}
			if got, want := handler.called, test.wantCode == http.StatusOK; got != want {
				if got {
					t.Error("Handler called")
				} else {
					t.Error("Handler not called")
				}
			}
			if got, want := resp.Header.Get("Location"), test.wantLocation; got != want {
				t.Errorf("Location = %q; want %q", got, want)
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
