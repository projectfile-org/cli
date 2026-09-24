// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"github.com/spf13/cobra"

	"projectfile.org/projectfile/cli/internal/usersetup"
)

var setupFormat string

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Edit your per-user configuration interactively",
	Long: "Answer questions once; answers prefill future prompts.\n" +
		"Enter accepts, Esc skips a section.",
	Example: "  pf-cli setup\n" +
		"  pf-cli setup --format yaml",
	RunE: func(_ *cobra.Command, _ []string) error {
		return usersetup.Run(usersetup.Options{Format: setupFormat})
	},
}

func init() {
	setupCmd.Flags().StringVarP(&setupFormat, "format", "f", "",
		"output format: yaml, toml, json (default: prompt)")
	rootCmd.AddCommand(setupCmd)
}
