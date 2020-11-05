// Copyright 2020 YourBase Inc.
// SPDX-License-Identifier: BSD-3-Clause

package ini_test

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/yourbase/commons/ini"
)

func ExampleParse() {
	const iniFile = `
		global = xyzzy
		[foo]
		bar = baz
		[mysection]
		host = example.com`
	cfg, err := ini.Parse(strings.NewReader(iniFile), nil)
	if err != nil {
		// handle error
	}

	// Print out sorted section names.
	var sections []string
	for name := range cfg.Sections() {
		sections = append(sections, name)
	}
	sort.Strings(sections)
	fmt.Printf("Sections: %q\n", sections)

	// Get specific values.
	fmt.Println("Global property:", cfg.Get("", "global"))
	fmt.Println("Property in section:", cfg.Get("foo", "bar"))

	// Output:
	// Sections: ["" "foo" "mysection"]
	// Global property: xyzzy
	// Property in section: baz
}

// This package can parse .env files, since they are a subset of this package's
// accepted syntax.
func ExampleParse_env() {
	const dotEnvFile = `
	FOO=bar
	# Comments!
	XYZZY=magic`
	cfg, err := ini.Parse(strings.NewReader(dotEnvFile), nil)
	if err != nil {
		// handle error
	}
	// Validate that the file does not contain sections.
	if cfg.HasSections() {
		// handle error
	}

	// Print out the environment variables one per line, like env output.
	varMap := cfg.Section("")
	envVars := make([]string, 0, len(varMap))
	for k := range varMap {
		envVars = append(envVars, k+"="+varMap.Get(k))
	}
	sort.Strings(envVars)
	fmt.Println(strings.Join(envVars, "\n"))

	// Output:
	// FOO=bar
	// XYZZY=magic
}

// Setting NormalizeSection and NormalizeKey options allow you to change how
// section names and property keys are interpreted.
//
// In this example, we are lowercasing all section names and keys.
func ExampleParse_caseInsensitive() {
	const iniFile = `
		[FOO]
		bar = first
		; This property key will be lowercased, overriding the first value.
		BAR = BAZ`
	cfg, err := ini.Parse(strings.NewReader(iniFile), &ini.ParseOptions{
		NormalizeSection: func(section string) string {
			return strings.ToLower(section)
		},
		NormalizeKey: func(section, key string) string {
			return strings.ToLower(key)
		},
	})
	if err != nil {
		// handle error
	}

	// Now we can access the property with the lowercased section and key.
	fmt.Println(cfg.Get("foo", "bar"))

	// Output:
	// BAZ
}

func ExampleFile_Get() {
	cfg, err := ini.Parse(strings.NewReader("foo = bar\n"), nil)
	if err != nil {
		// handle error
	}
	fmt.Println(cfg.Get("", "foo"))

	// Output:
	// bar
}

func ExampleFile_Get_fromSection() {
	cfg, err := ini.Parse(strings.NewReader(`
		foo = bar
		[baz]
		foo = quux
	`), nil)
	if err != nil {
		// handle error
	}
	fmt.Println(cfg.Get("baz", "foo"))

	// Output:
	// quux
}

func ExampleFile_MarshalText() {
	// Using new(ini.File) creates an empty File.
	// You can also modify an existing File from Parse.
	f := new(ini.File)

	// Use File.Set to populate values.
	f.Set("", "foo", "bar")
	f.Set("mysection", "host", "example.com")

	// Marshal to INI format and write to a file.
	text, err := f.MarshalText()
	if err != nil {
		// handle error
	}
	if _, err := os.Stdout.Write(text); err != nil {
		// handle error
	}

	// Output:
	// foo=bar
	//
	// [mysection]
	// host=example.com
}
