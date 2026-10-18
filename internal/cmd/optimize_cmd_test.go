// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kiota.ch/projectfile/core/v2/pkg/projectfile"
)

func resetOptimizeFlags(t *testing.T) {
	t.Helper()
	optimizeDryRun = false
	optimizeFile = ""
	projectfile.SetYAMLOutputSorted(false)
}

// optFixture is a projectfile whose local field duplicates what its include already provides, which is the
// only situation optimize has anything to do.
func optFixture(t *testing.T, dir, baseBody string) string {
	t.Helper()
	writeFile(t, dir, "base.yaml", baseBody)
	return writeFile(t, dir, "projectfile.yaml",
		"$schema: https://projectfile.org/schema/v1.json\nspec_version: \"1\"\nkind: \"SoftwareSourceCode\"\n"+
			"identity:\n  namespace: org.example\n  name: demo\nversion: \"1.0.0\"\n"+
			"includes:\n  - base.yaml\n")
}

// TestOptimizeStripsFieldsTheIncludeAlreadyProvides pins the core contract: a value the include declares is
// removed from the local document, because leaving it makes every later include edit show as a local override.
func TestOptimizeStripsFieldsTheIncludeAlreadyProvides(t *testing.T) {
	dir := t.TempDir()
	path := optFixture(t, dir, "version: \"1.0.0\"\n")
	resetOptimizeFlags(t)

	_, err := runRootCmd(t, cmdOptimize, flagFile, path)
	require.NoError(t, err)

	out, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.NotContains(t, string(out), "\nversion:", "a redundant local version must be stripped")
	assert.Contains(t, string(out), "- base.yaml", "the include itself is kept")
}

// TestOptimizeKeepsTheIncludeDeclaration pins that optimize removes redundant VALUES, never the include edge
// that is the only reason the file is smaller.
func TestOptimizeKeepsTheIncludeDeclaration(t *testing.T) {
	dir := t.TempDir()
	path := optFixture(t, dir, "version: \"1.0.0\"\n")
	resetOptimizeFlags(t)

	_, err := runRootCmd(t, cmdOptimize, flagFile, path)
	require.NoError(t, err)

	doc, err := projectfile.ReadBaseFromPath(path)
	require.NoError(t, err)
	assert.NotEmpty(t, doc.Includes)
}

// TestOptimizePrunesARedundantInclude pins the include-pruning half: an entry a sibling already provides is
// dropped, so the next validate runs quiet instead of warning forever.
func TestOptimizePrunesARedundantInclude(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "extra.yaml", "description: extra\n")
	path := writeFile(t, dir, "projectfile.yaml",
		"$schema: https://projectfile.org/schema/v1.json\nspec_version: \"1\"\nkind: \"SoftwareSourceCode\"\n"+
			"identity:\n  namespace: org.example\n  name: demo\nincludes:\n  - base.yaml\n  - extra.yaml\n")
	writeFile(t, dir, "base.yaml", "includes:\n  - extra.yaml\n")
	resetOptimizeFlags(t)

	_, err := runRootCmd(t, cmdOptimize, flagFile, path)
	require.NoError(t, err)

	out, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.NotContains(t, string(out), "- extra.yaml", "a transitively provided include must be pruned")
}

// TestOptimizeDryRunLeavesTheFileUntouched pins that -n reports the removals without saving them, so a user can
// preview before letting go of local overrides.
func TestOptimizeDryRunLeavesTheFileUntouched(t *testing.T) {
	dir := t.TempDir()
	path := optFixture(t, dir, "version: \"1.0.0\"\n")
	before, err := os.ReadFile(path)
	require.NoError(t, err)
	resetOptimizeFlags(t)

	out, err := runRootCmd(t, cmdOptimize, flagFile, path, "-n")
	require.NoError(t, err)
	assert.Contains(t, out, "dry-run")

	after, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, string(before), string(after))
}

// TestOptimizeOnAnAlreadyOptimizedFileIsQuiet pins the idempotence contract: a second run finds nothing and
// says so, rather than reporting a removal it did not make.
func TestOptimizeOnAnAlreadyOptimizedFileIsQuiet(t *testing.T) {
	dir := t.TempDir()
	path := optFixture(t, dir, "version: \"1.0.0\"\n")
	resetOptimizeFlags(t)
	_, err := runRootCmd(t, cmdOptimize, flagFile, path)
	require.NoError(t, err)

	out, err := runRootCmd(t, cmdOptimize, flagFile, path)
	require.NoError(t, err)
	assert.Contains(t, out, "already optimized")
}

// TestOptimizeOnAFileWithNoIncludesIsANoOp pins that a document with nothing to compose against is left alone.
func TestOptimizeOnAFileWithNoIncludesIsANoOp(t *testing.T) {
	path := writeFile(t, t.TempDir(), "projectfile.yaml", "identity:\n  name: demo\n  version: \"1.0.0\"\n")
	resetOptimizeFlags(t)

	out, err := runRootCmd(t, cmdOptimize, flagFile, path)
	require.NoError(t, err)
	assert.Contains(t, out, "no includes declared")

	body, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(body), "version: \"1.0.0\"", "a document with no includes has no redundant values")
}

