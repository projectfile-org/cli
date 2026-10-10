// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"testing"
)

// TestExitCodeTableValues pins the numeric map USAGE.md and the release notes promise callers keying on specific codes; an unrecognised error is a runtime failure.
func TestExitCodeTableValues(t *testing.T) {
	table := map[string]struct {
		err  error
		want int
	}{
		"usage":   {errUsage("bad flag"), exitUsage},
		"absent":  {errAbsent("no such field"), exitAbsent},
		"schema":  {errValidation("schema validation failed"), exitInvalidated},
		"runtime": {errDispatch{msg: "read /nope.yaml: no such file"}, exitFailure},
		"wrapped usage": {
			fmt.Errorf("get: %w", errUsage("identity.name: no such field")),
			exitUsage,
		},
		"wrapped schema": {
			fmt.Errorf("validate: %w", errValidation("schema validation failed")),
			exitInvalidated,
		},
	}
	if len(table) != 6 {
		t.Fatalf("table has %d rows, want the 6 documented classes", len(table))
	}
	for name, tc := range table {
		if got := exitCodeFor(tc.err); got != tc.want {
			t.Errorf("%s: exitCodeFor = %d, want %d", name, got, tc.want)
		}
	}
}

// TestDispatchErrorClassification keeps cobra's own dispatch failures in the usage class while every RunE failure keeps its own: a blanket re-class would turn a missing file into a usage mistake.
func TestDispatchErrorClassification(t *testing.T) {
	for _, msg := range []string{
		`unknown command "bogus" for "pf-cli"`,
		"unknown shorthand flag: 'Z' in -Z",
	} {
		if got := exitCodeFor(classifyDispatchError(rootCmd, errDispatch{msg: msg})); got != exitUsage {
			t.Errorf("dispatch error %q exits %d, want %d", msg, got, exitUsage)
		}
	}
	runtime := errDispatch{msg: "read /nope.yaml: no such file or directory"}
	if got := exitCodeFor(classifyDispatchError(rootCmd, runtime)); got != exitFailure {
		t.Errorf("a runtime failure re-classed to %d, want %d", got, exitFailure)
	}
}

// errDispatch carries the exact message cobra hands Execute().
type errDispatch struct{ msg string }

func (e errDispatch) Error() string { return e.msg }
