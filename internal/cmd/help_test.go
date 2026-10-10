// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"regexp"
	"slices"
	"strings"
	"testing"
)

// ansiRE matches the escape sequences the help palette emits, so assertions read the text a user sees.
var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// plain strips ANSI escapes from one rendered help screen.
func plain(s string) string { return ansiRE.ReplaceAllString(s, "") }

// pfYAMLForLinks is a minimal projectfile carrying the two link families the help renders: a public source repository (docs) and a bug tracker.
const pfYAMLForLinks = `identity:
  name: cli
links:
  - type: source-code
    preferred: true
    url: https://kiota.ch/projectfile/cli
  - type: source-code
    tags: [public]
    url: https://github.com/projectfile-org/cli
  - type: bugs
    url: https://github.com/projectfile-org/cli/issues
`

// TestBareInvocationIsConcise pins the first screen a new user sees: a one-line description, a few examples, the common flags and the pointer to --help — never the full command list.
func TestBareInvocationIsConcise(t *testing.T) {
	out, err := runRootCmd(t)
	out = plain(out)
	if err != nil {
		t.Fatalf("bare invocation: %v", err)
	}
	for _, want := range []string{
		"pf-cli get identity.name",
		"pf-cli set license.spdx MIT",
		"pf-cli validate",
		"--quiet",
		"Run pf-cli --help for every command and flag.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("bare output missing %q:\n%s", want, out)
		}
	}
	for _, unwanted := range []string{"Usage:", "Aliases:", "Global Flags:"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("bare output should stay concise, found %q:\n%s", unwanted, out)
		}
	}
	if len(out) > 900 {
		t.Errorf("bare output is %d bytes, want a short summary", len(out))
	}
}

// TestFullHelpKeepsEveryCommand pins that the full screen still carries the exhaustive list, so the concise form never hides a command.
func TestFullHelpKeepsEveryCommand(t *testing.T) {
	out, err := runRootCmd(t, "--help")
	out = plain(out)
	if err != nil {
		t.Fatalf("--help: %v", err)
	}
	for _, name := range []string{cmdGet, cmdSet, cmdAdd, cmdDel, cmdValidate, cmdCache, cmdConvert, cmdIncludes, cmdInit, cmdOptimize, cmdSetup, cmdCompletion} {
		if !strings.Contains(out, "  "+name) {
			t.Errorf("--help missing the %s command:\n%s", name, out)
		}
	}
}

// TestHelpLeadsWithExamples pins the checklist order: Examples come before the command list, so a reader sees a worked invocation first.
func TestHelpLeadsWithExamples(t *testing.T) {
	out, err := runRootCmd(t, "--help")
	out = plain(out)
	if err != nil {
		t.Fatalf("--help: %v", err)
	}
	ex := strings.Index(out, "Examples:")
	cmds := strings.Index(out, "\nRead:")
	if ex < 0 || cmds < 0 || ex > cmds {
		t.Fatalf("Examples (%d) must precede the command groups (%d):\n%s", ex, cmds, out)
	}
}

// TestHelpGroupsCommandsAlphabetically pins that each group is alphabetical and the everyday read/write commands head the list.
func TestHelpGroupsCommandsAlphabetically(t *testing.T) {
	out, err := runRootCmd(t, "--help")
	out = plain(out)
	if err != nil {
		t.Fatalf("--help: %v", err)
	}
	order := []string{"Read:", "Write:", "Validate:", "Maintain:", "Other:"}
	at := -1
	for _, g := range order {
		i := strings.Index(out, g)
		if i < 0 {
			t.Fatalf("missing group %q in:\n%s", g, out)
		}
		if i < at {
			t.Errorf("group %q is out of order in:\n%s", g, out)
		}
		at = i
	}
	block := out[strings.Index(out, "Write:"):strings.Index(out, "Validate:")]
	names := groupCommands(block)
	if !slices.IsSorted(names) {
		t.Errorf("Write group is not alphabetical: %v", names)
	}
}

// groupCommands extracts the command names from one rendered help group.
func groupCommands(block string) []string {
	var names []string
	for _, line := range strings.Split(block, "\n") {
		name := strings.TrimSpace(line)
		if name == "" || !strings.HasPrefix(line, "   ") {
			continue
		}
		names = append(names, strings.Fields(name)[0])
	}
	return names
}

// TestHelpRendersDocAndIssueLinks pins that the root and each subcommand link the docs and the tracker, the subcommand page being an anchor.
func TestHelpRendersDocAndIssueLinks(t *testing.T) {
	saved := projectfileYAML
	t.Cleanup(func() { projectfileYAML = saved })
	SetProjectfileYAML([]byte(pfYAMLForLinks))

	out, err := runRootCmd(t, "--help")
	out = plain(out)
	if err != nil {
		t.Fatalf("--help: %v", err)
	}
	for _, want := range []string{
		"https://github.com/projectfile-org/cli/tree/main/docs",
		"https://github.com/projectfile-org/cli/issues",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("root help missing %q:\n%s", want, out)
		}
	}

	out, err = runRootCmd(t, cmdGet, "--help")
	out = plain(out)
	if err != nil {
		t.Fatalf("get --help: %v", err)
	}
	wantPage := "https://github.com/projectfile-org/cli/blob/main/docs/USAGE.md#pf-cli-get"
	if !strings.Contains(out, wantPage) {
		t.Errorf("get help missing %q:\n%s", wantPage, out)
	}
	if !strings.Contains(out, "https://github.com/projectfile-org/cli/issues") {
		t.Errorf("get help missing the issue tracker:\n%s", out)
	}
}

// TestDocLinksPreferThePublicForge pins that the docs URL is the reachable one, not the preferred private host.
func TestDocLinksPreferThePublicForge(t *testing.T) {
	saved := projectfileYAML
	t.Cleanup(func() { projectfileYAML = saved })
	SetProjectfileYAML([]byte(pfYAMLForLinks))
	docs, issues := docLinks()
	if docs != "https://github.com/projectfile-org/cli/tree/main/docs" {
		t.Errorf("docs = %q, want the public forge's docs tree", docs)
	}
	if issues != "https://github.com/projectfile-org/cli/issues" {
		t.Errorf("issues = %q, want the bug tracker", issues)
	}
}

// TestEveryCommandHasADocPage keeps the per-subcommand link honest: a command with no page links nothing, so this catches a registration that bypassed groupCmd; cobra's own completion and help resolve from their names.
func TestEveryCommandHasADocPage(t *testing.T) {
	saved := projectfileYAML
	t.Cleanup(func() { projectfileYAML = saved })
	SetProjectfileYAML([]byte(pfYAMLForLinks))

	registered := map[string]bool{}
	for _, c := range rootCmd.Commands() {
		registered[c.Name()] = true
		if c.Name() == "help" {
			continue // cobra's own dispatcher has no page of its own
		}
		page := c.Annotations["doc-page"]
		if page == "" {
			page = c.Name()
		}
		if got := docsForPage(page); !strings.Contains(got, "pf-cli-"+strings.ReplaceAll(page, " ", "-")) {
			t.Errorf("pf-cli %s: doc page %q resolves to %q", c.Name(), page, got)
		}
	}
	for _, want := range []string{cmdGet, cmdSet, cmdAdd, cmdDel, cmdValidate, cmdCache, cmdConvert, cmdInit, cmdOptimize, cmdSetup, cmdIncludes} {
		if !registered[want] {
			t.Errorf("command %s is not registered", want)
		}
	}
}
