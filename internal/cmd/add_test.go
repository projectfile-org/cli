// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kiota.ch/projectfile/core/v2/pkg/genlog"
)

// listFixture is a projectfile whose keywords list already holds one entry, so the append paths have
// something to add beside and something to find already present.
const listFixture = `identity:
  namespace: org.example
  name: demo
keywords:
  - rust
`

func resetAddFlags(t *testing.T) {
	t.Helper()
	addFields = nil
	addValueJSON = ""
	addAllowDup = false
	addDryRun = false
	addFile = ""
}

// runAddCmd drives add in isolation, same Cobra-root pattern as runGetCmd.
func runAddCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	resetAddFlags(t)
	genlog.SetOutput(&bytes.Buffer{})
	t.Cleanup(func() { genlog.SetOutput(os.Stderr) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs(append([]string{cmdAdd}, args...))
	err := rootCmd.Execute()
	return buf.String(), err
}

// TestAddAppendsPositionalValues pins that bare positionals land in the list in the order typed.
func TestAddAppendsPositionalValues(t *testing.T) {
	path := writeFile(t, t.TempDir(), "projectfile.yaml", listFixture)
	_, err := runAddCmd(t, flagFile, path, keyKeywords, "wasm", "zig")
	require.NoError(t, err)

	out, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(out), "- rust")
	assert.Contains(t, string(out), "- wasm")
	assert.Contains(t, string(out), "- zig")
}

// TestAddSkipsAnItemAlreadyPresent pins the dedup contract: re-adding what is there leaves the file alone
// rather than duplicating the entry.
func TestAddSkipsAnItemAlreadyPresent(t *testing.T) {
	path := writeFile(t, t.TempDir(), "projectfile.yaml", listFixture)
	before, err := os.ReadFile(path)
	require.NoError(t, err)

	out, err := runAddCmd(t, flagFile, path, keyKeywords, "rust")
	require.NoError(t, err)
	assert.Contains(t, out, "already present")

	after, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, string(before), string(after), "an all-skipped add must not rewrite the file")
}

// TestAddAllowDuplicateAppendsAnyway pins that the escape hatch exists for a list where a repeat is meaningful.
func TestAddAllowDuplicateAppendsAnyway(t *testing.T) {
	path := writeFile(t, t.TempDir(), "projectfile.yaml", listFixture)
	_, err := runAddCmd(t, flagFile, path, keyKeywords, "rust", "--allow-duplicate")
	require.NoError(t, err)

	out, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, 2, countOccurrences(string(out), "- rust"))
}

// TestAddDryRunLeavesTheFileUntouched pins that -n reports what it would do without writing, so a user can
// preview an append before committing to it.
func TestAddDryRunLeavesTheFileUntouched(t *testing.T) {
	path := writeFile(t, t.TempDir(), "projectfile.yaml", listFixture)
	before, err := os.ReadFile(path)
	require.NoError(t, err)

	out, err := runAddCmd(t, flagFile, path, keyKeywords, "wasm", "-n")
	require.NoError(t, err)
	assert.Contains(t, out, "dry-run")
	assert.Contains(t, out, "1 added")

	after, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, string(before), string(after))
}

// TestAddRefusesAMissingValue pins the usage error: an append with nothing to append must say so instead of
// silently writing an empty list.
func TestAddRefusesAMissingValue(t *testing.T) {
	path := writeFile(t, t.TempDir(), "projectfile.yaml", listFixture)
	_, err := runAddCmd(t, flagFile, path, keyKeywords)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one value")
}

// TestAddRefusesMixedInputModes pins that the three input modes are exclusive, so a mistyped invocation
// cannot half-apply: a --value-json alongside a positional would otherwise silently drop one of them.
func TestAddRefusesMixedInputModes(t *testing.T) {
	path := writeFile(t, t.TempDir(), "projectfile.yaml", listFixture)
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"json with positional", []string{keyKeywords, "wasm", flagValueJSON, "{}"}, "--value-json is exclusive"},
		{"json with field", []string{keyKeywords, flagValueJSON, "{}", "--field", "url=x"}, "--value-json is exclusive"},
		{"field with positional", []string{keyKeywords, "wasm", "--field", "url=x"}, "--field is exclusive"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := runAddCmd(t, append([]string{flagFile, path}, tc.args...)...)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

// TestAddRejectsMalformedJSON pins that a --value-json which does not parse names the flag rather than
// reporting a bare decoder message.
func TestAddRejectsMalformedJSON(t *testing.T) {
	path := writeFile(t, t.TempDir(), "projectfile.yaml", listFixture)
	_, err := runAddCmd(t, flagFile, path, keyKeywords, flagValueJSON, "{not json")
	require.Error(t, err)
	assert.Contains(t, err.Error(), flagValueJSON)
}

// TestAddRejectsAFieldWithoutAnEquals pins the k=v grammar: a bare key would otherwise build an object with a
// missing value that no caller can interpret.
func TestAddRejectsAFieldWithoutAnEquals(t *testing.T) {
	path := writeFile(t, t.TempDir(), "projectfile.yaml", listFixture)
	for _, field := range []string{keyURL, "=orphan"} {
		_, err := runAddCmd(t, flagFile, path, keyKeywords, "--field", field)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "k=v")
	}
}

