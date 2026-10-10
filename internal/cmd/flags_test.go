// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"strings"
	"testing"
)

// TestShortFlagFIsForceEverywhere pins the one-letter namespace: -f means --force wherever a --force exists, and nothing else claims the letter — a command may only add -f back by taking on a --force flag.
func TestShortFlagFIsForceEverywhere(t *testing.T) {
	forceOwners := map[string]bool{
		cmdConvert: true,
	}
	for _, cmd := range rootCmd.Commands() {
		if f := cmd.Flags().ShorthandLookup("f"); f != nil {
			if !forceOwners[cmd.Name()] {
				t.Errorf("pf-cli %s: -f is %s, want it free (only --force may use -f)", cmd.Name(), f.Name)
				continue
			}
			if f.Name != "force" {
				t.Errorf("pf-cli %s: -f is %s, want force", cmd.Name(), f.Name)
			}
		}
		if forceOwners[cmd.Name()] {
			if f := cmd.Flags().Lookup("force"); f == nil {
				t.Errorf("pf-cli %s: classified as a force owner but has no --force", cmd.Name())
			}
		}
	}
}

// TestPathFileHasNoShortForm keeps the long spelling, so the flag's meaning never changes with the command it sits on.
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

// TestFormatShortFormIsUniform pins the same rule one level down: a short form for --format exists on every command or on none.
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
		t.Errorf("--format has a short form on %s but not on %s", strings.Join(withShort, ", "), strings.Join(without, ", "))
	}
}

// TestConvertForceKeepsShortF is the companion: the flags that DO own the letter still answer it, so the reallocation is visible and deliberate.
func TestConvertForceKeepsShortF(t *testing.T) {
	f := convertCmd.Flags().ShorthandLookup("f")
	if f == nil || f.Name != "force" {
		t.Fatalf("pf-cli convert -f is %v, want force", f)
	}
}
