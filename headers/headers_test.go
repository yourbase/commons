// Copyright 2020 YourBase Inc.
// SPDX-License-Identifier: BSD-3-Clause

package headers

import (
	"net/http"
	"testing"
)

func TestCanonical(t *testing.T) {
	constants := []string{
		Accept,
		AcceptCharset,
		AcceptEncoding,
		AcceptLanguage,
		AcceptRanges,
		Age,
		Authorization,
		CacheControl,
		ContentDisposition,
		ContentEncoding,
		ContentLength,
		ContentRange,
		ContentType,
		ETag,
		Expires,
		IfMatch,
		IfModifiedSince,
		IfNoneMatch,
		IfRange,
		IfUnmodifiedSince,
		LastModified,
		Location,
		Range,
		TransferEncoding,
		Vary,
		WWWAuthenticate,
		XForwardedFor,
		XForwardedHost,
		XForwardedProto,
	}
	for _, c := range constants {
		if want := http.CanonicalHeaderKey(c); want != c {
			t.Errorf("Canonical form of %q is %q", c, want)
		}
	}
}
