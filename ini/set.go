// Copyright 2020 YourBase Inc.
// SPDX-License-Identifier: BSD-3-Clause

package ini

import (
	"fmt"
	"os"
)

// FileSet is a list of files to obtain configuration from in descending order
// of precedence.
type FileSet []*File

// ParseFiles parses the files at the given paths as INI and returns a FileSet.
// If the returned error is nil, the returned file set's length will be the same
// as the number of arguments. ParseFiles will stop on the first error, but
// ignores missing file errors, instead filling the corresponding element of the
// set with a nil *File.
func ParseFiles(opts *ParseOptions, paths ...string) (FileSet, error) {
	fset := make(FileSet, 0, len(paths))
	for _, p := range paths {
		f, err := os.Open(p)
		if os.IsNotExist(err) {
			fset = append(fset, nil)
			continue
		}
		if err != nil {
			return fset, fmt.Errorf("parse ini files: %w", err)
		}
		parsed, err := Parse(f, opts)
		f.Close() // Close errors irrelevant.
		if err != nil {
			return fset, fmt.Errorf("parse ini files: %s: %w", p, err)
		}
		fset = append(fset, parsed)
	}
	return fset, nil
}

// Get returns the last value associated with the given key in the given
// section. Passing an empty section name searches for properties outside
// any section. If there are no values associated with the key, Get returns
// the empty string.
func (fset FileSet) Get(section, key string) string {
	for _, f := range fset {
		if v, ok := f.get(section, key); ok {
			return v
		}
	}
	return ""
}

// Find returns all the values associated with the given key in the given
// section. Passing an empty section name searches for properties outside
// any section.
func (fset FileSet) Find(section, key string) []string {
	var values []string
	for i := len(fset) - 1; i >= 0; i-- {
		values = append(values, fset[i].Find(section, key)...)
	}
	return values
}

// Sections returns the names of sections that have properties set in any file.
// This will include the empty string if there are properties set outside
// sections.
func (fset FileSet) Sections() map[string]struct{} {
	merged := make(map[string]struct{})
	for _, f := range fset {
		for name := range f.Sections() {
			merged[name] = struct{}{}
		}
	}
	return merged
}

// HasSections reports whether f has any sections with properties set other than
// the unnamed global section.
func (fset FileSet) HasSections() bool {
	for _, f := range fset {
		if f.HasSections() {
			return true
		}
	}
	return false
}

// Section returns a copy of the properties in the named section.
// Section("") returns the global section: the properties set outside any
// section.
func (fset FileSet) Section(name string) Section {
	merged := make(Section)
	for i := len(fset) - 1; i >= 0; i-- {
		for name, values := range fset[i].Section(name) {
			merged[name] = append(merged[name], values...)
		}
	}
	return merged
}

// Set sets the property on the first file and deletes the property in all
// subsequent files. Set will panic if len(fset) == 0, IsValidSection(sectionName)
// reports false, or IsValidKey(key) reports false.
//
// If fset[0] == nil, Set allocates a new File. Any other nil files in the set
// will be ignored.
func (fset FileSet) Set(sectionName, key, value string) {
	if fset[0] == nil {
		fset[0] = new(File)
	}
	fset[0].Set(sectionName, key, value)
	fset[1:].Delete(sectionName, key)
}

// Delete deletes any property with the given key in sections with the given
// name. If this causes any sections that do not have comments attached to
// become empty, then those sections will be removed. Nil elements of the set
// are ignored.
func (fset FileSet) Delete(sectionName, key string) {
	for _, f := range fset {
		if f != nil {
			f.Delete(sectionName, key)
		}
	}
}

// Add appends property with the given key under the given section to the first
// file. If the section name is empty, the property are appended to the global
// section. Add will panic if len(fset) == 0, IsValidSection(sectionName)
// reports false, or IsValidKey(key) reports false. If fset[0] == nil, it
// allocates a new File.
func (fset FileSet) Add(sectionName, key string, values []string) {
	if fset[0] == nil {
		fset[0] = new(File)
	}
	fset[0].Add(sectionName, key, values)
}
