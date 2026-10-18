// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package validate

import (
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompileSchemasReportsABrokenEmbed pins each way the one-shot compile can fail, naming the schema it choked on: a truncated embed, a schema the metaschema rejects and a pattern no ECMA-262 engine accepts.
func TestCompileSchemasReportsABrokenEmbed(t *testing.T) {
	_, err := compileSchemas([]byte("{"), fragmentSchemaBytes)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse embedded schema "+SchemaURL)

	_, err = compileSchemas([]byte(`{"$id":"`+SchemaURL+`","type":"nonsense"}`), fragmentSchemaBytes)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "compile embedded schema")

	_, err = compileSchemas([]byte(`{"$id":"`+SchemaURL+`","pattern":"("}`), fragmentSchemaBytes)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not valid regex")
}

// TestCompileSchemasAcceptsTheEmbeddedPair pins the happy path the cache serves every later call from: both schemas compile and answer by URL.
func TestCompileSchemasAcceptsTheEmbeddedPair(t *testing.T) {
	got, err := compileSchemas(schemaBytes, fragmentSchemaBytes)
	require.NoError(t, err)
	assert.Len(t, got, 2)
	assert.NotNil(t, got[SchemaURL])
	assert.NotNil(t, got[FragmentSchemaURL])
}

// TestValidateReportsAnUnencodableDocument pins that a document the JSON round-trip cannot carry is a validation failure in its own voice, not a silent pass.
func TestValidateReportsAnUnencodableDocument(t *testing.T) {
	err := Validate(map[string]any{keyIdentity: map[string]any{"name": math.NaN()}})
	require.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "normalise document for validation"), "got %q", err)
}

// TestFragmentReportsAnUnencodableDocument pins the same contract on the include-only schema.
func TestFragmentReportsAnUnencodableDocument(t *testing.T) {
	require.Error(t, Fragment(math.NaN()))
}

// TestECMA262RegexpRejectsAPatternItCannotCompile pins the schema contract: an invalid pattern is an error, never a pattern that silently matches nothing.
func TestECMA262RegexpRejectsAPatternItCannotCompile(t *testing.T) {
	_, err := ecma262Regexp("(")
	require.Error(t, err)
}

// TestECMA262RegexpCarriesItsSource pins the source the validator reports back to the user as the pattern that failed.
func TestECMA262RegexpCarriesItsSource(t *testing.T) {
	re, err := ecma262Regexp(`^a+$`)
	require.NoError(t, err)
	assert.Equal(t, `^a+$`, re.String())
	assert.True(t, re.MatchString("aaa"))
	assert.False(t, re.MatchString("b"))
}
