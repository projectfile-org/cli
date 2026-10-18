// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kiota.ch/projectfile/core/v2/pkg/genlog"
)

// runRootExit drives Execute's body with args, so the exit CLASS is pinnable without exiting the test process; the Error line goes to the real stderr, which is captured here.
func runRootExit(t *testing.T, args ...string) (int, string) {
	t.Helper()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs(args)
	t.Cleanup(func() {
		rootCmd.SetArgs(nil)
		rootCmd.SetOut(genlog.Styled(os.Stdout))
		rootCmd.SetErr(genlog.Styled(os.Stderr))
		resetVisited := func(c *cobra.Command) {
			c.Flags().Visit(func(f *pflag.Flag) { _ = f.Value.Set(f.DefValue) })
		}
		resetVisited(rootCmd)
		for _, c := range rootCmd.Commands() {
			resetVisited(c)
		}
	})
	r, w, err := os.Pipe()
	require.NoError(t, err)
	saved := os.Stderr
	os.Stderr = w
	code := runRoot()
	os.Stderr = saved
	require.NoError(t, w.Close())
	captured, err := io.ReadAll(r)
	require.NoError(t, err)
	require.NoError(t, r.Close())
	return code, string(captured)
}

// TestRunRootSuccessIsExitOK pins that a run that completes reports no exit class, so Execute leaves the process exit alone.
func TestRunRootSuccessIsExitOK(t *testing.T) {
	code, _ := runRootExit(t, "--version")
	assert.Equal(t, exitOK, code)
}

// TestRunRootMapsADispatchMistakeToUsage pins the re-class: an unknown command is a bad invocation, not a runtime failure.
func TestRunRootMapsADispatchMistakeToUsage(t *testing.T) {
	code, stderr := runRootExit(t, "bogus")
	assert.Equal(t, exitUsage, code)
	assert.Contains(t, stderr, "unknown command")
}

// TestRunRootMapsARuntimeFailureToOne pins that a subcommand's own failure keeps the runtime class: a blanket re-class would send a missing file to the usage branch.
func TestRunRootMapsARuntimeFailureToOne(t *testing.T) {
	code, stderr := runRootExit(t, cmdGet, "--file", filepath.Join(t.TempDir(), "absent.yaml"), addrIdentityName)
	assert.Equal(t, exitFailure, code)
	assert.Contains(t, stderr, "Error:")
}

// TestRootRunERejectsAPositionalArgument pins the bare-invocation guard: an argument the dispatcher let through is still refused, in the usage class.
func TestRootRunERejectsAPositionalArgument(t *testing.T) {
	require.NotNil(t, rootCmd.RunE)
	err := rootCmd.RunE(rootCmd, []string{"bogus"})
	require.Error(t, err)
	assert.Equal(t, exitUsage, exitCodeFor(err))
	assert.Contains(t, err.Error(), "unknown command")
}

// TestErrorKindsCarryTheirMessage pins the three marker errors' message accessor, which every caller formats through.
func TestErrorKindsCarryTheirMessage(t *testing.T) {
	assert.Equal(t, "u", errUsage("u").Error())
	assert.Equal(t, "a", errAbsent("a").Error())
	assert.Equal(t, "v", errValidation("v").Error())
}

// TestDevStampReadsTheRevisionAndTheDirtyMark pins the unstamped-build answer in its three shapes: a clean revision, a modified worktree, and no VCS metadata at all.
func TestDevStampReadsTheRevisionAndTheDirtyMark(t *testing.T) {
	saved := version
	t.Cleanup(func() { version = saved })
	version = "unknown"

	clean := debug.BuildSetting{Key: "vcs.revision", Value: "0123456789abcdef0123"}
	assert.Equal(t, "dev-0123456789ab", devStamp([]debug.BuildSetting{clean}))

	dirty := debug.BuildSetting{Key: "vcs.modified", Value: "true"}
	assert.Equal(t, "dev-0123456789ab-dirty", devStamp([]debug.BuildSetting{clean, dirty}))

	short := debug.BuildSetting{Key: "vcs.revision", Value: "0123"}
	assert.Equal(t, "unknown (built without a release stamp or VCS metadata)", devStamp([]debug.BuildSetting{short}))

	assert.Equal(t, "unknown (built without a release stamp or VCS metadata)", devStamp(nil))
	assert.Equal(t, "unknown (built without a release stamp or VCS metadata)",
		devStamp([]debug.BuildSetting{{Key: "vcs.modified", Value: "false"}}))
}

// TestConciseHelpWithoutLinksStaysConcise pins the no-links screen: a build with no embedded projectfile still prints the summary and the pointer to --help.
func TestConciseHelpWithoutLinksStaysConcise(t *testing.T) {
	var buf bytes.Buffer
	conciseHelp(&buf, "", "")
	out := buf.String()
	assert.NotContains(t, out, "Docs:")
	assert.NotContains(t, out, "Issues:")
	assert.True(t, strings.HasSuffix(out, "Run pf-cli --help for every command and flag.\n"))
}

// TestProjectfileDescriptionSurvivesBrokenYAML pins the fallback: an unparseable embedded projectfile yields no description rather than a panic.
func TestProjectfileDescriptionSurvivesBrokenYAML(t *testing.T) {
	saved := projectfileYAML
	t.Cleanup(func() { projectfileYAML = saved })
	SetProjectfileYAML([]byte("\tnot: [valid"))
	assert.Empty(t, projectfileDescription())
	assert.True(t, strings.HasPrefix(rootLong(), rootShort))
}

// TestDocLinksWalkEveryFallback pins the resolution order: a public tag wins, then the preferred source repository, then any source repository, and a broken document yields no links at all.
func TestDocLinksWalkEveryFallback(t *testing.T) {
	saved := projectfileYAML
	t.Cleanup(func() { projectfileYAML = saved })

	preferred := "links:\n  - type: source-code\n    preferred: true\n    url: https://kiota.ch/projectfile/cli\n"
	SetProjectfileYAML([]byte(preferred))
	docs, issues := docLinks()
	assert.Equal(t, "https://kiota.ch/projectfile/cli/tree/main/docs", docs)
	assert.Empty(t, issues)

	anySource := "links:\n  - type: documentation\n    url: https://example.org/docs\n  - type: source-code\n    url: https://example.org/src\n"
	SetProjectfileYAML([]byte(anySource))
	docs, _ = docLinks()
	assert.Equal(t, "https://example.org/src/tree/main/docs", docs)

	SetProjectfileYAML([]byte("links: [oops"))
	docs, issues = docLinks()
	assert.Empty(t, docs)
	assert.Empty(t, issues)
}
