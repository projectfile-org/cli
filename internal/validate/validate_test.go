// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package validate_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"projectfile.org/projectfile/cli/internal/validate"
)

const testURLKey = "url"

func minimalValidDoc() map[string]any {
	return map[string]any{
		"$schema": validate.SchemaURL,
		"identity": map[string]any{
			"name":      "test-project",
			"namespace": "org.example",
			"version":   "1.0.0",
		},
	}
}

func TestValidateMinimalDoc(t *testing.T) {
	err := validate.Validate(minimalValidDoc())
	assert.NoError(t, err)
}

func TestValidateMissingIdentity(t *testing.T) {
	doc := map[string]any{
		"$schema": validate.SchemaURL,
	}
	err := validate.Validate(doc)
	assert.Error(t, err)
}

func TestValidateEmptyDoc(t *testing.T) {
	err := validate.Validate(map[string]any{})
	assert.Error(t, err)
}

func TestValidateWithLicense(t *testing.T) {
	doc := minimalValidDoc()
	doc["license"] = map[string]any{"spdx": "MIT"}
	err := validate.Validate(doc)
	assert.NoError(t, err)
}

func TestValidateWithPeople(t *testing.T) {
	doc := minimalValidDoc()
	doc["people"] = []any{
		map[string]any{
			"family-names": "Smith",
			"given-names":  "Alice",
			"email":        "alice@example.com",
			"roles":        []any{"author"},
		},
	}
	err := validate.Validate(doc)
	assert.NoError(t, err)
}

func TestValidateExtensionsAllowed(t *testing.T) {
	// Extensions must be nested maps (org → projectfile → ignores).
	// Dotted flat keys like "org.projectfile.ignores" are rejected by the schema.
	doc := minimalValidDoc()
	doc["org"] = map[string]any{
		"projectfile": map[string]any{
			"ignores": map[string]any{
				"generate": []any{".gitignore"},
			},
		},
	}
	err := validate.Validate(doc)
	assert.NoError(t, err)
}

func TestAsValidationErrorWraps(t *testing.T) {
	err := validate.Validate(map[string]any{})
	require.Error(t, err)
	ve, ok := validate.AsValidationError(err)
	assert.True(t, ok)
	assert.NotNil(t, ve)
}

func TestAsValidationErrorNilPassthrough(t *testing.T) {
	_, ok := validate.AsValidationError(nil)
	assert.False(t, ok)
}

func TestValidateWithRepositories(t *testing.T) {
	doc := minimalValidDoc()
	doc["repositories"] = []any{
		map[string]any{
			testURLKey: "https://github.com/acme/test",
			"role":     "origin",
		},
	}
	err := validate.Validate(doc)
	assert.NoError(t, err)
}

func TestValidateDerivedLink(t *testing.T) {
	doc := minimalValidDoc()
	doc["links"] = []any{
		map[string]any{
			"type":     "bugs",
			testURLKey: "https://github.com/acme/test/issues",
			"derived":  true,
		},
	}
	err := validate.Validate(doc)
	assert.NoError(t, err)
}

func TestValidateLinkExtraProperty(t *testing.T) {
	doc := minimalValidDoc()
	doc["links"] = []any{
		map[string]any{
			"type":              "homepage",
			testURLKey:          "https://example.com",
			"some-future-field": "value",
		},
	}
	err := validate.Validate(doc)
	assert.NoError(t, err)
}
