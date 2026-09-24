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
	Long: "Drop local fields that repeat an include value.\n" +
		"Keeps includes, $schema and spec_version.",
	Aliases: []string{"opt"},
	Args:    cobra.MaximumNArgs(1),
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

	mergedIncludes, err := projectfile.ResolveIncludesOnly(rawBase, baseDir, readOpts())
	if err != nil {
		return fmt.Errorf("resolve includes: %w", err)
	}

	if len(mergedIncludes) == 0 {
		genlog.Plain("no includes declared — nothing to optimize")
		if projectfile.YAMLOutputSortedEnabled() && !optimizeDryRun {
			doc := projectfile.FromMap(rawBase)
			return projectfile.Write(doc, pfPath)
		}
		return nil
	}

	removed := projectfile.StripRedundant(rawBase, mergedIncludes)

	if len(removed) == 0 {
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

	if optimizeDryRun {
		genlog.Success(fmt.Sprintf("(%d field(s) would be removed — dry-run, not written)", len(removed)))
		return nil
	}

	doc := projectfile.FromMap(rawBase)
	if err := projectfile.Write(doc, pfPath); err != nil {
		return fmt.Errorf("write projectfile: %w", err)
	}

	genlog.Success(fmt.Sprintf("optimized: %d redundant field(s) removed", len(removed)))
	return nil
}

func init() {
	optimizeCmd.Flags().BoolVarP(&optimizeDryRun, "dry-run", "n", false,
		"list removals without saving")
	optimizeCmd.Flags().StringVarP(&optimizePathFile, "path-file", "f", "",
		"explicit projectfile path (skips detection)")
	rootCmd.AddCommand(optimizeCmd)
}
