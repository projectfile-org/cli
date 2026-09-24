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
	"kiota.ch/projectfile/core/v2/pkg/userconfig"
)

// withoutUserConfig empties the user-config layer for a deterministic test:
// SetIgnored resets the load cache and makes Load return a zero Config.
func withoutUserConfig(t *testing.T) {
	t.Helper()
	userconfig.SetIgnored(true)
	t.Cleanup(func() { userconfig.SetIgnored(false) })
}

// Shared fixture values, kept as constants so the linter does not read five
// copies of the same literal as five separate magic strings.
const initTestNamespace = "org.example"

func TestBasicInitNonInteractive(t *testing.T) {
	dir := t.TempDir()
	var out strings.Builder
	err := basicInit(&out, strings.NewReader(""), dir, basicOptions{
		Format:         fmtYAML,
		NonInteractive: true,
		Namespace:      initTestNamespace,
		Name:           "sample",
		License:        "MIT",
	})
	require.NoError(t, err)
	raw, _, err := projectfile.ReadRawWithOptions(dir, readOpts())
	require.NoError(t, err)
	identity, _ := raw["identity"].(map[string]any)
	assert.Equal(t, initTestNamespace, identity["namespace"])
	assert.Equal(t, "sample", identity["name"])
}

func TestBasicInitRefusesExisting(t *testing.T) {
	dir := t.TempDir()
	doc := &projectfile.Document{Kind: "SoftwareSourceCode"}
	doc.Identity.Namespace = initTestNamespace
	doc.Identity.Name = "existing-proj"
	require.NoError(t, projectfile.Write(doc, filepath.Join(dir, "projectfile.yaml")))
	var out strings.Builder
	err := basicInit(&out, strings.NewReader(""), dir, basicOptions{NonInteractive: true})
	assert.ErrorContains(t, err, "already exists")
}

func TestBasicInitMissingFields(t *testing.T) {
	withoutUserConfig(t)
	dir := filepath.Join(t.TempDir(), "proj")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	var out strings.Builder
	err := basicInit(&out, strings.NewReader(""), dir, basicOptions{NonInteractive: true})
	assert.ErrorContains(t, err, "identity.namespace")
}

func TestBasicInitInteractivePrompts(t *testing.T) {
	withoutUserConfig(t)
	dir := filepath.Join(t.TempDir(), "my-tool")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	in := strings.NewReader(initTestNamespace + "\n\nMy Tool\n")
	var out strings.Builder
	err := basicInit(&out, in, dir, basicOptions{Format: fmtYAML})
	require.NoError(t, err)
	raw, _, err := projectfile.ReadRawWithOptions(dir, readOpts())
	require.NoError(t, err)
	identity, _ := raw["identity"].(map[string]any)
	assert.Equal(t, initTestNamespace, identity["namespace"])
	assert.Equal(t, "my-tool", identity["name"])
	assert.Equal(t, "My Tool", identity["title"])
	assert.Contains(t, out.String(), "identity.namespace")
}

func TestDelegateInitArgv(t *testing.T) {
	bin := t.TempDir()
	script := "#!/bin/sh\necho delegated\n"
	require.NoError(t, os.WriteFile(filepath.Join(bin, "pf-bridge-init"), []byte(script), 0o755))
	t.Setenv("PATH", bin)
	oldArgs := os.Args
	os.Args = []string{"pf-cli", "init", "--format", fmtYAML, "."}
	defer func() { os.Args = oldArgs }()
	var gotArgv []string
	oldExec := execBin
	execBin = func(_ string, argv []string, _ []string) error {
		gotArgv = argv
		return nil
	}
	defer func() { execBin = oldExec }()
	require.NoError(t, delegateInit())
	assert.Equal(t, []string{"pf-bridge-init", "--format", fmtYAML, "."}, gotArgv)
}

func TestDelegateInitAbsent(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	assert.ErrorIs(t, delegateInit(), errNoBridge)
}
