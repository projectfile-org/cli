// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"strings"
	"testing"
)

func TestVersionShortFlagIsCapitalV(t *testing.T) {
	if f := rootCmd.Flags().ShorthandLookup("V"); f == nil || f.Name != "version" {
		t.Fatalf("-V is not bound to --version: %+v", f)
	}
}

func TestResolveVersionKeepsReleaseStamp(t *testing.T) {
	saved := version
	t.Cleanup(func() { version = saved })
	version = "1.2.3"
	if got := resolveVersion(); got != "1.2.3" {
		t.Fatalf("got %q, want the stamped 1.2.3", got)
	}
}

func TestFailOnRejectsEmptyValue(t *testing.T) {
	var f failOnFlagValue
	if err := f.Set(""); err == nil {
		t.Fatal("empty --fail-on was accepted")
	}
}

func TestDroppedAliasesAndPrefixesAreUnknown(t *testing.T) {
	for _, name := range []string{"rm", "v", "val", "sett"} {
		if _, _, err := rootCmd.Find([]string{name}); err == nil || !strings.Contains(err.Error(), "unknown command") {
			t.Errorf("pf-cli %s: got %v, want unknown command", name, err)
		}
	}
}

func TestKeptAliasesResolve(t *testing.T) {
	for alias, want := range map[string]string{"lint": "validate", "delete": "del", "conv": "convert", "opt": "optimize", "scaffold": "init"} {
		c, _, err := rootCmd.Find([]string{alias})
		if err != nil || c.Name() != want {
			t.Errorf("pf-cli %s: got %v %v, want %s", alias, c, err, want)
		}
	}
}
