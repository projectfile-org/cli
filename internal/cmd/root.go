// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"kiota.ch/projectfile/core/v2/pkg/genlog"
	"kiota.ch/projectfile/core/v2/pkg/projectfile"
	"kiota.ch/projectfile/core/v2/pkg/userconfig"
)

var version = "unknown"

var rootCmd = &cobra.Command{
	Use:   "pf-cli [command]",
	Short: "Read and edit projectfile documents",
	Long: "pf-cli reads and edits a projectfile document (projectfile.yaml, .toml,\n" +
		"or .json) — the single file at a project root that carries its identity,\n" +
		"licence, authorship, and technical metadata.\n" +
		"\n" +
		"Common tasks:\n" +
		"  pf-cli get identity.namespace      read a field\n" +
		"  pf-cli set license.spdx MIT        change a field\n" +
		"  pf-cli add keywords rust wasm      append to a list\n" +
		"  pf-cli del keywords[0]             remove a field or list item\n" +
		"  pf-cli validate                    check the document is well-formed\n" +
		"  pf-cli convert yaml toml           switch file format\n" +
		"  pf-cli optimize                    drop fields an include already supplies\n" +
		"  pf-cli sink ref --sink ghcr        compose a publish destination’s reference\n" +
		"  pf-cli cache                       manage the local cache for offline use\n" +
		"  pf-cli setup                       edit your personal defaults\n" +
		"\n" +
		"To project the document onto package files, forges, or the repo\n" +
		"(bridge, forge, scan, init), use the pf-bridge binary.",
	Version:       version,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRun: func(_ *cobra.Command, _ []string) {
		genlog.SetQuiet(quietFlag)
		if !verboseFlag {
			if v, _ := strconv.ParseBool(os.Getenv("PF_CLI_VERBOSE")); v {
				verboseFlag = true
			}
		}
		genlog.SetVerbose(verboseFlag)
		userconfig.SetIgnored(ignoreUserConfigFlag)
		projectfile.SetYAMLOutputSorted(sortedFlag)
		if offlineFlag {
			genlog.Info("offline mode", "message", "network fetches disabled")
		}
	},
}

// quietFlag wires the root-level --quiet to genlog.Quiet. Persistent so it
// applies to every subcommand without each one redeclaring it.
var quietFlag bool

// verboseFlag wires the root-level --verbose to genlog.Verbose. Also
// activated by PF_CLI_VERBOSE=1. Persistent so it applies to every
// subcommand without each one redeclaring it.
var verboseFlag bool

// ignoreUserConfigFlag bypasses $XDG_CONFIG_HOME/projectfile/cli.* loading
// for this invocation. Persistent so it applies to every subcommand. Use
// case: reproducible CI runs, debugging "is this behavior coming from my
// personal config?", or generating output that matches a teammate's setup.
var ignoreUserConfigFlag bool

// offlineFlag disables all network fetches. Persistent so it applies to every
// subcommand. When set, SPDX lookups skip the upstream fetch, HTTP includes
// use the XDG cache only, and forge push refuses immediately.
var offlineFlag bool

// sortedFlag forces YAML output keys to be written in sorted (alphabetical)
// order. Persistent so it applies to every subcommand that writes YAML.
// When false (default), the key order from the existing file is preserved.
var sortedFlag bool

// failOnFlag is the parsed value of --fail-on. pflag calls Set during parse,
// so an invalid value is rejected at flag-parse time with a clean error before
// any command runs. Default FailOnError: a missing local include is a warning
// and is skipped (partial data > hard stop) so consumers like m6e-sync are not
// blocked while an include is being fixed upstream. Pass --fail-on=warning to
// restore hard-fail strictness.
type failOnFlagValue struct{ level projectfile.IncludeFailLevel }

func (f *failOnFlagValue) String() string {
	switch f.level {
	case projectfile.FailOnWarning:
		return "warning"
	default:
		return "error"
	}
}

func (f *failOnFlagValue) Set(s string) error {
	switch s {
	case "", "error":
		f.level = projectfile.FailOnError
	case "warning":
		f.level = projectfile.FailOnWarning
	default:
		return fmt.Errorf("invalid value %q for --fail-on (must be 'error' or 'warning')", s)
	}
	return nil
}

func (f *failOnFlagValue) Type() string { return "string" }

var failOnFlag = failOnFlagValue{level: projectfile.FailOnError}

// readOpts builds the shared ReadOptions from the persistent include-resolution
// flags (offline, fail-on). Every command that resolves a projectfile uses this
// so the two flags stay in sync across call sites.
func readOpts() projectfile.ReadOptions {
	return projectfile.ReadOptions{Offline: offlineFlag, FailOn: failOnFlag.level}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&quietFlag, "quiet", "q", false,
		"suppress info/decision-trace output; warnings and errors still print")
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false,
		"show operational log lines (file detection, includes, locks); also PF_CLI_VERBOSE=1")
	rootCmd.PersistentFlags().BoolVar(&ignoreUserConfigFlag, "ignore-user-config", false,
		"skip $XDG_CONFIG_HOME/projectfile/cli.* loading — run as if no personal config existed")
	rootCmd.PersistentFlags().BoolVar(&offlineFlag, "offline", false,
		"refuse all network fetches; use embedded and cached data only")
	rootCmd.PersistentFlags().BoolVar(&sortedFlag, "sorted", false,
		"write YAML keys in sorted (alphabetical) order; disable canvas round-trip key preservation")
	rootCmd.PersistentFlags().Var(&failOnFlag, "fail-on",
		"abort when an include-resolution problem reaches this severity: "+
			"'error' (default; a missing local include warns and is skipped) or "+
			"'warning' (a missing local include aborts the command)")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		// Usage mistakes (bad flags/args) exit 2, distinct from a runtime
		// failure's exit 1 — the standard CLI convention, so callers can tell
		// "you invoked me wrong" apart from "the operation failed".
		var ue *usageError
		if errors.As(err, &ue) {
			os.Exit(2)
		}
		os.Exit(1)
	}
}
