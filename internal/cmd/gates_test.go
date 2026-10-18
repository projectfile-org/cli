// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"kiota.ch/projectfile/core/v2/pkg/genlog"
)

// TestValidateNamesADuplicatedInclude pins the include-hygiene guardrail: the same target listed twice is named, not silently deduped by the reader.
func TestValidateNamesADuplicatedInclude(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "shared.yaml", "identity:\n  license:\n    spdx: MIT\n")
	writeFile(t, dir, "projectfile.yaml",
		"$schema: https://projectfile.org/schema/v1.json\nspec_version: \"1\"\nkind: SoftwareSourceCode\nincludes:\n  - shared.yaml\n  - shared.yaml\nidentity:\n  namespace: org.example\n  name: demo\n")

	var logged bytes.Buffer
	genlog.SetOutput(&logged)
	t.Cleanup(func() { genlog.SetOutput(os.Stderr) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{cmdValidate, dir})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("a duplicated include is a warning, not a failure: %v", err)
	}
	if !strings.Contains(logged.String(), "include listed twice") {
		t.Errorf("the duplicate must be named, got %q", logged.String())
	}
}

// TestInitFallsBackToTheLocalScaffoldWithoutBridge pins the delegation contract: with pf-bridge-init off the PATH, `init DIR` scaffolds locally and writes the document.
func TestInitFallsBackToTheLocalScaffoldWithoutBridge(t *testing.T) {
	resetInitFlags(t)
	dir := t.TempDir()
	t.Setenv("PATH", t.TempDir()) // no pf-bridge-init on the PATH

	out, err := runRootCmd(t, cmdInit, dir, "--namespace", initTestNamespace, "--name", valDemo, "--license", "MIT", "--non-interactive")
	if err != nil {
		t.Fatalf("init %s: %v (%s)", dir, err, out)
	}
	body, readErr := os.ReadFile(filepath.Join(dir, "projectfile.yaml"))
	if readErr != nil {
		t.Fatalf("the scaffold must land in the named directory: %v", readErr)
	}
	for _, want := range []string{initTestNamespace, valDemo, "MIT"} {
		if !strings.Contains(string(body), want) {
			t.Errorf("scaffold missing %q:\n%s", want, body)
		}
	}
}

// resetInitFlags clears the package-level init flag globals so one case cannot leak into the next.
func resetInitFlags(t *testing.T) {
	t.Helper()
	saved := struct {
		format, ns, name, license string
		nonInteractive, noScan    bool
	}{initFormat, initNamespace, initName, initLicense, initNonInteractive, initNoScan}
	t.Cleanup(func() {
		initFormat, initNamespace, initName, initLicense = saved.format, saved.ns, saved.name, saved.license
		initNonInteractive, initNoScan = saved.nonInteractive, saved.noScan
	})
	initFormat, initNamespace, initName, initLicense = "", "", "", ""
	initNonInteractive, initNoScan = false, false
}
