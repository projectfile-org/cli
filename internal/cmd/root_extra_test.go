// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kiota.ch/projectfile/core/v2/pkg/genlog"
)

// TestFailOnFlagValueRoundTrips pins that --fail-on renders back to the spelling it accepted, since pflag
// prints the current value in usage output and a level enum would print as a number.
func TestFailOnFlagValueRoundTrips(t *testing.T) {
	for _, s := range []string{"error", "warning"} {
		var f failOnFlagValue
		require.NoError(t, f.Set(s))
		assert.Equal(t, s, f.String())
	}
}

// TestFailOnFlagValueRejectsAnUnknownLevel pins that a bad --fail-on is refused at parse time, naming the two
// accepted values, so it never silently keeps the previous level.
func TestFailOnFlagValueRejectsAnUnknownLevel(t *testing.T) {
	for _, s := range []string{"", "warn", "fatal", "ERROR"} {
		var f failOnFlagValue
		err := f.Set(s)
		require.Error(t, err, "%q must be refused", s)
		assert.Contains(t, err.Error(), "fail-on")
		assert.Contains(t, err.Error(), "error")
		assert.Contains(t, err.Error(), "warning")
	}
}

// TestFailOnFlagValueDefaultIsError pins the default level: a missing local include is a warning and is
// skipped, so consumers are not blocked while an include is being fixed upstream.
func TestFailOnFlagValueDefaultIsError(t *testing.T) {
	var zero failOnFlagValue
	assert.Equal(t, "error", zero.String(), "the zero value must render as the default level")
	assert.Equal(t, "error", failOnFlag.String(), "the live flag must render its own level")
}

// TestFailOnFlagValueType pins the declared flag type, so pflag's error text and usage line stay consistent.
func TestFailOnFlagValueType(t *testing.T) {
	var f failOnFlagValue
	assert.Equal(t, "string", f.Type())
}

// TestApplyColorsHonoursTheEnvOnlyWhenUnset pins the precedence: an explicit --colors wins, so a script can
// override an inherited NO_COLOR without editing the environment.
func TestApplyColorsHonoursTheEnvOnlyWhenUnset(t *testing.T) {
	t.Setenv("PF_CLI_NO_COLOR", "1")
	t.Cleanup(func() { _ = genlog.SetColor(genlog.ColorAuto) })

	require.NoError(t, rootCmd.Flags().Set("colors", "always"))
	require.NoError(t, applyColors(rootCmd))
	assert.Equal(t, genlog.ColorAlways, colorsFlag, "an explicit --colors must win over PF_CLI_NO_COLOR")
}

// TestApplyColorsReadsTheEnvWhenTheFlagIsUnset pins the inverse: with no flag, PF_CLI_NO_COLOR forces colour
// off, which is what a CI log needs.
func TestApplyColorsReadsTheEnvWhenTheFlagIsUnset(t *testing.T) {
	t.Setenv("PF_CLI_NO_COLOR", "1")
	t.Cleanup(func() { _ = genlog.SetColor(genlog.ColorAuto) })
	colorsFlag = genlog.ColorAuto
	if f := rootCmd.Flags().Lookup("colors"); f != nil {
		f.Changed = false
	}

	require.NoError(t, applyColors(rootCmd))
	assert.Equal(t, genlog.ColorNever, colorsFlag)
}

// TestClassifyDispatchErrorAlsoCoversUnknownFlag pins the third cobra dispatch message — the bare "unknown
// flag", which the existing table does not carry — stays in the usage class.
func TestClassifyDispatchErrorAlsoCoversUnknownFlag(t *testing.T) {
	got := classifyDispatchError(rootCmd, errDispatch{msg: "unknown flag: --nope"})
	assert.Equal(t, exitUsage, exitCodeFor(got))
}

// TestClassifyDispatchErrorPassesThroughASubcommandError pins that a RunE failure keeps its own class: a
// blanket re-class would turn a missing file into a usage mistake.
func TestClassifyDispatchErrorPassesThroughASubcommandError(t *testing.T) {
	got := classifyDispatchError(rootCmd, errValidation("the document fails the schema"))
	assert.Equal(t, exitInvalidated, exitCodeFor(got))
}

