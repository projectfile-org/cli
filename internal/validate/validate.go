// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

// Package validate runs the projectfile v1 JSON Schema against a parsed
// projectfile document. The schema is embedded at build time so validation
// is offline; .scripts/fetch-schema.sh installs embedded/ from the specification.
package validate

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/dlclark/regexp2"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

//go:embed embedded/v1.json
var schemaBytes []byte

//go:embed embedded/v1-fragment.json
var fragmentSchemaBytes []byte

// SchemaURL is the canonical $id of the embedded schema; the compiler keys
// its resource cache on this URL.
const SchemaURL = "https://projectfile.org/schema/v1.json"

// FragmentSchemaURL is the $id of the schema for a document only ever included.
const FragmentSchemaURL = "https://projectfile.org/schema/v1-fragment.json"

// compileOnce guards the one-shot schema compile — the schemas are constant
// per binary, and re-parsing them on every Validate call would be wasteful.
var (
	compileOnce sync.Once
	compiled    map[string]*jsonschema.Schema
	compileErr  error
)

func loadSchema(url string) (*jsonschema.Schema, error) {
	compileOnce.Do(func() {
		c := jsonschema.NewCompiler()
		// JSON Schema specifies ECMA-262 regex semantics, which Go's stdlib
		// regexp (RE2) does not implement — RE2 rejects lookarounds and
		// backreferences. regexp2 is the ECMA-262-compatible engine.
		c.UseRegexpEngine(ecma262Regexp)
		resources := map[string][]byte{SchemaURL: schemaBytes, FragmentSchemaURL: fragmentSchemaBytes}
		for id, data := range resources {
			raw, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
			if err != nil {
				compileErr = fmt.Errorf("parse embedded schema %s: %w", id, err)
				return
			}
			if err := c.AddResource(id, raw); err != nil {
				compileErr = fmt.Errorf("register embedded schema %s: %w", id, err)
				return
			}
		}
		compiled = make(map[string]*jsonschema.Schema, len(resources))
		for id := range resources {
			s, err := c.Compile(id)
			if err != nil {
				compileErr = fmt.Errorf("compile embedded schema %s: %w", id, err)
				return
			}
			compiled[id] = s
		}
	})
	return compiled[url], compileErr
}

// Validate runs the projectfile v1 schema against doc. doc may come from any
// TOML/YAML/JSON parser; the validator expects values in the JSON data-model
// (string/float64/bool/nil/[]any/map[string]any), so we round-trip through
// encoding/json to normalise non-JSON natives (TOML int64, time.Time, YAML
// alias collapse).
func Validate(doc any) error {
	return validateAgainst(SchemaURL, doc)
}

// Fragment runs the fragment schema against doc, a document only ever reached through includes.
func Fragment(doc any) error {
	return validateAgainst(FragmentSchemaURL, doc)
}

func validateAgainst(url string, doc any) error {
	s, err := loadSchema(url)
	if err != nil {
		return err
	}
	norm, err := normalize(doc)
	if err != nil {
		return err
	}
	return s.Validate(norm)
}

func normalize(doc any) (any, error) {
	b, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("normalise document for validation: %w", err)
	}
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, fmt.Errorf("normalise document for validation: %w", err)
	}
	return v, nil
}

// ecma262Regexp adapts regexp2 to the jsonschema RegexpEngine interface.
// regexp2.ECMAScript matches the regex flavour JSON Schema mandates.
func ecma262Regexp(s string) (jsonschema.Regexp, error) {
	re, err := regexp2.Compile(s, regexp2.ECMAScript)
	if err != nil {
		return nil, err
	}
	return &ecmaRegexp{re: re, src: s}, nil
}

type ecmaRegexp struct {
	re  *regexp2.Regexp
	src string
}

func (r *ecmaRegexp) MatchString(s string) bool {
	m, _ := r.re.MatchString(s)
	return m
}

func (r *ecmaRegexp) String() string { return r.src }

// AsValidationError unwraps the underlying *jsonschema.ValidationError if
// err carries one. Callers use this to distinguish "the document is invalid"
// (which should pretty-print failures and exit 1) from "validation itself
// failed" (which is an internal error).
func AsValidationError(err error) (*jsonschema.ValidationError, bool) {
	var ve *jsonschema.ValidationError
	if errors.As(err, &ve) {
		return ve, true
	}
	return nil, false
}
