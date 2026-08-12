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

// sinkFixtureTOML declares the two shapes the build plane has to tell apart: a
// sink whose reference IS a path prefix plus the image path (ghcr), and one
// whose template moves the path into a flattened name (flat). No `ci.image` and
// no container build — a project resolving a FOREIGN image need not publish one
// of its own, and `sink refs` must still answer.
const sinkFixtureTOML = `#:schema https://projectfile.org/schema/v1.json
spec_version = "1"
kind = "SoftwareSourceCode"

[identity]
namespace = "org.example"
name = "demo"

[org.projectfile.sinks.ghcr]
host = "ghcr.io"
owner = "damian-buho"
priority = 90

[org.projectfile.sinks.flat]
ref = "docker.io/damianbuho/${image.flatname}:${image.tag}"
priority = 80
`

// runSinkCmd drives the sink command in isolation, same Cobra-root pattern as
// runGetCmd. stdin is bound on the root so InOrStdin() reaches the subcommand.
func runSinkCmd(t *testing.T, stdin string, args ...string) (string, error) {
	t.Helper()
	sinkName, sinkBasename, sinkTag = "", "", ""
	genlog.SetOutput(&bytes.Buffer{})
	t.Cleanup(func() { genlog.SetOutput(os.Stderr) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetIn(strings.NewReader(stdin))
	rootCmd.SetArgs(append([]string{"sink"}, args...))
	err := rootCmd.Execute()
	return buf.String(), err
}

func sinkFixtureDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeFixture(t, dir, sinkFixtureTOML)
	return dir
}

// TestSinkRefsFlattenedBatch is the case the prefix model cannot express: no
// string P satisfies P + "/b19/ubuntu/resolute:dev" = the flattened reference,
// so composing each image is the only way a flattened sink is a legal source.
func TestSinkRefsFlattenedBatch(t *testing.T) {
	dir := sinkFixtureDir(t)

	out, err := runSinkCmd(t, "b19/ubuntu/resolute\tdev\nd9t/go-tools\tlatest\n",
		"refs", dir, "--sink", "flat")
	if err != nil {
		t.Fatalf("sink refs: %v", err)
	}
	want := "b19/ubuntu/resolute\tdev\tdocker.io/damianbuho/b19-ubuntu-resolute:dev\n" +
		"d9t/go-tools\tlatest\tdocker.io/damianbuho/d9t-go-tools:latest\n"
	if out != want {
		t.Fatalf("sink refs flat =\n%q\nwant\n%q", out, want)
	}
}

// TestSinkRefsEchoesCoordinates pins the contract the caller maps on: every row
// repeats the pair it was asked about, so a caller holding a dozen variables
// never has to match by line number.
func TestSinkRefsEchoesCoordinates(t *testing.T) {
	dir := sinkFixtureDir(t)

	out, err := runSinkCmd(t, "o9s/loki\t3.1\n", "refs", dir, "--sink", "ghcr")
	if err != nil {
		t.Fatalf("sink refs: %v", err)
	}
	want := "o9s/loki\t3.1\tghcr.io/damian-buho/o9s/loki:3.1\n"
	if out != want {
		t.Fatalf("sink refs ghcr = %q, want %q", out, want)
	}
}

// TestSinkRefsSkipsBlankLines keeps a caller that assembles its list from make
// variables from having to filter empty slots itself.
func TestSinkRefsSkipsBlankLines(t *testing.T) {
	dir := sinkFixtureDir(t)

	out, err := runSinkCmd(t, "\n  \no9s/loki\t3.1\n\n", "refs", dir, "--sink", "ghcr")
	if err != nil {
		t.Fatalf("sink refs: %v", err)
	}
	if lines := strings.Count(out, "\n"); lines != 1 {
		t.Fatalf("sink refs emitted %d rows for one coordinate: %q", lines, out)
	}
}

// TestSinkRefsRefusesTaglessLine is the guess this verb must never make: a
// reference tagged by default points at a different image than the caller
// holds, and it would push or pull silently.
func TestSinkRefsRefusesTaglessLine(t *testing.T) {
	dir := sinkFixtureDir(t)

	if _, err := runSinkCmd(t, "o9s/loki\n", "refs", dir, "--sink", "ghcr"); err == nil {
		t.Fatal("sink refs accepted a tagless line, want refusal")
	} else if !strings.Contains(err.Error(), "refusing to guess") {
		t.Fatalf("sink refs tagless error = %v, want a refusal to guess", err)
	}
}

// TestSinkRefsTagFlagFillsTaglessLine is the other half: one --tag for a whole
// batch is the normal case when every image shares the framework's version.
func TestSinkRefsTagFlagFillsTaglessLine(t *testing.T) {
	dir := sinkFixtureDir(t)

	out, err := runSinkCmd(t, "o9s/loki\n", "refs", dir, "--sink", "ghcr", "--tag", "dev")
	if err != nil {
		t.Fatalf("sink refs --tag: %v", err)
	}
	want := "o9s/loki\tdev\tghcr.io/damian-buho/o9s/loki:dev\n"
	if out != want {
		t.Fatalf("sink refs --tag = %q, want %q", out, want)
	}
}

// TestSinkRefsKeepsMatrixPlaceholder pins the passthrough the matrix layer
// depends on: {AXIS} carries no `$`, so composition must leave it for the layer
// that owns the matrix to substitute per cell.
func TestSinkRefsKeepsMatrixPlaceholder(t *testing.T) {
	dir := sinkFixtureDir(t)

	out, err := runSinkCmd(t, "b19/ubuntu/{B19_UBUNTU_SERIES}\tdev\n",
		"refs", dir, "--sink", "flat")
	if err != nil {
		t.Fatalf("sink refs matrix: %v", err)
	}
	want := "b19/ubuntu/{B19_UBUNTU_SERIES}\tdev\tdocker.io/damianbuho/b19-ubuntu-{B19_UBUNTU_SERIES}:dev\n"
	if out != want {
		t.Fatalf("sink refs matrix = %q, want %q", out, want)
	}
}

// TestSinkRefsUnknownSinkNamesTheDeclared is the Docker Hub typo guard at the
// CLI edge: a bare word naming no sink must fail loudly, and the message has to
// carry the names that would have worked.
func TestSinkRefsUnknownSinkNamesTheDeclared(t *testing.T) {
	dir := sinkFixtureDir(t)

	_, err := runSinkCmd(t, "o9s/loki\t3.1\n", "refs", dir, "--sink", "ghrc")
	if err == nil {
		t.Fatal("sink refs accepted an undeclared sink, want refusal")
	}
	for _, want := range []string{"ghrc", "ghcr", "flat"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("sink refs error %v does not name %q", err, want)
		}
	}
}

// TestSinkRefsRequiresSinkFlag keeps the required flag a usage error rather
// than a nil-sink compose.
func TestSinkRefsRequiresSinkFlag(t *testing.T) {
	dir := sinkFixtureDir(t)

	if _, err := runSinkCmd(t, "o9s/loki\t3.1\n", "refs", dir); err == nil {
		t.Fatal("sink refs ran with no --sink, want a usage error")
	}
}
