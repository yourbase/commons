// Copyright 2020 YourBase Inc.
// SPDX-License-Identifier: BSD-3-Clause

/*
Package ini provides a parser and serializer for the INI file format.
See https://en.wikipedia.org/wiki/INI_file.

This package is specifically designed for read-modify-write scenarios: it
preserves comments and edits existing values in-place.

This package can also parse .env files, since they are a subset of this
package's accepted syntax.

Syntax

An INI file is Unicode text encoded in UTF-8. The text is not canonicalized.

An INI file consists of zero or more properties. A property is a key and
value written on a single line, separated by an equals sign ('='):

	key=value

Keys are not allowed to contain semicolons (';'), contain equals signs ('='),
or start with a square bracket ('[' or ']'). Values may be surrounded by double
quotes ('"') to express values that begin or end with whitespace or to use
C-style escape sequences. Supported escape sequences:

	\n    U+000A line feed or newline
	\r    U+000D carriage return
	\t    U+0009 horizontal tab
	\\    U+005C backslash
	\"    U+0022 double quote
	\xFF  hex escape

Properties may be grouped into sections. A section is started by writing its
name in square brackets ('[' and ']') on its own line and ends at the next
section name or the end of file:

	[section]
	key1=value1
	key2=value2

Properties encountered before a section name are permitted. They are considered
part of the global section, identified by the empty string ("").

Whitespace (characters with the Unicode White Space property) at the
beginning or end of lines, around section names, around property keys, and
around property values are ignored. If the first non-whitespace character in
a line is a semicolon (';') or a hash ('#'), then the line is treated as a
comment. Inline comments are not supported.

Repeated names

Multiple properties in the same section may have the same key. When retrieving
the property in a single-value context (like using *File.Get), only the last
value will be used.

Multiple sections may have the same name. These are treated as if their
properties were presented contiguously in the same section.
*/
package ini
