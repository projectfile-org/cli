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

func TestValidateNamesRepositoriesOriginRule(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{
			name: "none origin",
			body: "repositories:\n  - url: https://example.org/a\n    role: mirror\n  - url: https://example.org/b\n    role: archive\n",
			want: "2 entries, 0 marked role: origin, exactly one required",
		},
		{
			name: "two origin",
			body: "repositories:\n  - url: https://example.org/a\n    role: origin\n  - url: https://example.org/b\n    role: origin\n",
			want: "2 entries, 2 marked role: origin, exactly one required",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, dir, "projectfile.yaml", "schema: https://projectfile.org/schema/v1.json\n\nidentity:\n  namespace: org.example\n  name: demo\n\n"+tc.body)
			out, err := runValidateCmd(t, dir)
			if err == nil {
				t.Fatalf("broken origin cardinality validated clean: %s", out)
			}
			if !strings.Contains(out, tc.want) {
				t.Fatalf("got %q, want the count message %q", out, tc.want)
			}
			if !strings.Contains(out, "https://example.org/a") {
				t.Fatalf("got %q, want the entry URLs listed", out)
			}
			for _, raw := range []string{"maxItems", "max 1 items required"} {
				if strings.Contains(out, raw) {
					t.Fatalf("got %q, want the opaque schema wording replaced", out)
				}
			}
		})
	}
}

func TestValidateKeepsNonCardinalityRepositoriesFailures(t *testing.T) {
	dir := t.TempDir()
	// Two entries, exactly one origin — the cardinality rule holds, so a
	// malformed second entry must keep its own schema line.
	writeFile(t, dir, "projectfile.yaml", "$schema: https://projectfile.org/schema/v1.json\nidentity:\n  namespace: org.example\n  name: demo\n\nrepositories:\n  - url: https://example.org/a\n    role: origin\n  - not-a-map\n")
	out, err := runValidateCmd(t, dir)
	if err == nil {
		t.Fatalf("malformed entry validated clean: %s", out)
	}
	if strings.Contains(out, "exactly one required") {
		t.Fatalf("got %q, want the cardinality message absent", out)
	}
	if !strings.Contains(out, "/repositories/1") {
		t.Fatalf("got %q, want the per-entry failure kept", out)
	}
}
