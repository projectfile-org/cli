// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"testing"
)

// flagForce is the one letter this file defends; spelling it once keeps
// goconst quiet across the assertions below.
const flagForce = "force"

// TestShortFlagFIsForceEverywhere pins the one-letter namespace: -f means
// --force wherever a --force exists, and nothing else claims the letter.
// A command may only add -f back by taking on a --force flag.
func TestShortFlagFIsForceEverywhere(t *testing.T) {
	forceOwners := map[string]bool{cmdConvert: true}
	for _, cmd := range rootCmd.Commands() {
		if f := cmd.Flags().ShorthandLookup("f"); f != nil {
			if !forceOwners[cmd.Name()] {
				t.Errorf("pf-cli %s: -f is %s, want it free (only --%s may use -f)", cmd.Name(), f.Name, flagForce)
				continue
			}
			if f.Name != flagForce {
				t.Errorf("pf-cli %s: -f is %s, want %s", cmd.Name(), f.Name, flagForce)
			}
		}
		if forceOwners[cmd.Name()] {
			if f := cmd.Flags().Lookup(flagForce); f == nil {
				t.Errorf("pf-cli %s: classified as a force owner but has no --%s", cmd.Name(), flagForce)
			}
		}
	}
}

// TestPathFileHasNoShortForm keeps the long spelling, so the flag's meaning
// never changes with the command it sits on.
func TestPathFileHasNoShortForm(t *testing.T) {
	for _, cmd := range rootCmd.Commands() {
		f := cmd.Flags().Lookup("path-file")
		if f == nil {
			continue
		}
		if f.Shorthand != "" {
			t.Errorf("pf-cli %s: --path-file carries shorthand %q, want none", cmd.Name(), f.Shorthand)
		}
	}
}

// TestFormatShortFormIsUniform pins the same rule one level down: a short form
// for --format either exists on every command or on none.
func TestFormatShortFormIsUniform(t *testing.T) {
	withShort, without := []string{}, []string{}
	for _, cmd := range rootCmd.Commands() {
		f := cmd.Flags().Lookup("format")
		if f == nil {
			continue
		}
		if f.Shorthand == "" {
			without = append(without, cmd.Name())
			continue
		}
		withShort = append(withShort, cmd.Name())
	}
	if len(withShort) > 0 && len(without) > 0 {
		t.Errorf("--format has a short form on %v but not on %v", withShort, without)
	}
}

// TestConvertAndCacheWarmKeepShortF is the companion: the flags that DO own
// the letter still answer it, so the reallocation is visible and deliberate.
func TestConvertAndCacheWarmKeepShortF(t *testing.T) {
	f := convertCmd.Flags().ShorthandLookup("f")
	if f == nil || f.Name != flagForce {
		t.Fatalf("pf-cli convert -f is %v, want %s", f, flagForce)
	}
	w := cacheWarmCmd.Flags().ShorthandLookup("f")
	if w == nil || w.Name != flagForce {
		t.Fatalf("pf-cli cache warm -f is %v, want %s", w, flagForce)
	}
}
