// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"kiota.ch/projectfile/core/v2/pkg/fieldpath"
	"kiota.ch/projectfile/core/v2/pkg/genlog"
	"kiota.ch/projectfile/core/v2/pkg/pflock"
	"kiota.ch/projectfile/core/v2/pkg/projectfile"
)

var (
	addFields    []string
	addValueJSON string
	addAllowDup  bool
	addDryRun    bool
	addPathFile  string
)

var addCmd = &cobra.Command{
	Use:   "add <path> [value…]",
	Short: "Append items to a projectfile list",
	Long: "Append items to the list at <path>.\n" +
		"Items already present are skipped; --allow-duplicate\n" +
		"forces the append. Objects take --field k=v or --value-json.",
	Args: cobra.MinimumNArgs(1),
	RunE: runAdd,
}

func runAdd(_ *cobra.Command, args []string) error {
	addr := args[0]
	rest := args[1:]
	p, err := fieldpath.Parse(addr)
	if err != nil {
		return err
	}

	pfPath, err := resolveProjectfilePath(addPathFile)
	if err != nil {
		return err
	}

	values, err := collectAddValues(rest)
	if err != nil {
		return err
	}

	return pflock.WithLock(pfPath, func() error {
		return runAddInner(addr, p, values, pfPath)
	})
}

func runAddInner(addr string, p fieldpath.Path, values []any, pfPath string) error {
	doc, err := readDocumentFromPath(pfPath)
	if err != nil {
		return err
	}

	addedCount := 0
	skippedCount := 0
	for _, v := range values {
		next, added, err := fieldpath.Add(doc, p, v, addAllowDup)
		if err != nil {
			return err
		}
		doc = next
		if added {
			addedCount++
		} else {
			skippedCount++
		}
	}

	if addDryRun {
		genlog.Success(fmt.Sprintf("add %s: %d added, %d skipped (dry-run, not written)", addr, addedCount, skippedCount))
		return nil
	}
	if addedCount == 0 {
		genlog.Plain(fmt.Sprintf("add %s: %d skipped (already present)", addr, skippedCount))
		return nil
	}
	if err := projectfile.Write(doc, pfPath); err != nil {
		return fmt.Errorf("write projectfile: %w", err)
	}
	genlog.Success(fmt.Sprintf("add %s: %d added, %d skipped", addr, addedCount, skippedCount))
	return nil
}

// collectAddValues assembles the slice of values Add will iterate. Three
// input modes are accepted; mixing them is a usage error so the user is
// forced to pick a single intent per invocation.
//
//   - bare positionals (scalar additions: `add keywords rust wasm`)
//   - --field k=v repeated (build one object: `add repositories --field url=...`)
//   - --value-json '<json>' (one fully-specified object/value)
func collectAddValues(positional []string) ([]any, error) {
	hasPositional := len(positional) > 0
	hasFields := len(addFields) > 0
	hasJSON := addValueJSON != ""
	if !hasPositional && !hasFields && !hasJSON {
		return nil, errUsage("add requires at least one value (positional, --field, or --value-json)")
	}
	if hasJSON && (hasPositional || hasFields) {
		return nil, errUsage("add: --value-json is exclusive with positional values and --field")
	}
	if hasFields && hasPositional {
		return nil, errUsage("add: --field is exclusive with positional values")
	}
	switch {
	case hasJSON:
		var v any
		if err := json.Unmarshal([]byte(addValueJSON), &v); err != nil {
			return nil, fmt.Errorf("add: --value-json parse: %w", err)
		}
		return []any{v}, nil
	case hasFields:
		obj := map[string]any{}
		for _, f := range addFields {
			eq := strings.IndexByte(f, '=')
			if eq <= 0 {
				return nil, errUsage(fmt.Sprintf("--field requires k=v, got %q", f))
			}
			k := strings.TrimSpace(f[:eq])
			v := strings.TrimSpace(f[eq+1:])
			// CSV-shorthand for known string-array sub-fields (roles is the
			// motivating case). The split is conservative — only the
			// recognised set gets exploded so a literal comma in a URL or
			// label is preserved verbatim.
			if isStringListField(k) && strings.Contains(v, ",") {
				parts := strings.Split(v, ",")
				items := make([]any, 0, len(parts))
				for _, p := range parts {
					items = append(items, strings.TrimSpace(p))
				}
				obj[k] = items
			} else {
				obj[k] = v
			}
		}
		return []any{obj}, nil
	default:
		out := make([]any, len(positional))
		for i, s := range positional {
			out[i] = s
		}
		return out, nil
	}
}

// isStringListField names the per-object sub-fields whose --field <k>=<v>
// value should be split on commas into a string array (matching the
// "--field roles=author,maintainer" sugar called out in the plan). Kept
// to a closed set so a label/URL containing a comma is NOT silently
// exploded.
func isStringListField(k string) bool {
	switch k {
	case "roles", "owners":
		return true
	}
	return false
}

func init() {
	addCmd.Flags().StringArrayVar(&addFields, "field", nil, "object field as k=v (repeatable)")
	addCmd.Flags().StringVar(&addValueJSON, "value-json", "", "value or object as JSON")
	addCmd.Flags().BoolVar(&addAllowDup, "allow-duplicate", false, "append even if already present")
	addCmd.Flags().BoolVarP(&addDryRun, "dry-run", "n", false, "show the append without saving it")
	addCmd.Flags().StringVarP(&addPathFile, "path-file", "f", "", "explicit projectfile path (skips detection)")
	rootCmd.AddCommand(addCmd)
}