// TestAddFieldSplitsOnlyKnownStringLists pins the conservative CSV shorthand: roles and owners explode on a
// comma, a label or URL holding one does not.
func TestAddFieldSplitsOnlyKnownStringLists(t *testing.T) {
	for _, tc := range []struct {
		name  string
		field string
		want  []string
	}{
		{"roles split", "roles=author,maintainer", []string{"author", "maintainer"}},
		{"owners split", "owners=ada,grace", []string{"ada", "grace"}},
		{"url kept whole", "url=https://example.com/a,b", []string{"https://example.com/a,b"}},
		{"label kept whole", "label=team,infra", []string{"team,infra"}},
		{"unknown key kept whole", "notes=a,b", []string{"a,b"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetAddFlags(t)
			addFields = []string{tc.field}
			values, err := collectAddValues(nil)
			require.NoError(t, err)
			require.Len(t, values, 1)
			obj, ok := values[0].(map[string]any)
			require.True(t, ok, "--field must build one object")
			key, _, _ := strings.Cut(tc.field, "=")
			got := obj[key]
			if len(tc.want) == 1 {
				assert.Equal(t, tc.want[0], got)
				return
			}
			assert.Equal(t, toAny(tc.want), got)
		})
	}
}

// TestCollectAddValuesAcceptsEachModeAlone pins that every documented mode works on its own.
func TestCollectAddValuesAcceptsEachModeAlone(t *testing.T) {
	t.Run("positional", func(t *testing.T) {
		addFields, addValueJSON = nil, ""
		got, err := collectAddValues([]string{"a", "b"})
		require.NoError(t, err)
		assert.Equal(t, []any{"a", "b"}, got)
	})

	t.Run("value json", func(t *testing.T) {
		addFields, addValueJSON = nil, `{"url":"https://example.com/r"}`
		got, err := collectAddValues(nil)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, map[string]any{keyURL: "https://example.com/r"}, got[0])
	})

	t.Run("scalar json", func(t *testing.T) {
		addFields, addValueJSON = nil, "5"
		got, err := collectAddValues(nil)
		require.NoError(t, err)
		assert.Equal(t, []any{float64(5)}, got)
	})
}

// TestIsStringListField pins the closed set: widening it would silently explode a comma inside any newly
// named field, which is the failure the comment on the helper warns about.
func TestIsStringListField(t *testing.T) {
	for _, k := range []string{"roles", "owners"} {
		assert.True(t, isStringListField(k), "%s must split on commas", k)
	}
	for _, k := range []string{keyURL, "label", "notes", "Roles", ""} {
		assert.False(t, isStringListField(k), "%s must not split on commas", k)
	}
}

// TestAddRejectsAMalformedPath pins that a path the grammar cannot read fails before the file is opened, so a
// typo never half-writes.
func TestAddRejectsAMalformedPath(t *testing.T) {
	path := writeFile(t, t.TempDir(), "projectfile.yaml", listFixture)
	_, err := runAddCmd(t, flagFile, path, "keywords[", "wasm")
	require.Error(t, err)
}

// TestAddReportsAMissingProjectfile pins that pointing --file at something absent is an error naming the path,
// rather than an empty success.
func TestAddReportsAMissingProjectfile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "absent.yaml")
	_, err := runAddCmd(t, flagFile, missing, keyKeywords, "wasm")
	require.Error(t, err)
}

func countOccurrences(haystack, needle string) int {
	n := 0
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			n++
		}
	}
	return n
}

func toAny(xs []string) []any {
	out := make([]any, len(xs))
	for i, s := range xs {
		out[i] = s
	}
	return out
}
