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
	Short: "Interactively edit the per-user pf-cli configuration",
	Long: "Walks every section of $XDG_CONFIG_HOME/projectfile/cli.{yaml,toml,json},\n" +
		"pre-filling fields from the existing file when present, and from git config\n" +
		"(`user.name`, `user.email`, `user.signingKey`) for the small overlap it carries.\n" +
		"Each section can be submitted with Enter (accept current values) or cancelled\n" +
		"with Esc. Values set here are consumed by `pf-cli init`, generators (FUNDING/\n" +
		"SECURITY/CONTRIBUTING), and the git scanner — most of what the wizard captures\n" +
		"is what git config can’t carry (ORCID, funding URLs, private-host redaction, …).",
	RunE: func(_ *cobra.Command, _ []string) error {
		return usersetup.Run(usersetup.Options{Format: setupFormat})
	},
}

func init() {
	setupCmd.Flags().StringVarP(&setupFormat, "format", "f", "",
		"output format: yaml, toml, json (default: prompt)")
	rootCmd.AddCommand(setupCmd)
}