// TestOptimizeSortedRewritesTheIncludesSet pins that --sorted still reorders the include set on a document
// with nothing redundant, which is what makes a sorted file stable and diff-friendly.
func TestOptimizeSortedRewritesTheIncludesSet(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.yaml", "description: a\n")
	writeFile(t, dir, "b.yaml", "description: b\n")
	path := writeFile(t, dir, "projectfile.yaml",
		"identity:\n  name: demo\nincludes:\n  - b.yaml\n  - a.yaml\n")
	resetOptimizeFlags(t)

	_, err := runRootCmd(t, cmdOptimize, flagFile, path, "--sorted")
	require.NoError(t, err)

	out, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Less(t, indexOf(string(out), "- a.yaml"), indexOf(string(out), "- b.yaml"),
		"--sorted must order the include set")
}

// TestOptimizeAcceptsADirectoryArgument pins that the positional argument selects the project to work on, so
// the command can be run without --file.
func TestOptimizeAcceptsADirectoryArgument(t *testing.T) {
	dir := t.TempDir()
	optFixture(t, dir, "version: \"1.0.0\"\n")
	resetOptimizeFlags(t)

	_, err := runRootCmd(t, cmdOptimize, dir)
	require.NoError(t, err)

	out, err := os.ReadFile(filepath.Join(dir, "projectfile.yaml"))
	require.NoError(t, err)
	assert.NotContains(t, string(out), "\nversion:")
}

// TestResolveProjectfilePathForDirPrefersTheExplicitPath pins the precedence: --file skips detection entirely,
// so a caller never has the project it named silently replaced by one discovered upward.
func TestResolveProjectfilePathForDirPrefersTheExplicitPath(t *testing.T) {
	got, err := resolveProjectfilePathForDir(t.TempDir(), "/explicit/projectfile.yaml")
	require.NoError(t, err)
	assert.Equal(t, "/explicit/projectfile.yaml", got)
}

// TestResolveProjectfilePathForDirReportsAnAbsentProject pins that detection failure surfaces rather than
// yielding an empty path the caller would then read.
func TestResolveProjectfilePathForDirReportsAnAbsentProject(t *testing.T) {
	_, err := resolveProjectfilePathForDir(t.TempDir(), "")
	require.Error(t, err)
}

// TestResolveProjectfilePathReportsAnAbsentProject pins the same for the set/add/del resolver.
func TestResolveProjectfilePathReportsAnAbsentProject(t *testing.T) {
	t.Chdir(t.TempDir())
	_, err := resolveProjectfilePath("")
	require.Error(t, err)
}

// TestResolveProjectfilePathKeepsTheExplicitPath pins that an explicit path is returned verbatim.
func TestResolveProjectfilePathKeepsTheExplicitPath(t *testing.T) {
	got, err := resolveProjectfilePath("/explicit/projectfile.yaml")
	require.NoError(t, err)
	assert.Equal(t, "/explicit/projectfile.yaml", got)
}

// TestIncludeRefReadsBothEncodings pins that an includes entry is addressable as a bare string and as the
// pinned {url, sha256} mapping, so optimize can prune either form.
func TestIncludeRefReadsBothEncodings(t *testing.T) {
	assert.Equal(t, includeA, includeRef(includeA))
	assert.Equal(t, includeA, includeRef(map[string]any{keyURL: includeA, keySHA256: "00"}))
	assert.Empty(t, includeRef(42), "a non-string, non-mapping entry has no reference")
	assert.Empty(t, includeRef(map[string]any{keySHA256: "00"}), "a mapping with no url has no reference")
}

// TestDropRedundantIncludesLeavesForeignEntriesAlone pins that an entry outside the top-level list is not
// removed by a prune naming the same string elsewhere in the document.
func TestDropRedundantIncludesLeavesForeignEntriesAlone(t *testing.T) {
	raw := map[string]any{includesKey: []any{"keep.yaml"}, "com.example": map[string]any{"ref": "gone.yaml"}}
	dropRedundantIncludes(raw, []projectfile.RedundantInclude{{Ref: "gone.yaml", Via: "other.yaml"}})
	assert.Equal(t, []any{"keep.yaml"}, raw[includesKey], "only the top-level list is pruned")
}

// TestDropRedundantIncludesPrunesOneOccurrencePerEntry pins the counting: a document listing the same include
// twice keeps the second listing, since only one occurrence is redundant.
func TestDropRedundantIncludesPrunesOneOccurrencePerEntry(t *testing.T) {
	raw := map[string]any{includesKey: []any{includeDup, includeDup}}
	dropRedundantIncludes(raw, []projectfile.RedundantInclude{{Ref: includeDup, Via: "other.yaml"}})
	assert.Equal(t, []any{includeDup}, raw[includesKey])
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
