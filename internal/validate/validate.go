// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

// Package validate runs the projectfile v1 JSON Schema against a parsed
// projectfile document. The schema is embedded at build time so validation
// is offline; .scripts/fetch-schema.sh installs embedded/ from the specification.
package validate

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strings"
	"sync"

	"github.com/dlclark/regexp2"
	"github.com/santhosh-tekuri/jsonschema/v6"

	"kiota.ch/projectfile/core/v2/pkg/genlog"
)

//go:embed embedded/v1.json
var schemaBytes []byte

//go:embed embedded/v1-fragment.json
var fragmentSchemaBytes []byte

//go:embed embedded/org.projectfile.*.v1.json
var namespaceFS embed.FS // each standalone extension schema, named after the subtree it checks

// SchemaURL is the canonical $id of the embedded schema; the compiler keys
// its resource cache on this URL.
const SchemaURL = "https://projectfile.org/schema/v1.json"

// FragmentSchemaURL is the $id of the schema for a document only ever included.
const FragmentSchemaURL = "https://projectfile.org/schema/v1-fragment.json"

// schemaBase is the URL every embedded schema's $id sits under.
const schemaBase = "https://projectfile.org/schema/"

// NamespacesURL keys the generated schema that applies each standalone schema at its own subtree.
const NamespacesURL = "urn:projectfile:cli:namespaces"

// compileOnce guards the one-shot schema compile — the schemas are constant
// per binary, and re-parsing them on every Validate call would be wasteful.
var (
	compileOnce sync.Once
	compiled    map[string]*jsonschema.Schema
	compileErr  error
)

func loadSchema(url string) (*jsonschema.Schema, error) {
	compileOnce.Do(func() {
		compiled, compileErr = compileSchemas(schemaBytes, fragmentSchemaBytes)
	})
	return compiled[url], compileErr
}

// compileSchemas parses and compiles both embedded schemas; it is split out from the cache so a corrupt embed is pinnable without the process-wide once.
func compileSchemas(schema, fragment []byte) (map[string]*jsonschema.Schema, error) {
	c := jsonschema.NewCompiler()
	// JSON Schema specifies ECMA-262 regex semantics, which Go's stdlib
	// regexp (RE2) does not implement — RE2 rejects lookarounds and
	// backreferences. regexp2 is the ECMA-262-compatible engine.
	c.UseRegexpEngine(ecma262Regexp)
	resources := map[string][]byte{SchemaURL: schema, FragmentSchemaURL: fragment}
	wrapper, err := namespaceResources(resources)
	if err != nil {
		return nil, err
	}
	resources[NamespacesURL] = wrapper
	for id, data := range resources {
		raw, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("parse embedded schema %s: %w", id, err)
		}
		if err := c.AddResource(id, raw); err != nil {
			return nil, fmt.Errorf("register embedded schema %s: %w", id, err)
		}
	}
	out := make(map[string]*jsonschema.Schema, len(resources))
	for id := range resources {
		s, err := c.Compile(id)
		if err != nil {
			return nil, fmt.Errorf("compile embedded schema %s: %w", id, err)
		}
		out[id] = s
	}
	return out, nil
}

// namespaceResources adds every embedded extension schema to resources and returns the wrapper placing each at its subtree.
func namespaceResources(resources map[string][]byte) ([]byte, error) {
	files, err := namespaceFS.ReadDir("embedded")
	if err != nil {
		return nil, fmt.Errorf("list embedded namespace schemas: %w", err)
	}
	root := map[string]any{}
	for _, f := range files {
		name := f.Name()
		if !strings.HasSuffix(name, ".v1.json") {
			continue
		}
		data, err := namespaceFS.ReadFile(path.Join("embedded", name))
		if err != nil {
			return nil, fmt.Errorf("read embedded schema %s: %w", name, err)
		}
		resources[schemaBase+name] = data
		node := root
		for _, segment := range strings.Split(strings.TrimSuffix(name, ".v1.json"), ".") {
			props, _ := node["properties"].(map[string]any)
			if props == nil {
				props = map[string]any{}
				node["properties"] = props
			}
			next, _ := props[segment].(map[string]any)
			if next == nil {
				next = map[string]any{}
				props[segment] = next
			}
			node = next
		}
		node["$ref"] = schemaBase + name
		genlog.Debug("namespace schema registered", "schema", name)
	}
	return json.Marshal(root)
}

// Namespaces runs each embedded extension schema against its subtree of doc, after include merge.
func Namespaces(doc any) error {
	return validateAgainst(NamespacesURL, doc)
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
