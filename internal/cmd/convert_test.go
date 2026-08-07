// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"kiota.ch/projectfile/core/v2/pkg/genlog"
	"kiota.ch/projectfile/core/v2/pkg/projectfile"
)

// minimalTOML / minimalYAML / minimalJSON each describe the same project at
// the minimal schema-valid shape — identity.{namespace,name} plus the
// encoding-specific discriminator. Keeping them in lock-step lets the
// round-trip test assert structural equivalence without bringing in golden
// files.
const (
	testProjectfileTOML = "projectfile.toml"
	testProjectfileYAML = "projectfile.yaml"
	testProjectfileJSON = "projectfile.json"

	minimalTOML = `#:schema https://projectfile.org/schema/v1.json
spec_version = "1"
kind = "SoftwareSourceCode"

[identity]
namespace = "org.example"
name = "demo"
`

	minimalYAML = `---
$schema: https://projectfile.org/schema/v1.json
kind: SoftwareSourceCode
identity:
  namespace: org.example
  name: demo
`

	minimalJSON = `{
  "$schema": "https://projectfile.org/schema/v1.json",
  "kind": "SoftwareSourceCode",
  "identity": {
    "namespace": "org.example",
    "name": "demo"
  }
}
`
)

// resetConvertFlags re-zeroes the package-level cobra globals so a test that
// flips --force or --delete-source doesn't bleed into the next case.
func resetConvertFlags(t *testing.T) {
	t.Helper()
	convertForce = false
	convertDeleteSource = false
}

// runConvertCmd drives the convert command in isolation. Cobra subcommand
// dispatch only works when Execute runs on the root, so we prepend the
// "convert" verb and SetArgs on rootCmd; calling Execute on the subcommand
// directly re-parses from scratch and falls through to the root's help.
func runConvertCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	resetConvertFlags(t)

	// Silence genlog status lines during tests; restore on exit.
	genlog.SetOutput(&bytes.Buffer{})
	t.Cleanup(func() { genlog.SetOutput(os.Stderr) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs(append([]string{"convert"}, args...))
	err := rootCmd.Execute()
	return buf.String(), err
}

func writeFile(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
	return p
}

// TestConvertHappyPaths exercises every legal direction across the three
// encoders. Each case is independent and runs in its own temp dir.
func TestConvertHappyPaths(t *testing.T) {
	cases := []struct {
		name     string
		from, to string
		srcFile  string
		srcBody  string
		wantOut  string
	}{
		{"toml to yaml", fmtTOML, fmtYAML, testProjectfileTOML, minimalTOML, testProjectfileYAML},
		{"toml to json", fmtTOML, fmtJSON, testProjectfileTOML, minimalTOML, testProjectfileJSON},
		{"yaml to toml", fmtYAML, fmtTOML, testProjectfileYAML, minimalYAML, testProjectfileTOML},
		{"yaml to json", fmtYAML, fmtJSON, testProjectfileYAML, minimalYAML, testProjectfileJSON},
		{"json to toml", fmtJSON, fmtTOML, testProjectfileJSON, minimalJSON, testProjectfileTOML},
		{"json to yaml", fmtJSON, fmtYAML, testProjectfileJSON, minimalJSON, testProjectfileYAML},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, dir, c.srcFile, c.srcBody)

			_, err := runConvertCmd(t, c.from, c.to, dir)
			if err != nil {
				t.Fatalf("convert %s->%s: %v", c.from, c.to, err)
			}

			out := filepath.Join(dir, c.wantOut)
			if _, err := os.Stat(out); err != nil {
				t.Fatalf("expected output %s to exist: %v", out, err)
			}
			// Source must still be present by default.
			src := filepath.Join(dir, c.srcFile)
			if _, err := os.Stat(src); err != nil {
				t.Fatalf("expected source %s to be kept: %v", src, err)
			}
		})
	}
}

// TestConvertSameEncoderRejected covers yaml↔yml: both spellings resolve to
// the same encoder, so the command refuses to act rather than perform a
// disguised rename.
func TestConvertSameEncoderRejected(t *testing.T) {
	cases := []struct {
		name     string
		from, to string
		srcFile  string
	}{
		{"yaml to yml", fmtYAML, fmtYML, "projectfile.yaml"},
		{"yml to yaml", fmtYML, fmtYAML, "projectfile.yml"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, dir, c.srcFile, minimalYAML)

			_, err := runConvertCmd(t, c.from, c.to, dir)
			if err == nil {
				t.Fatalf("expected error for same-encoder convert, got nil")
			}
			if !strings.Contains(err.Error(), "same encoder") {
				t.Fatalf("expected 'same encoder' error, got %v", err)
			}
		})
	}
}

// TestConvertForceSemantics: pre-create the output file. The command must
// refuse without --force and succeed with it.
func TestConvertForceSemantics(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, testProjectfileTOML, minimalTOML)
	writeFile(t, dir, testProjectfileYAML, "stale: content\n")

	_, err := runConvertCmd(t, fmtTOML, fmtYAML, dir)
	if err == nil {
		t.Fatalf("expected refusal when output exists without --force")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected 'already exists' error, got %v", err)
	}

	_, err = runConvertCmd(t, "--force", fmtTOML, fmtYAML, dir)
	if err != nil {
		t.Fatalf("--force should have succeeded: %v", err)
	}
	body, _ := os.ReadFile(filepath.Join(dir, testProjectfileYAML))
	if strings.Contains(string(body), "stale: content") {
		t.Fatalf("--force did not overwrite the stale output")
	}
}

