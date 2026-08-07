// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"kiota.ch/projectfile/core/v2/pkg/fieldpath"
	"kiota.ch/projectfile/core/v2/pkg/genlog"
	"kiota.ch/projectfile/core/v2/pkg/pflock"
	"kiota.ch/projectfile/core/v2/pkg/projectfile"
)

var (
	delStrict   bool
	delDryRun   bool
	delPathFile string
)

var delCmd = &cobra.Command{
	Use:     "del <path>",
	Aliases: []string{"delete", "rm"},
	Short:   "Remove a field, list item, or map entry from projectfile",
	Long: "Delete the value at <path>. Idempotent by default (deleting an absent\n" +
		"path exits 0); --strict flips that to exit 1 when the path was already\n" +
		"missing. --dry-run reports what would change without writing.",
	Args: cobra.ExactArgs(1),
	RunE: runDel,
}

func runDel(_ *cobra.Command, args []string) error {
	addr := args[0]
	p, err := fieldpath.Parse(addr)
	if err != nil {
		return err
	}

	pfPath, err := resolveProjectfilePath(delPathFile)
	if err != nil {
		return err
	}

	return pflock.WithLock(pfPath, func() error {
		return runDelInner(addr, p, pfPath)
	})
}

func runDelInner(addr string, p fieldpath.Path, pfPath string) error {
	doc, err := readDocumentFromPath(pfPath)
	if err != nil {
		return err
	}

	out, existed, err := fieldpath.Delete(doc, p)
	if err != nil {
		return err
	}
	if !existed {
		if delStrict {
			return fmt.Errorf("del: %q not found (--strict)", addr)
		}
		genlog.Plain(fmt.Sprintf("del %s: no-op (absent)", addr))
		return nil
	}

	if delDryRun {
		genlog.Plain(fmt.Sprintf("del %s (dry-run, not written)", addr))
		return nil
	}
	if err := projectfile.Write(out, pfPath); err != nil {
		return fmt.Errorf("write projectfile: %w", err)
	}
	genlog.Plain(fmt.Sprintf("del %s", addr))
	return nil
}

func init() {
	delCmd.Flags().BoolVar(&delStrict, "strict", false, "exit 1 if the path was already absent")
	delCmd.Flags().BoolVarP(&delDryRun, "dry-run", "n", false, "compute the deletion without persisting")
	delCmd.Flags().StringVarP(&delPathFile, "path-file", "f", "", "explicit projectfile path (skips detection)")
	rootCmd.AddCommand(delCmd)
}
