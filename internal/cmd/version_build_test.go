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
	// go build stamps VCS settings by shelling out to git. A linked worktree
	// carries a .git FILE pointing at the parent repository's gitdir, which a
	// container that mounts only the worktree tree cannot resolve — the build
	// then stamps nothing and the fallback text is correct, not a defect.
	if err := exec.Command("git", "rev-parse", "--absolute-git-dir").Run(); err != nil {
		t.Skipf("no resolvable git metadata for VCS stamping: %v", err)
	}
	bin := filepath.Join(t.TempDir(), "pf-cli")
	build := exec.Command("go", "build", "-o", bin, "../../cmd/pf-cli")
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
