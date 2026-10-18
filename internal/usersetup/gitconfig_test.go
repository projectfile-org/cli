// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package usersetup

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunGitConfigReturnsTrimmedValue pins that the value is read from stdout and trimmed, so a config
// file written with a trailing newline prefills the wizard without a phantom whitespace edit.
func TestRunGitConfigReturnsTrimmedValue(t *testing.T) {
	cmd := exec.Command("git", "config", "--get", "user.name")
	cmd.Dir = repoWithConfig(t, "[user]\n\tname = Ada Lovelace\n")
	assertGitConfig(t, cmd, "Ada Lovelace")
}

// TestRunGitConfigOnMissingKeyIsEmpty pins the best-effort contract: an unset key exits non-zero and must
// leave the field blank rather than block setup.
func TestRunGitConfigOnMissingKeyIsEmpty(t *testing.T) {
	cmd := exec.Command("git", "config", "--get", "user.signingKey")
	cmd.Dir = repoWithConfig(t, "[user]\n\tname = Ada Lovelace\n")
	assertGitConfig(t, cmd, "")
}

// TestRunGitConfigOnUnresolvableDirectoryIsEmpty pins that a cwd git cannot read yields the same empty
// string as an unset key, so a caller cannot tell the two apart and neither blocks the wizard.
func TestRunGitConfigOnUnresolvableDirectoryIsEmpty(t *testing.T) {
	cmd := exec.Command("git", "config", "--get", "user.name")
	cmd.Dir = filepath.Join(t.TempDir(), "absent")
	assertGitConfig(t, cmd, "")
}

// TestGitAccessorsReadTheirOwnKey pins that the three accessors address three distinct keys; a copy-paste
// that made all three read user.name would leave email and gpg-key silently empty in the wizard.
func TestGitAccessorsReadTheirOwnKey(t *testing.T) {
	t.Chdir(repoWithConfig(t, "[user]\n\tname = Ada Lovelace\n\temail = ada@example.com\n\tsigningKey = ABCDEF0123456789\n"))
	for _, tc := range []struct{ name, got, want string }{
		{"gitUserName", gitUserName(), "Ada Lovelace"},
		{"gitUserEmail", gitUserEmail(), "ada@example.com"},
		{"gitSigningKey", gitSigningKey(), "ABCDEF0123456789"},
	} {
		if tc.got != tc.want {
			t.Errorf("%s() = %q, want %q", tc.name, tc.got, tc.want)
		}
	}
}

// TestGitAccessorsAreEmptyWithoutARepository pins the fresh-machine path: no repository and no global
// config must not fail the wizard, it just prefills nothing.
func TestGitAccessorsAreEmptyWithoutARepository(t *testing.T) {
	isolateGit(t)
	t.Chdir(t.TempDir())
	for name, got := range map[string]string{"gitUserName": gitUserName(), "gitUserEmail": gitUserEmail(), "gitSigningKey": gitSigningKey()} {
		if got != "" {
			t.Errorf("%s() = %q, want empty outside a repository", name, got)
		}
	}
}

// TestGitAccessorsHonourLocalOverGlobal pins the deliberate absence of --global: a repo-local override is
// what `git commit` would attach, so the wizard must prefill the same address rather than disagree.
func TestGitAccessorsHonourLocalOverGlobal(t *testing.T) {
	dir := repoWithConfig(t, "[user]\n\tname = Ada Lovelace\n\temail = ada@example.com\n")
	if out, err := exec.Command("git", "-C", dir, "config", "--local", "user.email", "local@example.com").CombinedOutput(); err != nil {
		t.Fatalf("git config --local: %v\n%s", err, out)
	}
	t.Chdir(dir)
	if got := gitUserEmail(); got != "local@example.com" {
		t.Errorf("gitUserEmail() = %q, want the repo-local address", got)
	}
}

// repoWithConfig initialises a throwaway repository whose .git/config carries body, then returns its path.
func repoWithConfig(t *testing.T, body string) string {
	t.Helper()
	isolateGit(t)
	dir := t.TempDir()
	if out, err := exec.Command("git", "-C", dir, "init", "--quiet").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	if err := os.WriteFile(filepath.Join(dir, ".git", "config"), []byte(body), 0o600); err != nil {
		t.Fatalf("write fixture config: %v", err)
	}
	return dir
}

// isolateGit clears every inherited GIT_* variable for the rest of the test and points git at a
// throwaway global config. A commit hook exports GIT_DIR and GIT_INDEX_FILE, under which a fixture
// repo's `git init` re-initialises the REAL repository as bare and its commits land on the branch being
// committed; without the redirected global config the developer's own identity answers every lookup.
func isolateGit(t *testing.T) {
	t.Helper()
	for _, kv := range os.Environ() {
		if k, _, ok := strings.Cut(kv, "="); ok && strings.HasPrefix(k, "GIT_") {
			t.Setenv(k, "")
			if err := os.Unsetenv(k); err != nil {
				t.Fatalf("unset %s: %v", k, err)
			}
		}
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(home, "gitconfig-absent"))
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
}

func assertGitConfig(t *testing.T, cmd *exec.Cmd, want string) {
	t.Helper()
	if got := runGitConfig(cmd); got != want {
		t.Errorf("runGitConfig = %q, want %q", got, want)
	}
}
