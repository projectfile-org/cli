// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"kiota.ch/projectfile/core/v2/pkg/genlog"
	"kiota.ch/projectfile/core/v2/pkg/projectfile"
)

// cacheRoot returns the resolved on-disk path of the shared cache slot, so
// status/purge can show users the REAL location (not just the env-var name).
func cacheRoot() (string, error) {
	base, err := projectfile.XDGCacheDir()
	if err != nil {
		return "", err
	}
	return base + "/pf", nil
}

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage pf-cli’s local cache for offline use",
	Long: "pf-cli keeps a local copy of every HTTP include it fetches, so that\n" +
		"reading a projectfile works without a network connection once warmed.\n" +
		"\n" +
		"The cache lives at:\n" +
		"  ${XDG_CACHE_HOME:-~/.cache}/pf/\n" +
		"\n" +
		"This slot is shared with pf-bridge and pf-ci — a purge here clears the\n" +
		"cache they all read. SPDX license texts are warmed and read by pf-bridge;\n" +
		"run `pf-bridge cache` to manage them.",
}

var (
	cacheWarmForce bool
)

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
		st, err := projectfile.IncludesCacheStatusSummary()
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "cache location: %s\n", root)
		if st.Total == 0 {
			fmt.Fprintf(out, "includes: 0 cached\n")
			return nil
		}
		oldest := "—"
		if st.Stale > 0 {
			oldest = st.OldestAge.Truncate(time.Second).String()
		}
		fmt.Fprintf(out, "includes: %d cached (%d fresh, %d stale, oldest %s)\n",
			st.Total, st.Fresh, st.Stale, oldest)
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
		"By default, a cached include whose freshness window has not elapsed is\n" +
		"served from cache without a network round-trip; a stale entry is\n" +
		"revalidated with a conditional request (304 reuses the cached body).\n" +
		"\n" +
		"--force forces a conditional revalidation for every entry, regardless\n" +
		"of freshness.\n" +
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
		return warmIncludes(dir, cacheWarmForce)
	},
}

var cacheRefreshCmd = &cobra.Command{
	Use:   "refresh [directory]",
	Short: "Alias for `cache warm --force`",
	Long: "Force a conditional revalidation of every HTTP include referenced\n" +
		"by the projectfile in [directory]. Shorthand for `cache warm --force`.",
	Args: cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		if offlineFlag {
			return fmt.Errorf("cannot refresh cache in offline mode; remove --offline to proceed")
		}
		dir := "."
		if len(args) > 0 {
			dir = args[0]
		}
		return warmIncludes(dir, true)
	},
}

var cachePurgeCmd = &cobra.Command{
	Use:   "purge [url]",
	Short: "Delete cached HTTP includes (all, or one URL)",
	Long: "Without arguments, remove every HTTP include pf-cli has cached,\n" +
		"freeing the disk they use. With one argument, remove only the cached\n" +
		"entry for that URL (by hash). The next read that needs an include\n" +
		"fetches it fresh. This does not touch the projectfile or any other file.",
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		out := cmd.OutOrStdout()

		root, err := cacheRoot()
		if err != nil {
			return err
		}
		if len(args) == 1 {
			removed, err := projectfile.PurgeInclude(args[0])
			if err != nil {
				return fmt.Errorf("purge include: %w", err)
			}
			if removed {
				fmt.Fprintf(out, "purged %s\n", args[0])
			} else {
				fmt.Fprintf(out, "no cached entry for %s\n", args[0])
			}
			return nil
		}
		removed, err := projectfile.PurgeIncludes()
		if err != nil {
			return fmt.Errorf("purge includes: %w", err)
		}
		fmt.Fprintf(out, "purged %d cached include(s) from %s\n", removed, root)
		return nil
	},
}

func warmIncludes(dir string, force bool) error {
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
	includes := projectfile.AllHTTPIncludes(raw, dir, pfPath, projectfile.ReadOptions{ForceRefresh: force})
	if len(includes) == 0 {
		genlog.Plain("includes: no remote includes found in projectfile")
		return nil
	}

	warmed := 0
	for _, ref := range includes {
		if err := projectfile.WarmIncludeWithOptions(ref, projectfile.ReadOptions{ForceRefresh: force}); err != nil {
			genlog.Warn("include warm failed", "url", ref, "err", err.Error())
			continue
		}
		warmed++
	}
	genlog.Plain(fmt.Sprintf("includes: warmed %d/%d remote includes", warmed, len(includes)))
	return nil
}

func init() {
	cacheCmd.AddCommand(cacheStatusCmd)
	cacheCmd.AddCommand(cacheWarmCmd)
	cacheCmd.AddCommand(cacheRefreshCmd)
	cacheCmd.AddCommand(cachePurgeCmd)
	cacheWarmCmd.Flags().BoolVar(&cacheWarmForce, "force", false, "force conditional revalidation even when cache is fresh")
	rootCmd.AddCommand(cacheCmd)
}
