// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"kiota.ch/projectfile/core/v2/pkg/genlog"
)

// runValidateCmd drives validate against a directory, capturing both streams
// so the flat violation list and the returned error can be asserted together.
func runValidateCmd(t *testing.T, dir string) (string, error) {
	t.Helper()
	genlog.SetOutput(&bytes.Buffer{})
	t.Cleanup(func() { genlog.SetOutput(os.Stderr) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"validate", dir})
	err := rootCmd.Execute()
	return buf.String(), err
}

func TestValidateRejectsTOMLWithoutSpecVersion(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "projectfile.toml", "[identity]\nnamespace = \"org.example\"\nname = \"demo\"\n")
	out, err := runValidateCmd(t, dir)
	if err == nil {
		t.Fatalf("a TOML document without spec_version validated clean: %s", out)
	}
	if !strings.Contains(out, "spec_version") {
		t.Fatalf("got %q, want the discriminator key named", out)
	}
}

func TestValidateAcceptsTOMLWithSpecVersion(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "projectfile.toml", "spec_version = \"1\"\n\n[identity]\nnamespace = \"org.example\"\nname = \"demo\"\n")
	out, err := runValidateCmd(t, dir)
	if err != nil {
		t.Fatalf("document with spec_version = \"1\" failed: %s (%v)", out, err)
	}
}
