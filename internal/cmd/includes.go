// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"kiota.ch/projectfile/core/v2/pkg/genlog"
	"kiota.ch/projectfile/core/v2/pkg/pflock"
	"kiota.ch/projectfile/core/v2/pkg/projectfile"
)

var (
	includesPinDryRun   bool
	includesPinPathFile string
)

var includesCmd = &cobra.Command{
	Use:   "includes",
	Short: "Manage the includes list",
	Args:  usageArgs(cobra.NoArgs),
}

var includesPinCmd = &cobra.Command{
	Use:   "pin [url…]",
	Short: "Pin remote includes to the SHA-256 of their current bytes",
	Long: "Rewrite each HTTP(S) include as {url, sha256}, fetching it fresh.\n" +
		"Readers then reject the include once its bytes change.\n" +
		"With no URL every remote include is pinned; a pinned one is re-pinned.",
	Example: "  pf-cli includes pin\n" +
		"  pf-cli includes pin https://example.org/base.yaml --dry-run",
	RunE: runIncludesPin,
}

func runIncludesPin(_ *cobra.Command, args []string) error {
	pfPath, err := resolveProjectfilePathForDir(".", includesPinPathFile)
	if err != nil {
		return err
	}
	return pflock.WithLock(pfPath, func() error {
		raw, err := projectfile.ReadRawBaseFromPath(pfPath)
		if err != nil {
			return fmt.Errorf("read base: %w", err)
		}
		pinned, err := projectfile.PinIncludes(raw, args, readOpts())
		if err != nil {
			return err
		}
		if len(pinned) == 0 {
			genlog.Plain(fmt.Sprintf("no HTTP(S) includes to pin in %s", pfPath))
			return nil
		}
		for _, e := range pinned {
			genlog.Plain(fmt.Sprintf("%s  %s", e.SHA256, e.Ref))
		}
		if includesPinDryRun {
			genlog.Success(fmt.Sprintf("(%d include(s) would be pinned — dry-run, not written)", len(pinned)))
			return nil
		}
		if err := projectfile.Write(projectfile.FromMap(raw), pfPath); err != nil {
			return fmt.Errorf("write projectfile: %w", err)
		}
		genlog.Success(fmt.Sprintf("pinned %d include(s) in %s", len(pinned), pfPath))
		return nil
	})
}

func init() {
	includesPinCmd.Flags().BoolVarP(&includesPinDryRun, "dry-run", "n", false,
		"print the digests without saving")
	includesPinCmd.Flags().StringVarP(&includesPinPathFile, "path-file", "f", "",
		"explicit projectfile path (skips detection)")
	includesCmd.AddCommand(includesPinCmd)
	rootCmd.AddCommand(includesCmd)
}
