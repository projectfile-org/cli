// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"kiota.ch/projectfile/core/v2/pkg/genlog"
	"kiota.ch/projectfile/core/v2/pkg/projectfile"
)

// cacheApp is the per-binary slot this process owns under
// $XDG_CACHE_HOME/projectfile/. pf-cli only caches HTTP includes here — SPDX
// license texts are warmed and read by pf-bridge (see pf-bridge cache).
const cacheApp = "cli"

// cacheRoot returns the resolved on-disk path of this binary's cache slot, so
// status/purge can show users the REAL location (not just the env-var name).
func cacheRoot() (string, error) {
	base, err := projectfile.XDGCacheDir()
	if err != nil {
		return "", err
	}
	return base + "/projectfile/" + cacheApp, nil
}

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage pf-cli’s local cache for offline use",
	Long: "pf-cli keeps a local copy of every HTTP include it fetches, so that\n" +
		"reading a projectfile works without a network connection once warmed.\n" +
		"\n" +
		"The cache lives at:\n" +
		"  ${XDG_CACHE_HOME:-~/.cache}/projectfile/" + cacheApp + "/\n" +
		"\n" +
		"SPDX license texts are NOT cached here — they belong to pf-bridge.\n" +
		"Run `pf-bridge cache` to warm or purge them.",
}

var cacheStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show what is cached and where",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		out := cmd.OutOrStdout()

		root, err := cacheRoot()
		if err != nil {
			return err
		}
		includes := countCachedIncludes()

		fmt.Fprintf(out, "cache location: %s\n", root)
		fmt.Fprintf(out, "includes: %d cached\n", includes)
		return nil
	},
}

var cacheWarmCmd = &cobra.Command{
	Use:   "warm [directory]",
	Short: "Pre-fetch HTTP includes so pf-cli works offline",
	Long: "Download every HTTP include referenced by the projectfile in\n" +
		"[directory] (default: the current directory) and store it in the cache.\n" +
		"After warming, `pf-cli` reads those includes from disk even with\n" +
		"--offline set.\n" +
		"\n" +
		"If there is no projectfile in the directory, warm does nothing and\n" +
		"exits successfully (warming SPDX license texts is a pf-bridge job).",
	Args: cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		if offlineFlag {
			return fmt.Errorf("cannot warm cache in offline mode; remove --offline to proceed")
		}

		dir := "."
		if len(args) > 0 {
			dir = args[0]
		}
		return warmIncludes(dir)
	},
}

var cachePurgeCmd = &cobra.Command{
	Use:   "purge",
	Short: "Delete every cached HTTP include",
	Long: "Remove all HTTP includes pf-cli has cached, freeing the disk they\n" +
		"use. The next read that needs an include fetches it fresh. This does\n" +
		"not touch the projectfile or any other file.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		out := cmd.OutOrStdout()

		root, err := cacheRoot()
		if err != nil {
			return err
		}
		removed, err := projectfile.PurgeIncludes()
		if err != nil {
			return fmt.Errorf("purge includes: %w", err)
		}
		fmt.Fprintf(out, "purged %d cached include(s) from %s\n", removed, root)
		return nil
	},
}

func warmIncludes(dir string) error {
	pfPath, err := projectfile.DetectPath(dir)
	if err != nil {
		// No projectfile here is a legitimate no-op for warming includes, not a
		// failure: pf-cli's cache is includes-only, and a directory with no
		// projectfile simply has nothing to prefetch.
		genlog.Info("no projectfile found; nothing to warm", "dir", dir)
		return nil //nolint:nilerr // Intentional: missing projectfile is a valid no-op, not an error.
	}
	raw, err := projectfile.ReadRawBaseFromPath(pfPath)
	if err != nil {
		return fmt.Errorf("read projectfile: %w", err)
	}

	// Discover transitive HTTP includes: a remote fragment may itself include
	// another remote fragment, and the cache should prefetch the whole chain
	// so a subsequent --offline read finds every link present. AllHTTPIncludes
	// walks local includes too, so HTTP includes reached via a local fragment
	// are also warmed.
	includes := projectfile.AllHTTPIncludes(raw, dir, pfPath, projectfile.ReadOptions{})
	if len(includes) == 0 {
		genlog.Plain("includes: no remote includes found in projectfile")
		return nil
	}

	warmed := 0
	for _, ref := range includes {
		if err := projectfile.WarmInclude(ref); err != nil {
			genlog.Warn("include warm failed", "url", ref, "err", err.Error())
			continue
		}
		warmed++
	}
	genlog.Plain(fmt.Sprintf("includes: warmed %d/%d remote includes", warmed, len(includes)))
	return nil
}

func countCachedIncludes() int {
	dir, err := projectfile.IncludesCacheDir()
	if err != nil {
		return 0
	}
	f, err := os.Open(dir) // #nosec G304 -- path derived from XDG
	if err != nil {
		return 0
	}
	defer f.Close()
	names, err := f.Readdirnames(0)
	if err != nil {
		return 0
	}
	return len(names)
}

func init() {
	cacheCmd.AddCommand(cacheStatusCmd)
	cacheCmd.AddCommand(cacheWarmCmd)
	cacheCmd.AddCommand(cachePurgeCmd)
	rootCmd.AddCommand(cacheCmd)
}
