// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestVersionWithoutLdflags builds pf-cli the way `go install` does and expects a real version.
func TestVersionWithoutLdflags(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a binary")
	}
	bin := filepath.Join(t.TempDir(), "pf-cli")
	build := exec.Command("go", "build", "-o", bin, "../..")
	build.Env = withoutGitEnv(os.Environ())
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	out, err := exec.Command(bin, "--version").CombinedOutput()
	if err != nil {
		t.Fatalf("pf-cli --version: %v\n%s", err, out)
	}
	if got := strings.TrimSpace(string(out)); got == "" || strings.Contains(got, "unknown") {
		t.Fatalf("--version printed %q, want a version from the build info", got)
	}
}

// withoutGitEnv drops inherited GIT_* variables so a commit hook cannot redirect the build's VCS stamping.
func withoutGitEnv(env []string) []string {
	kept := make([]string, 0, len(env))
	for _, kv := range env {
		if !strings.HasPrefix(kv, "GIT_") {
			kept = append(kept, kv)
		}
	}
	return kept
}
