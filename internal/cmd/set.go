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
	Long: "Replace the value at <path>. Creates intermediate maps as needed.\n" +
		"For object/array values, use --value-json; for comma-separated string\n" +
		"lists, use --csv. --create-only refuses to overwrite an existing value.\n" +
		"--dry-run prints the would-be write to stderr without touching disk.",
	Args: cobra.RangeArgs(1, 2),
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
	if err != nil {
		return err
	}

	if setDryRun {
		genlog.Plain(fmt.Sprintf("set %s = %v (dry-run, not written)", addr, value))
		return nil
	}
	if err := projectfile.Write(out, pfPath); err != nil {
		return fmt.Errorf("write projectfile: %w", err)
	}
	genlog.Plain(fmt.Sprintf("set %s = %v", addr, value))
	return nil
}

// resolveSetValue picks the value from the flags + positional. Exactly
// one source must be supplied: --value-json, --csv, or a bare positional.
// The triplet conflict is rejected up front so callers see "you said both"
// instead of silently preferring one over the others.
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
		raw := args[1]
		// Try JSON so bare `true`, `false`, integers and floats coerce to their
		// native Go types (bool, float64). String values that are not valid JSON
		// literals (e.g. "main", "v1.2.3") pass through unchanged as strings.
		var v any
		if err := json.Unmarshal([]byte(raw), &v); err == nil {
			return v, nil
		}
		return raw, nil
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
	setCmd.Flags().StringVar(&setValueJSON, "value-json", "", "value as JSON (required for objects/arrays)")
	setCmd.Flags().StringVar(&setCSV, "csv", "", "value as comma-separated string list")
	setCmd.Flags().BoolVar(&setCreateOnly, "create-only", false, "refuse to overwrite an existing value")
	setCmd.Flags().BoolVarP(&setDryRun, "dry-run", "n", false, "compute the write without persisting")
	setCmd.Flags().StringVarP(&setPathFile, "path-file", "f", "", "explicit projectfile path (skips detection)")
	rootCmd.AddCommand(setCmd)
}
