// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetSetFlags clears the shared set flag vars before one run: pflag's StringVar keeps the last value, so a --value-json from an earlier test would otherwise coerce this test's bare positional.
// flagFile and flagFormat are spelled once across the cmd tests, where goconst counts the occurrences.
const (
	flagFile   = "--file"
	flagFormat = "--format"
	valTrue    = "true"
)

func resetSetFlags(t *testing.T) {
	t.Helper()
	setValueJSON = ""
	setCSV = ""
	setCreateOnly = false
	setDryRun = false
	setFile = ""
}

// TestSetBarePositionalIsTextVerbatim pins the no-silent-rewrite contract: a bare value lands exactly as typed, including the forms a JSON parse used to coerce (1.10 became 1.1, true became a boolean, null became null).
func TestSetBarePositionalIsTextVerbatim(t *testing.T) {
	for _, tc := range []struct{ typed, want string }{
		{"1.10", "1.10"},
		{"1.50", "1.50"},
		{valTrue, valTrue},
		{"null", "null"},
		{"007", "007"},
		{`"quoted"`, `"quoted"`},
		{"v1.2.3", "v1.2.3"},
	} {
		path := filepath.Join(t.TempDir(), "projectfile.yaml")
		require.NoError(t, os.WriteFile(path, []byte("identity:\n  namespace: org.example\n  name: demo\n  version: \"0\"\n"), 0o600))

		var buf bytes.Buffer
		resetSetFlags(t)
		rootCmd.SetOut(&buf)
		rootCmd.SetErr(&buf)
		rootCmd.SetArgs([]string{cmdSet, flagFile, path, "identity.version", tc.typed})
		require.NoError(t, rootCmd.Execute(), "set %s", tc.typed)

		buf.Reset()
		resetGetFlags(t)
		rootCmd.SetOut(&buf)
		rootCmd.SetErr(&buf)
		rootCmd.SetArgs([]string{cmdGet, flagFile, path, flagFormat, "json", "identity.version"})
		require.NoError(t, rootCmd.Execute())
		want, _ := json.Marshal(tc.want)
		assert.JSONEq(t, string(want), buf.String(), "set %s must land verbatim as text", tc.typed)
	}
}

// TestSetTypedFormsStillCoerce pins that the JSON and CSV forms are the documented route to a non-text value, so removing the bare parse costs nothing.
func TestSetTypedFormsStillCoerce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "projectfile.yaml")
	require.NoError(t, os.WriteFile(path, []byte("identity:\n  namespace: org.example\n  name: demo\n"), 0o600))

	var buf bytes.Buffer
	resetSetFlags(t)
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"set", flagFile, path, "org.demo.count", "--value-json", "5"})
	require.NoError(t, rootCmd.Execute())

	out, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(out), "count: 5")
}

// TestSetRefusesAValueTheFieldCannotHold pins the type guard on the one path that still produces a non-text value, --value-json; the message names the flag that spells the type instead of a quoting dance.
func TestSetRefusesAValueTheFieldCannotHold(t *testing.T) {
	path := filepath.Join(t.TempDir(), "projectfile.yaml")
	before := []byte("identity:\n  name: demo\n")
	require.NoError(t, os.WriteFile(path, before, 0o600))

	var buf bytes.Buffer
	resetSetFlags(t)
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"set", flagFile, path, addrIdentityName, "--value-json", "true"})
	err := rootCmd.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--value-json")
	after, readErr := os.ReadFile(path)
	require.NoError(t, readErr)
	assert.Equal(t, before, after, "a refused set leaves the file byte-identical")
}
