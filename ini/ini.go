// Copyright 2020 YourBase Inc.
// SPDX-License-Identifier: BSD-3-Clause

package ini

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"
)

// A File is a collection of properties. The zero value is an empty file.
// Files can be read by multiple concurrent goroutines.
type File struct {
	sections         []section
	trailingComments []string
}

type section struct {
	name       string
	comments   []string
	properties []property
}

type property struct {
	comments []string
	key      string
	value    string
}

// ParseOptions holds optional parameters for Parse.
type ParseOptions struct {
	// NormalizeSection is called on each section name to apply text transformations.
	// This can be used to make keys case-insensitive, for instance.
	// If nil, no transformations are made.
	NormalizeSection func(name string) string

	// NormalizeKey is called on each key to apply text transformations.
	// This can be used to make keys case-insensitive, for instance.
	// If nil, no transformations are made.
	NormalizeKey func(section, key string) string
}

// Parse parses an INI file. Nil options are treated identically as passing the
// zero value.
//
// See the Syntax section in the package documentation for the format recognized
// by Parse.
func Parse(r io.Reader, opts *ParseOptions) (*File, error) {
	s := bufio.NewScanner(r)
	f := &File{
		sections: []section{
			{name: ""}, // Always start with the default section.
		},
	}
	lineno := 1
	var comments []string
	for ; s.Scan(); lineno++ {
		line, err := cleanLine(s.Bytes())
		if err != nil {
			return f, fmt.Errorf("parse ini file: line %d: %w", lineno, err)
		}
		if line == "" {
			continue
		}
		switch line[0] {
		case ';', '#':
			comments = append(comments, line)
		case '[':
			name := line[1 : len(line)-1]
			if opts != nil && opts.NormalizeSection != nil {
				name = opts.NormalizeSection(name)
			}
			f.sections = append(f.sections, section{
				name:     name,
				comments: comments,
			})
			comments = nil
		default:
			currSection := &f.sections[len(f.sections)-1]
			i := strings.IndexByte(line, '=')
			key := line[:i]
			if !IsValidKey(key) {
				return f, fmt.Errorf("parse ini file: line %d: invalid key %q", lineno, key)
			}
			if opts != nil && opts.NormalizeKey != nil {
				key = opts.NormalizeKey(currSection.name, key)
			}
			currSection.properties = append(currSection.properties, property{
				comments: comments,
				key:      key,
				value:    unquote(line[i+1:]),
			})
			comments = nil
		}
	}
	if err := s.Err(); err != nil {
		return f, fmt.Errorf("parse ini file: line %d: %w", lineno, err)
	}
	f.trailingComments = comments
	return f, nil
}

func unquote(v string) string {
	if !strings.HasPrefix(v, `"`) {
		return v
	}
	v = v[1 : len(v)-1]
	sb := new(strings.Builder)
	sb.Grow(len(v))
	for i := 0; i < len(v); i++ {
		if v[i] != '\\' {
			sb.WriteByte(v[i])
			continue
		}
		i++
		switch v[i] {
		case 'r':
			sb.WriteByte('\r')
		case 'n':
			sb.WriteByte('\n')
		case 't':
			sb.WriteByte('\t')
		case 'x':
			sb.WriteByte(fromHex(v[i+1])<<4 | fromHex(v[i+2]))
			i += 2
		case '"', '\\':
			sb.WriteByte(v[i])
		default:
			panic("unreachable")
		}
	}
	return sb.String()
}

func cleanLine(line []byte) (string, error) {
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return "", nil
	}
	if line[0] == '#' || line[0] == ';' {
		// Comment
		val := new(strings.Builder)
		val.Grow(len(line))
		val.WriteByte(line[0])
		if comment := bytes.TrimSpace(line[1:]); len(comment) > 0 {
			val.WriteByte(' ')
			val.Write(comment)
		}
		return val.String(), nil
	}
	if line[0] == '[' {
		// Section name
		if line[len(line)-1] != ']' {
			return "", errors.New("missing section closing bracket")
		}
		name := bytes.TrimSpace(line[1 : len(line)-1])
		if len(name) == 0 {
			return "", errors.New("section name missing")
		}
		if bytes.ContainsAny(name, "[]") {
			return "", errors.New("unexpected brackets in section name")
		}
		return "[" + string(name) + "]", nil
	}
	// Property
	i := bytes.IndexByte(line, '=')
	if i == -1 {
		return "", errors.New("could not find '='")
	}
	k := bytes.TrimRightFunc(line[:i], unicode.IsSpace)
	v := bytes.TrimLeftFunc(line[i+1:], unicode.IsSpace)
	if bytes.HasPrefix(v, []byte{'"'}) {
		if err := validateQuotedString(v); err != nil {
			return "", err
		}
	}
	sb := new(strings.Builder)
	sb.Grow(len(k) + 1 + len(v))
	sb.Write(k)
	sb.WriteByte('=')
	sb.Write(v)
	return sb.String(), nil
}

