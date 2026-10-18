// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kiota.ch/projectfile/core/v2/pkg/genlog"
)

func resetDelFlags(t *testing.T) {
	t.Helper()
	delStrict = false
	delDryRun = false
	delFile = ""
}

// runDelCmd drives del in isolation, same Cobra-root pattern as runGetCmd.
func runDelCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	resetDelFlags(t)
	genlog.SetOutput(&bytes.Buffer{})
	t.Cleanup(func() { genlog.SetOutput(os.Stderr) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs(append([]string{cmdDel}, args...))
	err := rootCmd.Execute()
	return buf.String(), err
}

// TestDelRemovesAListItem pins that an indexed delete removes exactly that element, which is what makes the
// index form worth having over a value match.
func TestDelRemovesAListItem(t *testing.T) {
	path := writeFile(t, t.TempDir(), "projectfile.yaml", listFixture)
	_, err := runDelCmd(t, flagFile, path, "keywords[0]")
	require.NoError(t, err)

	out, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.NotContains(t, string(out), "- rust")
}

// TestDelAbsentPathIsANoOp pins that deleting something already gone succeeds quietly, so a script can delete
// idempotently without guarding each call.
func TestDelAbsentPathIsANoOp(t *testing.T) {
	path := writeFile(t, t.TempDir(), "projectfile.yaml", listFixture)
	before, err := os.ReadFile(path)
	require.NoError(t, err)

	out, err := runDelCmd(t, flagFile, path, "keywords[99]")
	require.NoError(t, err)
	assert.Contains(t, out, "no-op")

	after, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, string(before), string(after), "an absent delete must not rewrite the file")
}

// TestDelStrictTurnsAnAbsentPathIntoAnError pins the opt-in for scripts that must know the field existed.
func TestDelStrictTurnsAnAbsentPathIntoAnError(t *testing.T) {
	path := writeFile(t, t.TempDir(), "projectfile.yaml", listFixture)
	_, err := runDelCmd(t, flagFile, path, "keywords[99]", "--strict")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.Contains(t, err.Error(), "--strict")
}

// TestDelDryRunLeavesTheFileUntouched pins that -n previews the deletion without saving it.
func TestDelDryRunLeavesTheFileUntouched(t *testing.T) {
	path := writeFile(t, t.TempDir(), "projectfile.yaml", listFixture)
	before, err := os.ReadFile(path)
	require.NoError(t, err)

	out, err := runDelCmd(t, flagFile, path, "keywords[0]", "-n")
	require.NoError(t, err)
	assert.Contains(t, out, "dry-run")

	after, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, string(before), string(after))
}

// TestDelRemovesAMapEntry pins the map form, which is how a namespaced extension field is dropped wholesale.
func TestDelRemovesAMapEntry(t *testing.T) {
	path := writeFile(t, t.TempDir(), "projectfile.yaml", "identity:\n  name: demo\ncom.example:\n  build:\n    target: wasm\n")
	_, err := runDelCmd(t, flagFile, path, "com.example.build")
	require.NoError(t, err)

	out, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.NotContains(t, string(out), "target: wasm")
}

// TestDelRejectsAMalformedPath pins that an unparseable address fails before the file is touched.
func TestDelRejectsAMalformedPath(t *testing.T) {
	path := writeFile(t, t.TempDir(), "projectfile.yaml", listFixture)
	before, err := os.ReadFile(path)
	require.NoError(t, err)

	_, err = runDelCmd(t, flagFile, path, "keywords[")
	require.Error(t, err)

	after, readErr := os.ReadFile(path)
	require.NoError(t, readErr)
	assert.Equal(t, string(before), string(after))
}

// TestDelReportsAMissingProjectfile pins that an absent --file target is an error rather than a silent no-op,
// which would read as "deleted successfully".
func TestDelReportsAMissingProjectfile(t *testing.T) {
	missing := t.TempDir() + "/absent.yaml"
	_, err := runDelCmd(t, flagFile, missing, "keywords[0]")
	require.Error(t, err)
}
