// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"kiota.ch/projectfile/core/v2/pkg/genlog"
	"kiota.ch/projectfile/core/v2/pkg/pflock"
	"kiota.ch/projectfile/core/v2/pkg/projectfile"
)

var (
	optimizeDryRun   bool
	optimizePathFile string
)

var optimizeCmd = &cobra.Command{
	Use:   "optimize [directory]",
	Short: "Remove local fields that duplicate include values",
	Long: "Remove values your includes already provide.\n" +
		"Keeps includes, $schema and spec_version.",
	Example: "  pf-cli optimize\n" +
		"  pf-cli optimize --dry-run",
	Aliases: []string{"opt"},
	Args:    usageArgs(cobra.MaximumNArgs(1)),
	RunE:    runOptimize,
}

func runOptimize(_ *cobra.Command, args []string) error {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}

	pfPath, err := resolveProjectfilePathForDir(dir, optimizePathFile)
	if err != nil {
		return err
	}

	return pflock.WithLock(pfPath, func() error {
		return runOptimizeInner(pfPath)
	})
}

func resolveProjectfilePathForDir(dir, explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	return projectfile.DetectPath(dir)
}

func runOptimizeInner(pfPath string) error {
	baseDir := filepath.Dir(pfPath)

	rawBase, err := projectfile.ReadRawBaseFromPath(pfPath)
	if err != nil {
		return fmt.Errorf("read base: %w", err)
	}

	// --sorted reorders the includes set so the written file is stable and
	// diff-friendly. Includes are an unordered set, so reordering is safe;
	// only the includes field is touched, every other list keeps its order.
	if projectfile.YAMLOutputSortedEnabled() {
		projectfile.SortIncludes(rawBase)
	}

	// Prune include entries a sibling already provides. validate only warns
	// about them; optimize drops them so the next validate runs quiet.
	redundant := projectfile.RedundantIncludes(rawBase, baseDir, pfPath, readOpts())
	pruned := dropRedundantIncludes(rawBase, redundant)

	mergedIncludes, err := projectfile.ResolveIncludesOnly(rawBase, baseDir, readOpts())
	if err != nil {
		return fmt.Errorf("resolve includes: %w", err)
	}

	if len(mergedIncludes) == 0 && len(pruned) == 0 {
		genlog.Plain("no includes declared — nothing to optimize")
		if projectfile.YAMLOutputSortedEnabled() && !optimizeDryRun {
			doc := projectfile.FromMap(rawBase)
			return projectfile.Write(doc, pfPath)
		}
		return nil
	}

	removed := projectfile.StripRedundant(rawBase, mergedIncludes)

	if len(removed) == 0 && len(pruned) == 0 {
		genlog.Plain("already optimized — no redundant fields found")
		if projectfile.YAMLOutputSortedEnabled() && !optimizeDryRun {
			doc := projectfile.FromMap(rawBase)
			return projectfile.Write(doc, pfPath)
		}
		return nil
	}

	for _, p := range removed {
		genlog.Debug(fmt.Sprintf("  removed %s", p))
	}
	for _, r := range pruned {
		genlog.Debug(fmt.Sprintf("  pruned include %s (already provided by %s)", r.Ref, r.Via))
	}

	if optimizeDryRun {
		genlog.Success(fmt.Sprintf("(%d field(s) and %d include(s) would be removed — dry-run, not written)", len(removed), len(pruned)))
		return nil
	}

	doc := projectfile.FromMap(rawBase)
	if err := projectfile.Write(doc, pfPath); err != nil {
		return fmt.Errorf("write projectfile: %w", err)
	}

	genlog.Success(fmt.Sprintf("optimized: %d redundant field(s) removed, %d redundant include(s) pruned", len(removed), len(pruned)))
	return nil
}

// includesKey is the top-level document key holding the include list.
const includesKey = "includes"

// dropRedundantIncludes removes one list occurrence per RedundantIncludes
// entry from the top-level includes list. A verbatim duplicate keeps its
// first listing, a transitively provided entry goes. Entries living outside
// the top-level list are left untouched. Returns the entries dropped.
func dropRedundantIncludes(raw map[string]any, redundant []projectfile.RedundantInclude) []projectfile.RedundantInclude {
	if len(redundant) == 0 {
		return nil
	}
	list, ok := raw[includesKey].([]any)
	if !ok {
		return nil
	}
	pending := make(map[string]int, len(redundant))
	for _, r := range redundant {
		pending[r.Ref]++
	}
	kept := make([]any, 0, len(list))
	for _, item := range list {
		if ref := includeRef(item); ref != "" && pending[ref] > 0 {
			pending[ref]--
			continue
		}
		kept = append(kept, item)
	}
	raw[includesKey] = kept
	return redundant
}

// includeRef is the path or URL of an includes item in string or {url, sha256} form.
func includeRef(item any) string {
	if m, ok := item.(map[string]any); ok {
		item = m["url"]
	}
	s, _ := item.(string)
	return s
}

func init() {
	optimizeCmd.Flags().BoolVarP(&optimizeDryRun, "dry-run", "n", false,
		"list removals without saving")
	optimizeCmd.Flags().StringVar(&optimizePathFile, "path-file", "",
		"explicit projectfile path (skips detection)")
	groupCmd(optimizeCmd, "maintain", "optimize")
}
