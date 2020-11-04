// Copyright 2020 YourBase Inc.
// SPDX-License-Identifier: BSD-3-Clause

package ini

import (
	"encoding"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// Ensure File satisfies the encoding.Text* interfaces.
var _ interface {
	encoding.TextMarshaler
	encoding.TextUnmarshaler
} = new(File)

func TestNil(t *testing.T) {
	f := (*File)(nil)
	if got := f.Get("foo", "bar"); got != "" {
		t.Errorf("Get(...) = %q; want empty", got)
	}
	if got := f.Find("foo", "bar"); len(got) > 0 {
		t.Errorf("Find(...) = %q; want empty", got)
	}
	if got := f.Sections(); len(got) > 0 {
		t.Errorf("Sections(...) = %q; want empty", got)
	}
	if f.HasSections() {
		t.Error("HasSections() = true; want false")
	}
	if got := f.Section("foo"); len(got) > 0 {
		t.Errorf("Section(...) = %q; want empty", got)
	}
	if got, err := f.MarshalText(); err != nil {
		t.Errorf("MarshalText(): %v", err)
	} else if len(got) > 0 {
		t.Errorf("MarshalText() = %q; want empty", got)
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		options     *ParseOptions
		want        map[string]Section
		wantErr     bool
		canonical   string
		hasSections bool
	}{
		{
			name: "Empty",
		},
		{
			name:   "EmptyWithNewline",
			source: "\n",
		},
		{
			name:   "Single",
			source: "FOO=bar\n",
			want: map[string]Section{
				"": {
					"FOO": {"bar"},
				},
			},
			canonical: "FOO=bar\n",
		},
		{
			name:    "NoEquals",
			source:  "FOO\n",
			wantErr: true,
		},
		{
			name:   "SpaceSurroundingValue",
			source: "FOO= bar \n",
			want: map[string]Section{
				"": {
					"FOO": {"bar"},
				},
			},
			canonical: "FOO=bar\n",
		},
		{
			name:   "SpaceSurroundingKey",
			source: " FOO =bar\n",
			want: map[string]Section{
				"": {
					"FOO": {"bar"},
				},
			},
			canonical: "FOO=bar\n",
		},
		{
			name:   "SpaceSurroundingBoth",
			source: " FOO = bar \n",
			want: map[string]Section{
				"": {
					"FOO": {"bar"},
				},
			},
			canonical: "FOO=bar\n",
		},
		{
			name:   "NoNewline",
			source: "FOO=bar",
			want: map[string]Section{
				"": {
					"FOO": {"bar"},
				},
			},
			canonical: "FOO=bar\n",
		},
		{
			name:    "SemicolonKey",
			source:  "FOO;Bar=bar\n",
			wantErr: true,
		},
		{
			name:   "MultipleKeys",
			source: "FOO=bar\nBAZ=quux\n",
			want: map[string]Section{
				"": {
					"FOO": {"bar"},
					"BAZ": {"quux"},
				},
			},
			canonical: "FOO=bar\nBAZ=quux\n",
		},
		{
			name:   "MultipleValues",
			source: "FOO=bar\nFOO=baz\n",
			want: map[string]Section{
				"": {
					"FOO": {"bar", "baz"},
				},
			},
			canonical: "FOO=bar\nFOO=baz\n",
		},
		{
			name:   "BlankLine",
			source: "FOO=bar\n\nBAZ=quux\n",
			want: map[string]Section{
				"": {
					"FOO": {"bar"},
					"BAZ": {"quux"},
				},
			},
			canonical: "FOO=bar\nBAZ=quux\n",
		},
		{
			name:   "CRLF",
			source: "FOO=bar\r\n\r\nBAZ=quux\r\n",
			want: map[string]Section{
				"": {
					"FOO": {"bar"},
					"BAZ": {"quux"},
				},
			},
			canonical: "FOO=bar\nBAZ=quux\n",
		},
		{
			name:   "Section",
			source: "[foo]\nbar=baz\n",
			want: map[string]Section{
				"foo": {
					"bar": {"baz"},
				},
			},
			canonical:   "[foo]\nbar=baz\n",
			hasSections: true,
		},
		{
			name:    "MissingSectionName",
			source:  "[]\nbar=baz\n",
			wantErr: true,
		},
		{
			name:    "MissingSectionBracket",
			source:  "[foo\nbar=baz\n",
			wantErr: true,
		},
		{
			name:    "MismatchedSectionBracket",
			source:  "[foo]]\nbar=baz\n",
			wantErr: true,
		},
		{
			name:   "SectionWhitespace",
			source: "  [  foo  ] \nbar=baz\n",
			want: map[string]Section{
				"foo": {
					"bar": {"baz"},
				},
			},
			canonical:   "[foo]\nbar=baz\n",
			hasSections: true,
		},
		{
			name: "MultipleSections",
			source: "[foo]\nbar=baz\n" +
				"[python]\nspam=eggs\n",
			want: map[string]Section{
				"foo": {
					"bar": {"baz"},
				},
				"python": {
					"spam": {"eggs"},
				},
			},
			canonical: "[foo]\nbar=baz\n\n" +
				"[python]\nspam=eggs\n",
			hasSections: true,
		},
		{
			name:   "Comment",
			source: "; This explains everything!\nFOO=bar\n",
			want: map[string]Section{
				"": {
					"FOO": {"bar"},
				},
			},
			canonical: "; This explains everything!\nFOO=bar\n",
		},
		{
			name:   "HashComment",
			source: "# This explains everything!\nFOO=bar\n",
			want: map[string]Section{
				"": {
					"FOO": {"bar"},
				},
			},
			canonical: "# This explains everything!\nFOO=bar\n",
		},
		{
			name:   "EmptyComment",
			source: "; \t\nFOO=bar\n",
			want: map[string]Section{
				"": {
					"FOO": {"bar"},
				},
			},
			canonical: ";\nFOO=bar\n",
		},
		{
			name:      "OnlyComments",
			source:    "; This explains everything!\n# ... 42\n",
			canonical: "; This explains everything!\n# ... 42\n",
		},
		{
			name:   "SectionComment",
			source: "\n; This explains everything!\n\n[foo]\nbar=baz\n",
			want: map[string]Section{
				"foo": {
					"bar": {"baz"},
				},
			},
			canonical:   "; This explains everything!\n[foo]\nbar=baz\n",
			hasSections: true,
		},
		{
			name:   "CommentAtEndOfSection",
			source: "[foo]\nbar = baz\n; P.S.: You're awesome!\n",
			want: map[string]Section{
				"foo": {
					"bar": {"baz"},
				},
			},
			canonical:   "[foo]\nbar=baz\n\n; P.S.: You're awesome!\n",
			hasSections: true,
		},
		{
			name:   "NormalizeSection",
			source: "[foo]\nbar=baz\n",
			options: &ParseOptions{
				NormalizeSection: strings.ToUpper,
			},
			want: map[string]Section{
				"FOO": {
					"bar": {"baz"},
				},
			},
			canonical:   "[FOO]\nbar=baz\n",
			hasSections: true,
		},
		{
			name:   "NormalizeKey",
			source: "[foo]\nbar=baz\n",
			options: &ParseOptions{
				NormalizeKey: func(section, key string) string {
					return strings.ToUpper(key)
				},
			},
			want: map[string]Section{
				"foo": {
					"BAR": {"baz"},
				},
			},
			canonical:   "[foo]\nBAR=baz\n",
			hasSections: true,
		},
		{
			name:   "InnerQuote",
			source: `foo=bar"baz`,
			want: map[string]Section{
				"": {
					"foo": {`bar"baz`},
				},
			},
			canonical: `foo="bar\"baz"` + "\n",
		},
		{
			name:   "BackslashOutsideQuotes",
			source: `foo=\\` + "\nbar=baz\n",
			want: map[string]Section{
				"": {
					"foo": {`\\`},
					"bar": {`baz`},
				},
			},
			canonical: `foo=\\` + "\nbar=baz\n",
		},
		{
			name:   "Quoted",
			source: `foo = "hello world"` + "\n",
			want: map[string]Section{
				"": {
					"foo": {"hello world"},
				},
			},
			canonical: `foo=hello world` + "\n",
		},
		{
			name:   "Quoted/Escapes",
			source: `foo=" bar \" \\ blab\r\n\t\x00"` + "\n",
			want: map[string]Section{
				"": {
					"foo": {" bar \" \\ blab\r\n\t\x00"},
				},
			},
			canonical: `foo=" bar \" \\ blab\r\n\t\x00"` + "\n",
		},
		{
			name:   "Quoted/SurroundingSpaces",
			source: `foo= "  bar  "  `,
			want: map[string]Section{
				"": {
					"foo": {"  bar  "},
				},
			},
			canonical: `foo="  bar  "` + "\n",
		},
		{
			name:    "Quoted/SingleQuote",
			source:  "foo=\"\nbar=baz\"",
			wantErr: true,
		},
		{
			name:    "Quoted/Unterminated",
			source:  "foo=\"f\nbar=baz\"",
			wantErr: true,
		},
		{
			name:    "Quoted/UnterminatedEscape",
			source:  `foo="\"` + "\n",
			wantErr: true,
		},
		{
			name:    "Quoted/UnterminatedHexEscape",
			source:  `foo="\x0"` + "\n",
			wantErr: true,
		},
		{
			name:    "Quoted/UnknownEscape",
			source:  `foo="\y"` + "\n",
			wantErr: true,
		},
		{
			name:    "Quoted/BadHexDigits",
			source:  `foo="\xgG"` + "\n",
			wantErr: true,
		},
		{
			name:    "Quoted/TripleQuote",
			source:  `foo="""` + "\n",
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f, err := Parse(strings.NewReader(test.source), test.options)
			if err != nil {
				t.Logf("Parse: %v", err)
				if !test.wantErr {
					t.Fail()
				}
			} else if test.wantErr {
				t.Error("Parse did not return error")
			}

			t.Run("Sections", func(t *testing.T) {
				got := make(map[string]Section)
				for sectionName := range f.Sections() {
					got[sectionName] = f.Section(sectionName)
				}
				if diff := cmp.Diff(test.want, got, cmpopts.EquateEmpty()); diff != "" {
					t.Errorf("sections (-want +got):\n%s", diff)
				}
			})

			t.Run("HasSection", func(t *testing.T) {
				if got := f.HasSections(); got != test.hasSections {
					t.Errorf("f.HasSections() = %t; want %t", got, test.hasSections)
				}
			})

			t.Run("MarshalText", func(t *testing.T) {
				got, err := f.MarshalText()
				if err != nil {
					t.Fatal("MarshalText:", err)
				}
				if diff := cmp.Diff(test.canonical, string(got)); diff != "" {
					t.Errorf("MarshalText (-want +got):\n%s", diff)
				}
			})

			if test.source != test.canonical {
				t.Run("MarshalTextIdempotent", func(t *testing.T) {
					f, err := Parse(strings.NewReader(test.canonical), nil)
					if err != nil {
						t.Fatal("Parse:", err)
					}
					got, err := f.MarshalText()
					if err != nil {
						t.Fatal("MarshalText:", err)
					}
					if diff := cmp.Diff(test.canonical, string(got)); diff != "" {
						t.Errorf("MarshalText (-want +got):\n%s", diff)
					}
				})
			}
		})
	}
}

