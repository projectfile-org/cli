// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package usersetup

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
)

// runGitConfig runs a pre-built `git config` command, honouring the standard
// local → global → system precedence from the current working directory.
// The caller builds the command with literal arguments only — no caller value
// flows into exec.Command — so the subprocess launch is provably safe.
// Empty string on any failure: git missing, key unset, exec trouble — the
// wizard's prefill is best-effort and never blocks setup on the absence of git.
//
// We deliberately do NOT pass --global. A user editing setup from inside a
// repo that overrides user.email locally should see the same address `git
// commit` would attach; --global would silently disagree.
func runGitConfig(cmd *exec.Cmd) string {
	// Suppress any credential / GPG passphrase prompts git might attempt —
	// setup must never block on a TTY popup, same rule as scanners/git.run.
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return ""
	}
	return strings.TrimSpace(out.String())
}

// gitUserName / gitUserEmail / gitSigningKey are the typed accessors over the
// three git config keys the wizard prefills. Each builds its git invocation
// with literal arguments — the keys are compile-time constants, never derived
// from caller data.
func gitUserName() string {
	return runGitConfig(exec.Command("git", "config", "--get", "user.name"))
}

func gitUserEmail() string {
	return runGitConfig(exec.Command("git", "config", "--get", "user.email"))
}

func gitSigningKey() string {
	return runGitConfig(exec.Command("git", "config", "--get", "user.signingKey"))
}
