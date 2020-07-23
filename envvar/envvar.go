// Copyright 2020 YourBase Inc.
// SPDX-License-Identifier: BSD-3-Clause

// Package envvar provides functions to read environment variables for
// configuration.
package envvar

import (
	"os"
	"strconv"
)

// Get returns the value of the given environment variable. If it is empty or
// unset, it returns the default value.
func Get(key string, defaultValue string) string {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue
	}
	return v
}

// Bool returns the value of a boolean environment variable. If it is unset or
// not one of the strings 1, t, T, TRUE, true, or True, then it returns false.
func Bool(key string) bool {
	v := os.Getenv(key)
	if v == "" {
		return false
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false
	}
	return b
}