func TestAccess(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		section  string
		key      string
		wantGet  string
		wantFind []string
	}{
		{
			name:     "Global",
			source:   "FOO=bar\n",
			section:  "",
			key:      "FOO",
			wantGet:  "bar",
			wantFind: []string{"bar"},
		},
		{
			name:     "GlobalDoesNotExist",
			source:   "FOO=bar\n",
			section:  "",
			key:      "xyzzy",
			wantGet:  "",
			wantFind: []string{},
		},
		{
			name:     "MultipleValues",
			source:   "FOO=bar\nFOO=baz\n",
			section:  "",
			key:      "FOO",
			wantGet:  "baz",
			wantFind: []string{"bar", "baz"},
		},
		{
			name:     "Section",
			source:   "[foo]\nbar=baz\n",
			section:  "foo",
			key:      "bar",
			wantGet:  "baz",
			wantFind: []string{"baz"},
		},
		{
			name: "FirstSection",
			source: "[foo]\n" +
				"bar=baz\n" +
				"[xyzzy]\n" +
				"bork=bork\n" +
				"[foo]\n" +
				"something=else\n",
			section:  "foo",
			key:      "bar",
			wantGet:  "baz",
			wantFind: []string{"baz"},
		},
	}
	t.Run("Get", func(t *testing.T) {
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				f, err := Parse(strings.NewReader(test.source), nil)
				if err != nil {
					t.Fatal(err)
				}
				if got := f.Get(test.section, test.key); got != test.wantGet {
					t.Errorf("f.Get(%q, %q) = %q; want %q", test.section, test.key, got, test.wantGet)
				}
				if got := f.Section(test.section).Get(test.key); got != test.wantGet {
					t.Errorf("f.Section(%q).Get(%q) = %q; want %q", test.section, test.key, got, test.wantGet)
				}
			})
		}
	})
	t.Run("Find", func(t *testing.T) {
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				f, err := Parse(strings.NewReader(test.source), nil)
				if err != nil {
					t.Fatal(err)
				}
				got := f.Find(test.section, test.key)
				if diff := cmp.Diff(test.wantFind, got, cmpopts.EquateEmpty()); diff != "" {
					t.Errorf("f.Find(%q, %q) (-want +got):\n%s", test.section, test.key, diff)
				}
				got = f.Section(test.section)[test.key]
				if diff := cmp.Diff(test.wantFind, got, cmpopts.EquateEmpty()); diff != "" {
					t.Errorf("f.Section(%q)[%q] (-want +got):\n%s", test.section, test.key, diff)
				}
			})
		}
	})
}

