// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"kiota.ch/projectfile/core/v2/pkg/fieldpath"
	"kiota.ch/projectfile/core/v2/pkg/genlog"
	"kiota.ch/projectfile/core/v2/pkg/pflock"
	"kiota.ch/projectfile/core/v2/pkg/projectfile"
)

var (
	setValueJSON  string
	setCSV        string
	setCreateOnly bool
	setDryRun     bool
	setPathFile   string
)

var setCmd = &cobra.Command{
	Use:   "set <path> [value]",
	Short: "Write a value into a projectfile field",
	Long: "Replace the value at <path>.\n" +
		"A bare value is stored as text, exactly as typed.\n" +
		"Use --value-json or --csv for a typed value.\n" +
		"Missing sections are created.",
	Example: "  pf-cli set license.spdx MIT\n" +
		"  pf-cli set identity.version 1.10\n" +
		"  pf-cli set keywords --csv rust,wasm\n" +
		"  pf-cli set contacts --value-json '{\"email\":\"a@example.com\"}'",
	Args: usageArgs(cobra.RangeArgs(1, 2)),
	RunE: runSet,
}

func runSet(_ *cobra.Command, args []string) error {
	addr := args[0]
	p, err := fieldpath.Parse(addr)
	if err != nil {
		return err
	}

	value, err := resolveSetValue(args)
	if err != nil {
		return err
	}

	pfPath, err := resolveProjectfilePath(setPathFile)
	if err != nil {
		return err
	}

	return pflock.WithLock(pfPath, func() error {
		return runSetInner(addr, p, value, pfPath)
	})
}

func runSetInner(addr string, p fieldpath.Path, value any, pfPath string) error {
	doc, err := readDocumentFromPath(pfPath)
	if err != nil {
		return err
	}

	if setCreateOnly {
		if _, err := fieldpath.Resolve(doc, p); err == nil {
			return fmt.Errorf("set: %q already has a value (use without --create-only to overwrite)", addr)
		} else if !errors.Is(err, fieldpath.ErrNotFound) {
			return err
		}
	}

	out, err := fieldpath.Set(doc, p, value)
	if errors.Is(err, fieldpath.ErrTypeMismatch) {
		// A bare positional is text, so a refusal now means the FIELD wants
		// another type — the fix is the flag that spells it, not a quoting
		// dance around a value that is already text.
		return fmt.Errorf("%s was not changed: it cannot hold text (%v). To store a number, list or object, use --value-json: pf-cli set %s --value-json '%v'", addr, value, addr, value)
	}
	if err != nil {
		return err
	}

	if setDryRun {
		genlog.Success(fmt.Sprintf("set %s = %v (dry-run, not written)", addr, value))
		return nil
	}
	if err := writeProjectfile(out, pfPath); err != nil {
		return fmt.Errorf("write projectfile: %w", err)
	}
	genlog.Success(fmt.Sprintf("set %s = %v", addr, value))
	return nil
}

// resolveSetValue picks the value from the flags + positional. Exactly
// one source must be supplied: --value-json, --csv, or a bare positional.
// The triplet conflict is rejected up front so callers see "you said both"
// instead of silently preferring one over the others.
//
// A bare positional is a STRING, full stop. An earlier version ran it
// through a JSON parse, which silently rewrote what the user typed —
// `1.10` landed as `1.1`, `true` as a boolean, `null` as null. A version,
// a serial or a build id looks numeric and is not, and the document then
// carries a value the user never wrote. The JSON forms stay reachable
// through the flags that exist for them.
func resolveSetValue(args []string) (any, error) {
	sources := 0
	if len(args) >= 2 {
		sources++
	}
	if setValueJSON != "" {
		sources++
	}
	if setCSV != "" {
		sources++
	}
	if sources == 0 {
		return nil, errUsage("set requires a value (positional, --value-json, or --csv)")
	}
	if sources > 1 {
		return nil, errUsage("set: only one of positional, --value-json, --csv may be given")
	}
	switch {
	case setValueJSON != "":
		var v any
		if err := json.Unmarshal([]byte(setValueJSON), &v); err != nil {
			return nil, fmt.Errorf("set: --value-json parse: %w", err)
		}
		return v, nil
	case setCSV != "":
		parts := strings.Split(setCSV, ",")
		out := make([]any, 0, len(parts))
		for _, p := range parts {
			out = append(out, strings.TrimSpace(p))
		}
		return out, nil
	default:
		return args[1], nil
	}
}

func resolveProjectfilePath(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	path, err := projectfile.DetectPath(".")
	if err != nil {
		return "", err
	}
	return path, nil
}

func readDocumentFromPath(path string) (*projectfile.Document, error) {
	return projectfile.ReadBaseFromPath(path)
}

func init() {
	setCmd.Flags().StringVar(&setValueJSON, "value-json", "", "value as JSON (a bare value is text)")
	setCmd.Flags().StringVar(&setCSV, "csv", "", "value as a comma-separated string list")
	setCmd.Flags().BoolVar(&setCreateOnly, "create-only", false, "fail if the path already has a value")
	setCmd.Flags().BoolVarP(&setDryRun, "dry-run", "n", false, "show the write without saving it")
	setCmd.Flags().StringVar(&setPathFile, "path-file", "", "explicit projectfile path (skips detection)")
	groupCmd(setCmd, "write", "set")
}
