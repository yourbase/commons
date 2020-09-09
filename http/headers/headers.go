// Copyright 2020 YourBase Inc.
// SPDX-License-Identifier: BSD-3-Clause

// Package headers provides constants for well-known HTTP headers.
package headers

// HTTP header constants, all in canonical format.
const (
	// Message body
	ContentEncoding  = "Content-Encoding"
	ContentLength    = "Content-Length"
	ContentType      = "Content-Type"
	TransferEncoding = "Transfer-Encoding"

	// Authentication
	Authorization   = "Authorization"
	WWWAuthenticate = "Www-Authenticate"

	// Caching
	Age               = "Age"
	CacheControl      = "Cache-Control"
	ETag              = "Etag"
	Expires           = "Expires"
	IfMatch           = "If-Match"
	IfModifiedSince   = "If-Modified-Since"
	IfNoneMatch       = "If-None-Match"
	IfUnmodifiedSince = "If-Unmodified-Since"
	LastModified      = "Last-Modified"
	Vary              = "Vary"

	// Content negotiation
	Accept         = "Accept"
	AcceptCharset  = "Accept-Charset"
	AcceptEncoding = "Accept-Encoding"
	AcceptLanguage = "Accept-Language"

	// Downloads
	ContentDisposition = "Content-Disposition"

	// Errors
	Allow      = "Allow"
	RetryAfter = "Retry-After"

	// Proxies
	XForwardedFor   = "X-Forwarded-For"
	XForwardedHost  = "X-Forwarded-Host"
	XForwardedProto = "X-Forwarded-Proto"

	// Redirects
	Location = "Location"

	// Range requests
	AcceptRanges = "Accept-Ranges"
	Range        = "Range"
	IfRange      = "If-Range"
	ContentRange = "Content-Range"
)

// X-Content-Type-Options header and its value.
// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/X-Content-Type-Options
const (
	XContentTypeOptions = "X-Content-Type-Options"
	NoSniff             = "nosniff"
)

// Connection management headers.
const (
	Connection = "Connection"
	KeepAlive  = "Keep-Alive"

	// ConnectionClose is the value for the Connection header to indicate the
	// sender would like to close the connection.
	ConnectionClose = "close"
)
