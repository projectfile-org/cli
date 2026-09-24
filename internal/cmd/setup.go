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
	Long: "Walk every config section, prefilling from the file\n" +
		"and git config. Enter accepts, Esc skips a section.",
	RunE: func(_ *cobra.Command, _ []string) error {
		return usersetup.Run(usersetup.Options{Format: setupFormat})
	},
}

func init() {
	setupCmd.Flags().StringVarP(&setupFormat, "format", "f", "",
		"output format: yaml, toml, json (default: prompt)")
	rootCmd.AddCommand(setupCmd)
}
