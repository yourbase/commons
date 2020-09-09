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
		Allow,
		Authorization,
		CacheControl,
		Connection,
		ContentDisposition,
		ContentEncoding,
		ContentLength,
		ContentRange,
		ContentType,
		ETag,
		Expect,
		Expires,
		IfMatch,
		IfModifiedSince,
		IfNoneMatch,
		IfRange,
		IfUnmodifiedSince,
		KeepAlive,
		LastModified,
		Location,
		Range,
		RetryAfter,
		TransferEncoding,
		Vary,
		WWWAuthenticate,
		XContentTypeOptions,
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
