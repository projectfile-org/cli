// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kiota.ch/projectfile/core/v2/pkg/fieldpath"
	"kiota.ch/projectfile/core/v2/pkg/projectfile"
)

// TestShellValueQuotesScalarsSafely pins the eval-safety contract: a value carrying a shell metacharacter must
// come back quoted, because --format sh output is meant to be pasted into a script and evaluated.
func TestShellValueQuotesScalarsSafely(t *testing.T) {
	resetGetFlags(t)
	for _, tc := range []struct{ name, in, want string }{
		{"plain word passes through", "simple", "simple"},
		{"empty becomes an empty quoted pair", "", "''"},
		{"space forces quoting", "two words", "'two words'"},
		{"dollar is neutralised", "$HOME", "'$HOME'"},
		{"backtick is neutralised", "`id`", "'`id`'"},
		{"command substitution is neutralised", "$(id)", "'$(id)'"},
		{"semicolon is neutralised", "a;b", "'a;b'"},
		{"pipe is neutralised", "a|b", "'a|b'"},
		{"glob is neutralised", "*.go", "'*.go'"},
		{"embedded quote is escaped the POSIX way", "it's", `'it'\''s'`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, shellValue(tc.in))
		})
	}
}

// TestShellValueJoinsListsWithSpaces pins that a list becomes one quoted argument, so a shell variable holding
// several values survives the round trip as the whole list rather than as separate words.
func TestShellValueJoinsListsWithSpaces(t *testing.T) {
	resetGetFlags(t)
	assert.Equal(t, "'a b'", shellValue([]any{"a", "b"}))
	assert.Equal(t, "'a b'", shellValue([]string{"a", "b"}))
}

// TestShellQuoteRoundTripsThroughEval pins that eval'ing the output and reading it back gives the original,
// which is the only property that makes --format sh usable. Idempotence is deliberately NOT the contract: the
// second pass sees the quote characters themselves, which is correct — the value is already a quoted literal.
func TestShellQuoteRoundTripsThroughEval(t *testing.T) {
	resetGetFlags(t)
	for _, in := range []string{"plain", "two words", "it's", "$x; rm -rf /", "back`tick`", "a|b", "*.go"} {
		assert.Equal(t, in, evalShell(t, shellValue(in)), "eval of shellValue(%q) must give back %q", in, in)
	}
}

// TestLoadDocumentResolvesFromAnExplicitPath pins the --file route: an explicit path is read even when the
// working directory holds no projectfile at all.
func TestLoadDocumentResolvesFromAnExplicitPath(t *testing.T) {
	resetGetFlags(t)
	path := writeFile(t, t.TempDir(), "projectfile.yaml", "identity:\n  name: explicit\n")
	t.Chdir(t.TempDir())

	doc, err := loadDocument(path)
	require.NoError(t, err)
	assert.Equal(t, "explicit", doc.Identity.Name)
}

// TestLoadDocumentFallsBackToDetection pins that an unset --file detects the projectfile in the working
// directory, which is what a bare `pf-cli get` relies on.
func TestLoadDocumentFallsBackToDetection(t *testing.T) {
	resetGetFlags(t)
	dir := t.TempDir()
	writeFile(t, dir, "projectfile.yaml", "identity:\n  name: detected\n")
	t.Chdir(dir)

	doc, err := loadDocument("")
	require.NoError(t, err)
	assert.Equal(t, "detected", doc.Identity.Name)
}

// TestLoadDocumentReportsAnAbsentProjectfile pins that no document yields an error rather than an empty one, so
// a script can tell "nothing here" from "nothing at that address".
func TestLoadDocumentReportsAnAbsentProjectfile(t *testing.T) {
	resetGetFlags(t)
	t.Chdir(t.TempDir())
	_, err := loadDocument("")
	require.Error(t, err)
}

// TestLoadDocumentExpandsBracedEnvOnly pins the deliberate divergence from envsubst: `${VAR}` is substituted
// while a bare `$schema` is left verbatim, since expanding it would destroy the YAML discriminator.
func TestLoadDocumentExpandsBracedEnvOnly(t *testing.T) {
	resetGetFlags(t)
	getExpandEnv = true
	t.Setenv("TEST_TARGET", "wasm")
	path := writeFile(t, t.TempDir(), "projectfile.yaml",
		"$schema: https://projectfile.org/schema/v1.json\nidentity:\n  name: ${TEST_TARGET}\n")

	doc, err := loadDocument(path)
	require.NoError(t, err)
	assert.Equal(t, "wasm", doc.Identity.Name, "${TEST_TARGET} must be substituted")

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(raw), "$schema", "the file on disk is untouched by the expansion")
}