// TestResolveVersionPrefersTheReleaseStamp pins that a stamped build reports the stamp: the VCS fallback
// exists for unstamped builds, not for ones that know better.
func TestResolveVersionPrefersTheReleaseStamp(t *testing.T) {
	saved := version
	t.Cleanup(func() { version = saved })
	version = "3.2.1"
	assert.Equal(t, "3.2.1", resolveVersion())
}

// TestResolveVersionFallsBackToTheBuildStamp pins the dev-build shape: with no release stamp the answer comes
// from the build info, and an unstamped build says so in words rather than printing a bare "unknown".
func TestResolveVersionFallsBackToTheBuildStamp(t *testing.T) {
	saved := version
	t.Cleanup(func() { version = saved })
	version = "unknown"

	got := resolveVersion()
	// Either the toolchain stamped a VCS revision (dev-<sha>[-dirty]) or it stamped nothing at all; both are
	// valid, and neither may print the bare word with no explanation.
	if strings.HasPrefix(got, "dev-") {
		assert.Regexp(t, `^dev-[0-9a-f]{12}(-dirty)?$`, got)
		return
	}
	assert.Equal(t, "unknown (built without a release stamp or VCS metadata)", got)
}

// TestWrap80KeepsEveryLineWithinEighty pins the help-width contract: the fleet's help output is read in an
// 80-column terminal, so an over-long line wraps into ragged output.
func TestWrap80KeepsEveryLineWithinEighty(t *testing.T) {
	long := strings.Repeat("word ", 60)
	out := wrap80(long)
	for i, line := range strings.Split(out, "\n") {
		if len(line) > 80 {
			t.Errorf("line %d is %d columns, want <= 80: %q", i, len(line), line)
		}
	}
	assert.NotEmpty(t, strings.TrimSpace(out), "wrap80 must not swallow its input")
}

// TestWrap80LeavesShortLinesAlone pins that a line already inside the budget is not re-flowed, so intentional
// line breaks survive.
func TestWrap80LeavesShortLinesAlone(t *testing.T) {
	assert.Equal(t, "short line", wrap80("short line"))
	assert.Empty(t, wrap80(""))
}

// TestValidateFragmentRequiresAFile pins the usage error: --fragment with nothing to check must say so rather
// than validating the projectfile in the working directory.
func TestValidateFragmentRequiresAFile(t *testing.T) {
	require.Error(t, validateFragment(rootCmd, nil))
}

// TestValidateFragmentAcceptsAPartialDocument pins the point of --fragment: an include carries no
// discriminator and no required fields, so a document that would fail a full validate must pass as one.
func TestValidateFragmentAcceptsAPartialDocument(t *testing.T) {
	path := writeFile(t, t.TempDir(), "fragment.yaml", "com:\n  example:\n    build:\n      target: wasm\n")
	genlog.SetOutput(discard{})
	t.Cleanup(func() { genlog.SetOutput(nil) })
	require.NoError(t, validateFragment(rootCmd, []string{path}))
}

// TestValidateFragmentRejectsAnInvalidFragment pins that the relaxed mode still enforces the schema body: a
// malformed fragment is an error, not a free pass.
func TestValidateFragmentRejectsAnInvalidFragment(t *testing.T) {
	path := writeFile(t, t.TempDir(), "fragment.yaml", "identity:\n  name: 42\n")
	genlog.SetOutput(discard{})
	t.Cleanup(func() { genlog.SetOutput(nil) })
	require.Error(t, validateFragment(rootCmd, []string{path}))
}

// TestValidateFragmentReportsAMissingFile pins that an absent fragment path is an error rather than a
// vacuous pass.
func TestValidateFragmentReportsAMissingFile(t *testing.T) {
	require.Error(t, validateFragment(rootCmd, []string{t.TempDir() + "/absent.yaml"}))
}

type discard struct{}

func (discard) Write(p []byte) (int, error) { return len(p), nil }
