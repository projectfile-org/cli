// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kiota.ch/projectfile/core/v2/pkg/projectfile"
)

// foldedFixture is a document whose description is a folded scalar written the way an author writes it: several physical lines, none near the limit.
const foldedFixture = `---
$schema: https://projectfile.org/schema/v1.json
identity:
  name: demo
  description:
    en: >-
      This is a long description with several sentences that would
      certainly exceed any reasonable line length limit once it is folded
      onto a single physical line, and the consumer yamllint refuses it.
license:
  spdx: MIT
`

// TestWriteProjectfileKeepsTheFold is the regression the fleet felt in projectfile/actions: a plain `set` elsewhere in the document rewrote the canvas and collapsed the folded description onto one 300-character line, which the pre-push yamllint (line-length: 120) refused.
func TestWriteProjectfileKeepsTheFold(t *testing.T) {
	path := filepath.Join(t.TempDir(), "projectfile.yaml")
	require.NoError(t, os.WriteFile(path, []byte(foldedFixture), 0o600))

	var buf strings.Builder
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	resetSetFlags(t)
	rootCmd.SetArgs([]string{cmdSet, flagFile, path, keyIdentity + "." + keyName, valDemo})
	require.NoError(t, rootCmd.Execute())

	out, err := os.ReadFile(path)
	require.NoError(t, err)
	longest := 0
	for _, line := range strings.Split(string(out), "\n") {
		if len(line) > longest {
			longest = len(line)
		}
	}
	assert.LessOrEqual(t, longest, yamlLineWidth, "the folded description must stay inside the line limit:\n%s", out)
	assert.Contains(t, string(out), "en: >-")
}

// TestWriteProjectfileKeepsTheValue pins that the refold is a shape change only: folding a space into a line break is what the author wrote, so the value reads back exactly as it was.
func TestWriteProjectfileKeepsTheValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "projectfile.yaml")
	require.NoError(t, os.WriteFile(path, []byte(foldedFixture), 0o600))

	before, err := projectfile.ReadRawBaseFromPath(path)
	require.NoError(t, err)
	doc, err := readDocumentFromPath(path)
	require.NoError(t, err)
	require.NoError(t, writeProjectfile(doc, path))

	after, err := projectfile.ReadRawBaseFromPath(path)
	require.NoError(t, err)
	assert.Equal(t, before, after, "the refold must not change the value")
}

// TestRefoldFoldedBlocksKeepsParagraphBreaks pins that an empty line inside a folded block — a real newline in the value — survives the rewrap and is never a fold point, while each paragraph around it is still wrapped.
func TestRefoldFoldedBlocksKeepsParagraphBreaks(t *testing.T) {
	long := strings.Repeat("word ", 40)
	in := "a:\n  b: >-\n    " + long + "\n\n    " + long + "\n"
	out, changed := refoldFoldedBlocks(in)
	require.True(t, changed)
	assert.Equal(t, 1, strings.Count(out, "\n\n"), "the paragraph break must survive:\n%s", out)
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		assert.LessOrEqual(t, len(line), yamlLineWidth, "line too long: %q", line)
	}
	assert.Equal(t,
		strings.Join(strings.Fields(in), " "),
		strings.Join(strings.Fields(out), " "),
		"the value must be unchanged:\n%s", out)
}

// TestRefoldFoldedBlocksWrapsTheLongLine pins the positive case: the block is rewrapped at isolated spaces and the value still reads back the same.
func TestRefoldFoldedBlocksWrapsTheLongLine(t *testing.T) {
	in := "a:\n  b: >-\n    " + strings.Repeat("word ", 40) + "\n"
	out, changed := refoldFoldedBlocks(in)
	require.True(t, changed)
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		assert.LessOrEqual(t, len(line), yamlLineWidth, "line too long: %q", line)
	}
	assert.Equal(t,
		strings.Join(strings.Fields(in), " "),
		strings.Join(strings.Fields(out), " "),
		"the value must be unchanged:\n%s", out)
}

// TestRefoldFoldedScalarsSkipsOtherEncodings pins that a TOML or JSON document, which carries no block scalar, is left alone.
func TestRefoldFoldedScalarsSkipsOtherEncodings(t *testing.T) {
	dir := t.TempDir()
	toml := filepath.Join(dir, "projectfile.toml")
	require.NoError(t, os.WriteFile(toml, []byte("spec_version = \"1\"\n"), 0o600))
	require.NoError(t, refoldFoldedScalars(toml))

	out, err := os.ReadFile(toml)
	require.NoError(t, err)
	assert.Equal(t, "spec_version = \"1\"\n", string(out))
}

// TestRefoldFoldedBlocksLeavesUntouchedShapes pins the guards: a literal block carries its newlines as data, a value with no fold point cannot be wrapped, and a `>` inside a plain scalar is not a block indicator.
func TestRefoldFoldedBlocksLeavesUntouchedShapes(t *testing.T) {
	for _, tc := range []struct{ name, in string }{
		{"literal block", "a:\n  b: |\n    " + strings.Repeat("word ", 40) + "\n"},
		{"short folded", "a:\n  b: >-\n    short value\n"},
		{"one long word", "a:\n  b: >-\n    " + strings.Repeat("x", 200) + "\n"},
		{"not an indicator", "a: value > 5 is bigger\n"},
	} {
		out, changed := refoldFoldedBlocks(tc.in)
		assert.False(t, changed, "%s was rewritten", tc.name)
		assert.Equal(t, tc.in, out, "%s", tc.name)
	}
}
