// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import "testing"

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