func validateQuotedString(v []byte) error {
	if len(v) < 2 {
		return errors.New("unterminated string")
	}
	endsInQuote := bytes.HasSuffix(v, []byte{'"'})
	v = v[1 : len(v)-1]
	for i := 0; i < len(v); i++ {
		if v[i] == '"' {
			return errors.New("trailing characters after string")
		}
		if v[i] != '\\' {
			continue
		}
		if i+1 >= len(v) {
			return errors.New("unexpected end of string")
		}
		switch v[i+1] {
		case 'n', 'r', 't', '\\', '"':
			i++
		case 'x':
			if i+3 >= len(v) {
				return errors.New("unexpected end of string")
			}
			if !isHexDigit(v[i+2]) || !isHexDigit(v[i+3]) {
				return fmt.Errorf("bad hex escape %s", v[i:i+4])
			}
			i += 3
		default:
			return fmt.Errorf("unknown escape %q", v[i+1])
		}
	}
	if !endsInQuote {
		return errors.New("unterminated string")
	}
	return nil
}

func isHexDigit(c byte) bool {
	return '0' <= c && c <= '9' ||
		'a' <= c && c <= 'f' ||
		'A' <= c && c <= 'F'
}

func fromHex(c byte) byte {
	switch {
	case '0' <= c && c <= '9':
		return c - '0'
	case 'a' <= c && c <= 'f':
		return c - 'a' + 0xa
	case 'A' <= c && c <= 'F':
		return c - 'A' + 0xa
	default:
		panic("invalid hex digit")
	}
}

// Get returns the last value associated with the given key in the given
// section. Passing an empty section name searches for properties outside
// any section. If there are no values associated with the key, Get returns
// the empty string.
func (f *File) Get(section, key string) string {
	v, _ := f.get(section, key)
	return v
}

func (f *File) get(section, key string) (_ string, ok bool) {
	if f == nil {
		return "", false
	}
	for i := len(f.sections) - 1; i >= 0; i-- {
		currSection := &f.sections[i]
		if currSection.name != section {
			continue
		}
		for j := len(currSection.properties) - 1; j >= 0; j-- {
			currProperty := &currSection.properties[j]
			if currProperty.key == key {
				return currProperty.value, true
			}
		}
	}
	return "", false
}

// Find returns all the values associated with the given key in the given
// section. Passing an empty section name searches for properties outside
// any section.
func (f *File) Find(section, key string) []string {
	if f == nil {
		return nil
	}
	var values []string
	for _, s := range f.sections {
		if s.name != section {
			continue
		}
		for _, p := range s.properties {
			if p.key == key {
				values = append(values, p.value)
			}
		}
	}
	return values
}

// Sections returns the names of sections in a file that have properties set.
// This will include the empty string if there are properties set outside
// a section.
func (f *File) Sections() map[string]struct{} {
	if f == nil {
		return nil
	}
	names := make(map[string]struct{}, len(f.sections))
	for _, s := range f.sections {
		if len(s.properties) > 0 {
			names[s.name] = struct{}{}
		}
	}
	return names
}

// HasSections reports whether f has any sections with properties set other than
// the unnamed global section.
func (f *File) HasSections() bool {
	if f == nil {
		return false
	}
	for _, s := range f.sections {
		if s.name != "" && len(s.properties) > 0 {
			return true
		}
	}
	return false
}

// Section returns a copy of the properties in the named section.
// Section("") returns the global section: the properties set outside any
// section.
func (f *File) Section(name string) Section {
	if f == nil {
		return nil
	}
	var result Section
	for _, s := range f.sections {
		if s.name != name {
			continue
		}
		for _, prop := range s.properties {
			if result == nil {
				result = make(Section)
			}
			result[prop.key] = append(result[prop.key], prop.value)
		}
	}
	return result
}