func TestSet(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		section string
		key     string
		value   string
		want    string
	}{
		{
			name:    "AddToEmpty",
			section: "",
			key:     "foo",
			value:   "bar",
			want:    "foo=bar\n",
		},
		{
			name:    "AddSectionToEmpty",
			section: "foo",
			key:     "bar",
			value:   "baz",
			want:    "[foo]\nbar=baz\n",
		},
		{
			name:    "Overwrite",
			source:  "foo=bar\n",
			section: "",
			key:     "foo",
			value:   "xyzzy",
			want:    "foo=xyzzy\n",
		},
		{
			name:    "DeletePrevious",
			source:  "; Comment 1\nfoo=bar\n; Comment 2\nfoo=baz\n",
			section: "",
			key:     "foo",
			value:   "quux",
			want:    "; Comment 2\nfoo=quux\n",
		},
		{
			name:    "AddToExistingSection",
			source:  "foo=bar\n",
			section: "",
			key:     "baz",
			value:   "quux",
			want:    "foo=bar\nbaz=quux\n",
		},
		{
			name:    "AddGlobalSection",
			source:  "[foo]\nbar=baz\n",
			section: "",
			key:     "global",
			value:   "world",
			want:    "global=world\n\n[foo]\nbar=baz\n",
		},
		{
			name:    "AddNewSection",
			source:  "[foo]\nbar=baz\n",
			section: "python",
			key:     "spam",
			value:   "eggs",
			want:    "[foo]\nbar=baz\n\n[python]\nspam=eggs\n",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := new(File)
			if test.source != "" {
				var err error
				f, err = Parse(strings.NewReader(test.source), nil)
				if err != nil {
					t.Fatal(err)
				}
			}
			f.Set(test.section, test.key, test.value)
			got, err := f.MarshalText()
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, string(got)); diff != "" {
				t.Errorf("MarshalText (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAdd(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		section string
		key     string
		values  []string
		want    string
	}{
		{
			name:    "AddNothing",
			section: "",
			key:     "foo",
			values:  []string{},
			want:    "",
		},
		{
			name:    "AddToEmpty",
			section: "",
			key:     "foo",
			values:  []string{"bar"},
			want:    "foo=bar\n",
		},
		{
			name:    "AddMultiple",
			section: "",
			key:     "foo",
			values:  []string{"bar", "baz"},
			want:    "foo=bar\nfoo=baz\n",
		},
		{
			name:    "AddSectionToEmpty",
			section: "foo",
			key:     "bar",
			values:  []string{"baz"},
			want:    "[foo]\nbar=baz\n",
		},
		{
			name:    "RetainPrevious",
			source:  "; Comment 1\nfoo=bar\n; Comment 2\nfoo=baz\n",
			section: "",
			key:     "foo",
			values:  []string{"quux"},
			want:    "; Comment 1\nfoo=bar\n; Comment 2\nfoo=baz\nfoo=quux\n",
		},
		{
			name:    "AddToExistingSection",
			source:  "foo=bar\n",
			section: "",
			key:     "baz",
			values:  []string{"quux"},
			want:    "foo=bar\nbaz=quux\n",
		},
		{
			name:    "AddGlobalSection",
			source:  "[foo]\nbar=baz\n",
			section: "",
			key:     "global",
			values:  []string{"world"},
			want:    "global=world\n\n[foo]\nbar=baz\n",
		},
		{
			name:    "AddNewSection",
			source:  "[foo]\nbar=baz\n",
			section: "python",
			key:     "spam",
			values:  []string{"eggs"},
			want:    "[foo]\nbar=baz\n\n[python]\nspam=eggs\n",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := new(File)
			if test.source != "" {
				var err error
				f, err = Parse(strings.NewReader(test.source), nil)
				if err != nil {
					t.Fatal(err)
				}
			}
			f.Add(test.section, test.key, test.values)
			got, err := f.MarshalText()
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, string(got)); diff != "" {
				t.Errorf("MarshalText (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		section string
		key     string
		want    string
	}{
		{
			name:    "Empty",
			section: "",
			key:     "foo",
			want:    "",
		},
		{
			name:    "Global",
			source:  "junk1=\nfoo=bar\njunk2=\n",
			section: "",
			key:     "foo",
			want:    "junk1=\njunk2=\n",
		},
		{
			name:    "EmptyGlobal",
			source:  "foo=bar\n",
			section: "",
			key:     "foo",
			want:    "",
		},
		{
			name:    "MultipleGlobal",
			source:  "junk=\nfoo=bar\nfoo=baz\n",
			section: "",
			key:     "foo",
			want:    "junk=\n",
		},
		{
			name:    "Section",
			source:  "[group]\njunk1=\nfoo=bar\njunk2=\n",
			section: "group",
			key:     "foo",
			want:    "[group]\njunk1=\njunk2=\n",
		},
		{
			name:    "EmptySection",
			source:  "[group]\nfoo=bar\n",
			section: "group",
			key:     "foo",
			want:    "",
		},
		{
			name:    "MultipleInSection",
			source:  "[group]\njunk=\nfoo=bar\nfoo=baz\n",
			section: "group",
			key:     "foo",
			want:    "[group]\njunk=\n",
		},
		{
			name: "MultipleAcrossSections",
			source: "[group]\njunk=\nfoo=bar\n" +
				"[other]\nfoo=other\n" +
				"[group]\nfoo=baz\n",
			section: "group",
			key:     "foo",
			want:    "[group]\njunk=\n\n[other]\nfoo=other\n",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := new(File)
			if test.source != "" {
				var err error
				f, err = Parse(strings.NewReader(test.source), nil)
				if err != nil {
					t.Fatal(err)
				}
			}
			f.Delete(test.section, test.key)
			got, err := f.MarshalText()
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, string(got)); diff != "" {
				t.Errorf("MarshalText (-want +got):\n%s", diff)
			}
		})
	}
}

func TestIsValidSection(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"", true},
		{" ", false},
		{"\t", false},
		{"foo", true},
		{"foo bar", true},
		{" foo ", false},
		{"[foo", false},
		{"]foo", false},
		{"foo[bar", false},
		{"foo]bar", false},
	}
	for _, test := range tests {
		if got := IsValidSection(test.name); got != test.want {
			t.Errorf("IsValidSection(%q) = %t; want %t", test.name, got, test.want)
		}
	}
}

func TestIsValidKey(t *testing.T) {
	tests := []struct {
		key  string
		want bool
	}{
		{"", false},
		{" ", false},
		{"\t", false},
		{"foo", true},
		{"foo bar", true},
		{" foo ", false},
		{";foo", false},
		{"#foo", false},
		{"=foo", false},
		{"[foo", false},
		{"]foo", false},
		{"foo;bar", false},
		{"foo#bar", false},
		{"foo=bar", false},
		{"foo[bar", true},
		{"foo]bar", true},
	}
	for _, test := range tests {
		if got := IsValidKey(test.key); got != test.want {
			t.Errorf("IsValidKey(%q) = %t; want %t", test.key, got, test.want)
		}
	}
}
