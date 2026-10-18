// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResolveSetValueRequiresExactlyOneSource pins the ambiguity guard: naming two sources at once is
// rejected up front, so the user sees "you said both" instead of one silently winning.
func TestResolveSetValueRequiresExactlyOneSource(t *testing.T) {
	t.Run("no source", func(t *testing.T) {
		resetSetFlags(t)
		_, err := resolveSetValue([]string{addrDemoCount})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "requires a value")
	})

	t.Run("two sources", func(t *testing.T) {
		resetSetFlags(t)
		setValueJSON = "5"
		_, err := resolveSetValue([]string{addrDemoCount, "7"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "only one of")
	})

	t.Run("positional with csv", func(t *testing.T) {
		resetSetFlags(t)
		setCSV = "a,b"
		_, err := resolveSetValue([]string{addrDemoKeywords, "x"})
		require.Error(t, err)
	})
}

// TestResolveSetValueAcceptsEachSourceAlone pins that every documented source works on its own.
func TestResolveSetValueAcceptsEachSourceAlone(t *testing.T) {
	t.Run("positional", func(t *testing.T) {
		resetSetFlags(t)
		got, err := resolveSetValue([]string{addrDemoCount, "5"})
		require.NoError(t, err)
		assert.Equal(t, "5", got, "a bare positional is text, never a coerced number")
	})

	t.Run("value json", func(t *testing.T) {
		resetSetFlags(t)
		setValueJSON = "5"
		got, err := resolveSetValue([]string{addrDemoCount})
		require.NoError(t, err)
		assert.Equal(t, float64(5), got)
	})

	t.Run("csv", func(t *testing.T) {
		resetSetFlags(t)
		setCSV = "a, b ,c"
		got, err := resolveSetValue([]string{addrDemoKeywords})
		require.NoError(t, err)
		assert.Equal(t, []any{"a", "b", "c"}, got)
	})
}

// TestResolveSetValueRejectsMalformedJSON pins that a --value-json which does not parse names the flag rather
// than reporting a bare decoder message.
func TestResolveSetValueRejectsMalformedJSON(t *testing.T) {
	resetSetFlags(t)
	setValueJSON = "{not json"
	_, err := resolveSetValue([]string{addrDemoCount})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--value-json")
}

// TestSetCSVKeepsEmptyCells pins that an empty cell is preserved rather than dropped: a caller splitting on a
// comma wants the same arity it typed, which is what lets an empty trailing element clear a value.
func TestSetCSVKeepsEmptyCells(t *testing.T) {
	resetSetFlags(t)
	setCSV = "a,,b"
	got, err := resolveSetValue([]string{addrDemoKeywords})
	require.NoError(t, err)
	assert.Equal(t, []any{"a", "", "b"}, got)
}

// TestSetDryRunLeavesTheFileUntouched pins that -n previews a write without saving it.
func TestSetDryRunLeavesTheFileUntouched(t *testing.T) {
	path := writeFile(t, t.TempDir(), "projectfile.yaml", "identity:\n  name: demo\n")
	before, err := os.ReadFile(path)
	require.NoError(t, err)

	resetSetFlags(t)
	out, err := runRootCmd(t, cmdSet, flagFile, path, addrDemoCount, "5", "-n")
	require.NoError(t, err)
	assert.Contains(t, out, "dry-run")

	after, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, string(before), string(after))
}

// TestSetCreateOnlyRefusesToOverwrite pins that --create-only reports an existing value instead of replacing
// it, which is what lets a scaffold step be re-run without clobbering edits.
func TestSetCreateOnlyRefusesToOverwrite(t *testing.T) {
	path := writeFile(t, t.TempDir(), "projectfile.yaml", "identity:\n  name: demo\norg:\n  demo:\n    count: 5\n")
	before, err := os.ReadFile(path)
	require.NoError(t, err)

	resetSetFlags(t)
	_, err = runRootCmd(t, cmdSet, flagFile, path, addrDemoCount, "7", "--create-only")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already has a value")

	after, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, string(before), string(after))
}

// TestSetCreateOnlyFillsAnAbsentValue pins the other half: the flag exists to seed a scaffold, so a path with
// nothing there must be filled rather than refused.
func TestSetCreateOnlyFillsAnAbsentValue(t *testing.T) {
	path := writeFile(t, t.TempDir(), "projectfile.yaml", "identity:\n  name: demo\norg:\n  demo:\n    other: x\n")
	resetSetFlags(t)

	_, err := runRootCmd(t, cmdSet, flagFile, path, addrDemoCount, "5", "--create-only")
	require.NoError(t, err)

	out, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(out), `count: "5"`, "a bare positional stays text")
}

// TestSetRejectsAMalformedPath pins that an unparseable address fails before the file is opened.
func TestSetRejectsAMalformedPath(t *testing.T) {
	path := writeFile(t, t.TempDir(), "projectfile.yaml", "identity:\n  name: demo\n")
	resetSetFlags(t)
	_, err := runRootCmd(t, cmdSet, flagFile, path, "identity[", "x")
	require.Error(t, err)
}

// TestSetReportsAMissingProjectfile pins that an absent --file target is an error rather than a silent success.
func TestSetReportsAMissingProjectfile(t *testing.T) {
	resetSetFlags(t)
	_, err := runRootCmd(t, cmdSet, flagFile, t.TempDir()+"/absent.yaml", "identity.name", "x")
	require.Error(t, err)
}
