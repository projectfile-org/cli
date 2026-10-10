// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"errors"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Shared literals across the cmd tests, hoisted so goconst stays quiet: the
// same command names recur in the arity, flag and dispatch tables.
const (
	cmdAdd        = "add"
	cmdCache      = "cache"
	cmdCompletion = "completion"
	cmdConvert    = "convert"
	cmdDel        = "del"
	cmdGet        = "get"
	cmdIncludes   = "includes"
	cmdInit       = "init"
	cmdOptimize   = "optimize"
	cmdSet        = "set"
	cmdSetup      = "setup"
	cmdValidate   = "validate"
	keyIdentity   = "identity"
	keyName       = "name"
	valDemo       = "demo"
	flagNope      = "--nope"
)

// runRootCmd drives the shared root with args, capturing both streams. Flags
// are reset afterwards: pflag keeps a --help value across runs, so one help
// execution would silently turn every later one into a help print.
func runRootCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	resetChanged := func(c *cobra.Command) {
		c.Flags().Visit(func(f *pflag.Flag) { _ = f.Value.Set(f.DefValue) })
	}
	t.Cleanup(func() {
		rootCmd.SetArgs(nil)
		resetChanged(rootCmd)
		for _, c := range rootCmd.Commands() {
			resetChanged(c)
		}
	})
	return buf.String(), err
}

// TestArityErrorsNameCommandShapeAndFix pins the rewritten arity messages:
// command, expected shape (the Use line), argument count and the fix, instead
// of cobra's raw "accepts between 1 and 2 arg(s), received 0".
func TestArityErrorsNameCommandShapeAndFix(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{cmdSet}, "pf-cli set needs <path> [value] (got 0). Run pf-cli set --help."},
		{[]string{"add"}, "pf-cli add needs <path> [value…] (got 0). Run pf-cli add --help."},
		{[]string{"del"}, "pf-cli del needs <path> (got 0). Run pf-cli del --help."},
		{[]string{cmdConvert, "yaml"}, "pf-cli convert needs <from-format> <to-format> [directory] (got 1). Supported: json, toml, yaml, yml. Run pf-cli convert --help."},
		{[]string{cmdValidate, "a", "b"}, "pf-cli validate needs [directory] (got 2). Run pf-cli validate --help."},
		{[]string{"cache", "status", "x"}, "pf-cli cache status takes no arguments (got 1). Run pf-cli cache status --help."},
	} {
		out, err := runRootCmd(t, tc.args...)
		if err == nil {
			t.Errorf("%v: no error, output %q", tc.args, out)
			continue
		}
		if err.Error() != tc.want {
			t.Errorf("%v: got %q, want %q", tc.args, err.Error(), tc.want)
		}
	}
}

// TestArityErrorsAreUsageClass pins the exit-2 class: a rewritten arity error
// must surface as *usageError, like the other usage mistakes (see Execute).
func TestArityErrorsAreUsageClass(t *testing.T) {
	_, err := runRootCmd(t, "set")
	var ue *usageError
	if !errors.As(err, &ue) {
		t.Fatalf("arity error is %T, want *usageError (exit 2)", err)
	}
}

// TestFlagErrorsNameCommandAndFix pins the rewritten flag-parse failures: the
// failing command, pflag's reason and the fix, also in the usage exit class.
func TestFlagErrorsNameCommandAndFix(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"get", flagNope}, "pf-cli get: unknown flag: --nope. Run pf-cli get --help."},
		{[]string{flagNope}, "pf-cli: unknown flag: --nope. Run pf-cli --help."},
		{[]string{cmdConvert, flagNope}, "pf-cli convert: unknown flag: --nope. Run pf-cli convert --help."},
		{[]string{cmdValidate, "-Z"}, "pf-cli validate: unknown shorthand flag: 'Z' in -Z. Run pf-cli validate --help."},
	} {
		_, err := runRootCmd(t, tc.args...)
		if err == nil || err.Error() != tc.want {
			t.Errorf("%v: got %v, want %q", tc.args, err, tc.want)
			continue
		}
		var ue *usageError
		if !errors.As(err, &ue) {
			t.Errorf("%v: flag error is %T, want *usageError (exit 2)", tc.args, err)
		}
	}
}

// TestUsageArgsKeepsValidAritySilent guards the wrapper against rejecting valid
// input: a satisfied validator must return nil and let the command run.
func TestUsageArgsKeepsValidAritySilent(t *testing.T) {
	called := false
	fn := usageArgs(func(_ *cobra.Command, _ []string) error { called = true; return nil })
	getCmd.Args = fn
	t.Cleanup(func() { getCmd.Args = cobra.ArbitraryArgs })
	_, err := runRootCmd(t, "get", "identity.name", "--print-path")
	if err != nil {
		t.Fatalf("valid args failed: %v", err)
	}
	if !called {
		t.Fatal("the wrapped validator never ran")
	}
}