// Set sets the property to the given value. If the section name is empty, the
// property is set outside any section. Set will panic if
// IsValidSection(sectionName) or IsValidKey(key) report false.
//
// If the file already had at least one property for in given section with the
// given key, then the last one will be set to value and the properties defined
// earlier in the file will be removed. Otherwise, the property will be appended
// to the appropriate section, creating a section at the end of the file if
// necessary.
func (f *File) Set(sectionName, key, value string) {
	if !IsValidSection(sectionName) {
		panic("File.Set invalid section: " + sectionName)
	}
	if !IsValidKey(key) {
		panic("File.Set invalid key: " + key)
	}
	var addToSection *section
	wrote := false
	for i := len(f.sections) - 1; i >= 0; i-- {
		currSection := &f.sections[i]
		if currSection.name != sectionName {
			continue
		}
		if addToSection == nil {
			addToSection = currSection
		}
		for j := len(currSection.properties) - 1; j >= 0; j-- {
			prop := &currSection.properties[j]
			if prop.key != key {
				continue
			}
			if wrote {
				// Delete any previous properties with the same section/key.
				copy(currSection.properties[j:], currSection.properties[j+1:])
				// Zero out truncated element for garbage collection.
				currSection.properties[len(currSection.properties)-1] = property{}
				currSection.properties = currSection.properties[:len(currSection.properties)-1]
			} else {
				prop.value = value
				wrote = true
			}
		}
	}
	if wrote {
		return
	}
	if addToSection == nil {
		if sectionName == "" {
			// Global section must be first.
			f.sections = append(f.sections, section{})
			copy(f.sections[1:], f.sections)
			f.sections[0] = section{}
			addToSection = &f.sections[0]
		} else {
			// Add new section to end of file.
			f.sections = append(f.sections, section{name: sectionName})
			addToSection = &f.sections[len(f.sections)-1]
		}
	}
	addToSection.properties = append(addToSection.properties, property{
		key:   key,
		value: value,
	})
}

// Delete deletes any property with the given key in sections with the
// given name. If this causes any sections that do not have comments attached to
// become empty, then those sections will be removed.
func (f *File) Delete(sectionName, key string) {
	sectionCount := 0
	for i := range f.sections {
		s := &f.sections[i]
		if s.name != sectionName {
			f.sections[sectionCount] = *s
			sectionCount++
			continue
		}

		origPropertyCount := len(s.properties)
		propertyCount := 0
		for j := range s.properties {
			if s.properties[j].key != key {
				s.properties[propertyCount] = s.properties[j]
				propertyCount++
			}
		}
		for j := propertyCount; j < len(s.properties); j++ {
			// Zero out for garbage collection.
			s.properties[j] = property{}
		}
		s.properties = s.properties[:propertyCount]

		// Keep the section if it still has properties or comments, or we didn't
		// modify it. Always keep the global section to avoid shuffle later.
		if sectionName == "" || propertyCount > 0 || origPropertyCount == 0 || len(s.comments) > 0 {
			f.sections[sectionCount] = *s
			sectionCount++
		}
	}
	for i := sectionCount; i < len(f.sections); i++ {
		// Zero out for garbage collection.
		f.sections[i] = section{}
	}
	f.sections = f.sections[:sectionCount]
}

// Add appends properties with the given key under the given section. If the
// section name is empty, the property are appended to the global section.
// Add will panic if IsValidSection(sectionName) or IsValidKey(key) report false.
// If there is no section with the given name, one will be created at the end of
// the file.
func (f *File) Add(sectionName, key string, values []string) {
	if !IsValidSection(sectionName) {
		panic("File.Add invalid section: " + sectionName)
	}
	if !IsValidKey(key) {
		panic("File.Add invalid key: " + key)
	}
	if len(values) == 0 {
		return
	}
	var addToSection *section
	for i := len(f.sections) - 1; i >= 0; i-- {
		currSection := &f.sections[i]
		if currSection.name == sectionName {
			addToSection = currSection
			break
		}
	}
	if addToSection == nil {
		if sectionName == "" {
			// Global section must be first.
			f.sections = append(f.sections, section{})
			copy(f.sections[1:], f.sections)
			f.sections[0] = section{}
			addToSection = &f.sections[0]
		} else {
			// Add new section to end of file.
			f.sections = append(f.sections, section{name: sectionName})
			addToSection = &f.sections[len(f.sections)-1]
		}
	}
	for _, value := range values {
		addToSection.properties = append(addToSection.properties, property{
			key:   key,
			value: value,
		})
	}
}