// TestConvertDeleteSource: --delete-source removes the source on success
// but must NOT remove it when the conversion fails upstream of the write.
func TestConvertDeleteSource(t *testing.T) {
	t.Run("success removes source", func(t *testing.T) {
		dir := t.TempDir()
		srcPath := writeFile(t, dir, testProjectfileTOML, minimalTOML)
		_, err := runConvertCmd(t, "--delete-source", fmtTOML, fmtYAML, dir)
		if err != nil {
			t.Fatalf("convert: %v", err)
		}
		if _, err := os.Stat(srcPath); !os.IsNotExist(err) {
			t.Fatalf("expected source removed, stat err = %v", err)
		}
	})

	t.Run("failure keeps source", func(t *testing.T) {
		dir := t.TempDir()
		// Schema-invalid: missing identity entirely.
		bad := `#:schema https://projectfile.org/schema/v1.json
spec_version = "1"
kind = "SoftwareSourceCode"
`
		srcPath := writeFile(t, dir, testProjectfileTOML, bad)
		_, err := runConvertCmd(t, "--delete-source", fmtTOML, fmtYAML, dir)
		if err == nil {
			t.Fatalf("expected validation error, got nil")
		}
		if _, statErr := os.Stat(srcPath); statErr != nil {
			t.Fatalf("source should survive a failed conversion, stat err = %v", statErr)
		}
		if _, statErr := os.Stat(filepath.Join(dir, testProjectfileYAML)); !os.IsNotExist(statErr) {
			t.Fatalf("output should not exist after failed validation, stat err = %v", statErr)
		}
	})
}

// TestConvertValidationGate: feed a schema-invalid fixture, expect failure
// before any output is written.
func TestConvertValidationGate(t *testing.T) {
	dir := t.TempDir()
	// Missing the required identity object.
	writeFile(t, dir, testProjectfileTOML, "spec_version = \"1\"\n")

	_, err := runConvertCmd(t, fmtTOML, fmtJSON, dir)
	if err == nil {
		t.Fatalf("expected schema validation failure, got nil")
	}
	if !strings.Contains(err.Error(), "schema validation") {
		t.Fatalf("expected validation-related error, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, testProjectfileJSON)); !os.IsNotExist(statErr) {
		t.Fatalf("no output file should be written on validation failure")
	}
}

// TestConvertSourceNotFound: explicit-format means no DetectPath fallback,
// so a missing source file is a hard error.
func TestConvertSourceNotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := runConvertCmd(t, fmtTOML, fmtYAML, dir)
	if err == nil {
		t.Fatalf("expected error for missing source")
	}
	if !strings.Contains(err.Error(), "source not found") {
		t.Fatalf("expected 'source not found' error, got %v", err)
	}
}

// TestConvertUnknownFormat: bad tokens fail at the dispatch step with a
// listing of supported formats.
func TestConvertUnknownFormat(t *testing.T) {
	dir := t.TempDir()
	_, err := runConvertCmd(t, "xml", "yaml", dir)
	if err == nil || !strings.Contains(err.Error(), "unknown source format") {
		t.Fatalf("expected unknown source-format error, got %v", err)
	}
	_, err = runConvertCmd(t, "toml", "xml", dir)
	if err == nil || !strings.Contains(err.Error(), "unknown target format") {
		t.Fatalf("expected unknown target-format error, got %v", err)
	}
}

// TestConvertRoundTrip walks toml→yaml→json→toml and asserts the final
// ToMap()-form is structurally identical to the original. This is the
// data-loss safety net for the whole conversion pipeline.
func TestConvertRoundTrip(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, testProjectfileTOML, minimalTOML)

	origDoc, err := projectfile.ReadFromPath(filepath.Join(dir, testProjectfileTOML))
	if err != nil {
		t.Fatalf("read original: %v", err)
	}
	orig := origDoc.ToMap()

	steps := []struct{ from, to string }{
		{fmtTOML, fmtYAML},
		{fmtYAML, fmtJSON},
		{fmtJSON, fmtTOML}, // overwrites the original .toml with the round-tripped result
	}
	for _, s := range steps {
		if _, err := runConvertCmd(t, "--force", s.from, s.to, dir); err != nil {
			t.Fatalf("convert %s->%s: %v", s.from, s.to, err)
		}
	}

	finalDoc, err := projectfile.ReadFromPath(filepath.Join(dir, testProjectfileTOML))
	if err != nil {
		t.Fatalf("read final: %v", err)
	}
	final := finalDoc.ToMap()

	if !reflect.DeepEqual(orig, final) {
		t.Fatalf("round-trip drift:\nbefore: %#v\nafter:  %#v", orig, final)
	}
}

// TestConvertDoesNotMaterialiseIncludes verifies that convert reads the BASE
// document (no include resolution) so include data stays in the include file
// and the `includes` list survives the encoding change.
func TestConvertDoesNotMaterialiseIncludes(t *testing.T) {
	dir := t.TempDir()

	// Include carries license data.
	writeFile(t, dir, "shared.yaml",
		"---\nlicense:\n  spdx: Apache-2.0\n")

	// Base references the include and has identity only.
	writeFile(t, dir, testProjectfileYAML, `---
$schema: https://projectfile.org/schema/v1.json
kind: SoftwareSourceCode
identity:
  namespace: org.example
  name: demo
includes:
  - shared.yaml
`)

	if _, err := runConvertCmd(t, fmtYAML, fmtTOML, dir); err != nil {
		t.Fatalf("convert yaml->toml: %v", err)
	}

	// Read the output as BASE (no include resolution).
	outDoc, err := projectfile.ReadBaseFromPath(filepath.Join(dir, testProjectfileTOML))
	if err != nil {
		t.Fatalf("read converted: %v", err)
	}

	assert.Nil(t, outDoc.License,
		"include-sourced license must NOT appear in converted base file")
	assert.Equal(t, []string{"shared.yaml"}, outDoc.Includes,
		"includes list must survive conversion")
}