// TestLoadDocumentReportsAnUnreadablePathUnderExpandEnv pins that the expand-env branch reads the file itself,
// so a missing path is reported with the path named.
func TestLoadDocumentReportsAnUnreadablePathUnderExpandEnv(t *testing.T) {
	resetGetFlags(t)
	getExpandEnv = true
	_, err := loadDocument(t.TempDir() + "/absent.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "absent.yaml")
}

// TestExpandBracedEnvSubstitutesOnlyBraces pins the substitution rule at its edges: a closed `${VAR}` is
// replaced, an unclosed one is left alone, and a bare `$VAR` survives.
func TestExpandBracedEnvSubstitutesOnlyBraces(t *testing.T) {
	t.Setenv("PF_TEST_VAR", "value")
	t.Setenv("PF_TEST_UNSET", "")
	for _, tc := range []struct{ name, in, want string }{
		{"braced is substituted", "a ${PF_TEST_VAR} b", "a value b"},
		{"bare is left verbatim", "a $PF_TEST_VAR b", "a $PF_TEST_VAR b"},
		{"unclosed brace is left verbatim", "a ${PF_TEST_VAR b", "a ${PF_TEST_VAR b"},
		{"unset braced becomes empty", "[${PF_TEST_UNSET}]", "[]"},
		{"unknown braced becomes empty", "[${PF_TEST_ABSENT}]", "[]"},
		{"no dollar is untouched", "plain text", "plain text"},
		{"empty name collapses", "a ${} b", "a  b"},
		{"empty input", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, expandBracedEnv(tc.in))
		})
	}
}

// TestDeriveKeyNamesAPathByItsAddress pins the --path naming rule: a positional path keeps its dotted address
// as the key, with only the bracket tail dropped so every member of one list shares a single key.
func TestDeriveKeyNamesAPathByItsAddress(t *testing.T) {
	for _, tc := range []struct{ addr, want string }{
		{"com.example.env.FOO", "com.example.env.FOO"},
		{"com.example.env[0]", "com.example.env"},
		{"com.example.env[12]", "com.example.env"},
		{addrIdentityName, addrIdentityName},
		{"", ""},
	} {
		if got := deriveKey(tc.addr); got != tc.want {
			t.Errorf("deriveKey(%q) = %q, want %q", tc.addr, got, tc.want)
		}
	}
}

// evalShell runs the quoted form through a real shell and reports what the variable would hold, which is how
// the --format sh contract is actually verified.
func evalShell(t *testing.T, quoted string) string {
	t.Helper()
	script := "v=" + quoted + "; printf '%s' \"$v\""
	out, err := exec.Command("sh", "-c", script).Output()
	if err != nil {
		t.Fatalf("eval %s: %v", script, err)
	}
	return string(out)
}

// TestDefaultValueForIgnoresAnUnsetDefault pins that --default only substitutes for an ABSENT field, never over
// a value the document really holds: with the flag unset the caller gets no fallback at all.
func TestDefaultValueForIgnoresAnUnsetDefault(t *testing.T) {
	resetGetFlags(t)
	v, ok := defaultValueFor(&projectfile.Document{}, fieldpath.Path{}, false)
	assert.False(t, ok, "an unset --default must not inject a value")
	assert.Nil(t, v)
}

// TestDefaultValueForSubstitutesWhenTheFlagWasPassed pins the actual contract: --default wins on presence, not
// on a non-empty value, so an empty --default still yields the empty string rather than falling through.
func TestDefaultValueForSubstitutesWhenTheFlagWasPassed(t *testing.T) {
	resetGetFlags(t)
	getDefault = "fallback"
	v, ok := defaultValueFor(&projectfile.Document{}, fieldpath.Path{}, true)
	assert.True(t, ok)
	assert.Equal(t, "fallback", v)

	getDefault = ""
	v, ok = defaultValueFor(&projectfile.Document{}, fieldpath.Path{}, true)
	assert.True(t, ok, "an empty --default is still a passed --default")
	assert.Equal(t, "", v)
}

// TestDefaultValueForReadsTheDocumentDefault pins that --or-default consults the document, which is how a
// namespaced field falls back to a declared default rather than to nothing.
func TestDefaultValueForReadsTheDocumentDefault(t *testing.T) {
	resetGetFlags(t)
	getOrDefault = true
	p := mustPath(t, "identity.namespace")

	_, ok := defaultValueFor(&projectfile.Document{}, p, false)
	assert.False(t, ok, "a document with no value for the path has no default")

	_, ok = defaultValueFor(&projectfile.Document{}, p, true)
	assert.True(t, ok, "an explicit --default outranks the document")
}

func mustPath(t *testing.T, addr string) fieldpath.Path {
	t.Helper()
	p, err := fieldpath.Parse(addr)
	require.NoError(t, err)
	return p
}