// MarshalText serializes the file in INI format, including comments from the
// original file.
func (f *File) MarshalText() ([]byte, error) {
	if f == nil {
		return nil, nil
	}
	var buf []byte
	for _, s := range f.sections {
		if s.name != "" && len(buf) > 0 {
			buf = append(buf, '\n')
		}
		for _, comment := range s.comments {
			buf = append(buf, comment...)
			buf = append(buf, '\n')
		}
		if s.name != "" {
			buf = append(buf, '[')
			buf = append(buf, s.name...)
			buf = append(buf, "]\n"...)
		}
		for _, prop := range s.properties {
			for _, comment := range prop.comments {
				buf = append(buf, comment...)
				buf = append(buf, '\n')
			}
			buf = append(buf, prop.key...)
			buf = append(buf, '=')
			if shouldQuoteValue(prop.value) {
				buf = appendQuotedString(buf, prop.value)
			} else {
				buf = append(buf, prop.value...)
			}
			buf = append(buf, '\n')
		}
	}
	if len(f.trailingComments) > 0 && len(buf) > 0 {
		buf = append(buf, '\n')
	}
	for _, comment := range f.trailingComments {
		buf = append(buf, comment...)
		buf = append(buf, '\n')
	}
	return buf, nil
}

func appendQuotedString(dst []byte, v string) []byte {
	dst = append(dst, '"')
	for i := 0; i < len(v); i++ {
		switch c := v[i]; {
		case c == '\n':
			dst = append(dst, '\\', 'n')
		case c == '\r':
			dst = append(dst, '\\', 'r')
		case c == '\t':
			dst = append(dst, '\\', 't')
		case c == '\\':
			dst = append(dst, '\\', '\\')
		case c == '"':
			dst = append(dst, '\\', '"')
		case c < ' ' || c == del:
			const hexDigits = "0123456789abcdef"
			dst = append(dst, '\\', 'x', hexDigits[c>>4], hexDigits[c&0xf])
		default:
			dst = append(dst, c)
		}
	}
	dst = append(dst, '"')
	return dst
}

const del = '\x7f'

func shouldQuoteValue(v string) bool {
	if strings.TrimSpace(v) != v {
		return true
	}
	for _, c := range v {
		if c == '"' || (c < ' ' || c == del) {
			return true
		}
	}
	return false
}

// UnmarshalText parses the INI data with default options, replacing any
// properties or sections in f.
func (f *File) UnmarshalText(data []byte) error {
	parsed, err := Parse(bytes.NewReader(data), nil)
	if err != nil {
		return err
	}
	*f = *parsed
	return nil
}

// A Section is a map of string keys to a list of values.
type Section map[string][]string

// Get returns the last value associated with the given key. If there are no
// values associated with the key, Get returns the empty string.
func (sect Section) Get(key string) string {
	values := sect[key]
	if len(values) == 0 {
		return ""
	}
	return values[len(values)-1]
}

// IsValidSection reports whether a string can be used as a section name in
// an INI file.
func IsValidSection(name string) bool {
	if name == "" {
		// Special case: global section.
		return true
	}
	first, _ := utf8.DecodeRuneInString(name)
	last, _ := utf8.DecodeLastRuneInString(name)
	if unicode.IsSpace(first) || unicode.IsSpace(last) {
		return false
	}
	return !strings.ContainsAny(name, "[]")
}

// IsValidKey reports whether a string can be used as a property key in
// an INI file.
func IsValidKey(key string) bool {
	if key == "" {
		return false
	}
	first, _ := utf8.DecodeRuneInString(key)
	last, _ := utf8.DecodeLastRuneInString(key)
	if unicode.IsSpace(first) || unicode.IsSpace(last) {
		return false
	}
	if first == '[' || first == ']' {
		return false
	}
	return !strings.ContainsAny(key, ";=#")
}
