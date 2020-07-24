// Copyright 2020 YourBase Inc.
// SPDX-License-Identifier: BSD-3-Clause

package herokurequest

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddleware(t *testing.T) {
	tests := []struct {
		name string
		hdr  http.Header
		want string
	}{
		{
			name: "NoHeader",
			want: "",
		},
		{
			name: "HeaderEmpty",
			hdr: http.Header{
				http.CanonicalHeaderKey("X-Request-ID"): {""},
			},
			want: "",
		},
		{
			name: "Set",
			hdr: http.Header{
				// Minimum 20 characters
				http.CanonicalHeaderKey("X-Request-ID"): {"abcdefghijklmnopqrst"},
			},
			want: "abcdefghijklmnopqrst",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ch := make(chan string, 1)
			srv := httptest.NewServer(&Middleware{
				Wrap: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					ch <- ContextID(r.Context())
					w.WriteHeader(http.StatusNoContent)
				}),
			})
			t.Cleanup(srv.Close)

			req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
			if err != nil {
				t.Fatal(err)
			}
			for k, v := range test.hdr {
				req.Header[k] = v
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusNoContent {
				t.Errorf("status code = %d; want %d", resp.StatusCode, http.StatusNoContent)
			}
			select {
			case got := <-ch:
				if got != test.want {
					t.Errorf("ContextID(r.Context()) = %q; want %q", got, test.want)
				}
			default:
				t.Error("Handler not called")
			}
		})
	}
}
