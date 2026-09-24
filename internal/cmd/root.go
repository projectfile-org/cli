// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"

	"kiota.ch/projectfile/core/v2/pkg/genlog"
	"kiota.ch/projectfile/core/v2/pkg/projectfile"
	"kiota.ch/projectfile/core/v2/pkg/userconfig"
)

var version = "unknown"

// projectfileYAML carries the root projectfile bytes main embeds at build time.
var projectfileYAML []byte

// SetProjectfileYAML hands the embedded projectfile to the help renderer.
func SetProjectfileYAML(b []byte) {
	projectfileYAML = b
}

const rootShort = "Read and edit projectfile documents"

// Help palette mirrors the setup wizard so help and prompts feel like one product.
var (
	helpHeading = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	helpCommand = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
)

// helpTemplate orders every help screen as Description, Commands, Flags, Examples.
const helpTemplate = `{{hdr "Usage:"}}{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

{{hdr "Aliases:"}}
  {{.NameAndAliases}}{{end}}{{if .HasAvailableSubCommands}}

{{hdr "Commands:"}}{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{cmd (rpad .Name .NamePadding)}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

{{hdr "Flags:"}}
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

{{hdr "Global Flags:"}}
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasExample}}

{{hdr "Examples:"}}
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`

// orderHelp pins the Description, Commands, Flags, Examples order onto one command.
func orderHelp(c *cobra.Command) {
	c.SetUsageTemplate(helpTemplate)
}

// registerHelpPalette exposes the lipgloss styles to the help template.
func registerHelpPalette() {
	cobra.AddTemplateFunc("hdr", helpHeading.Render)
	cobra.AddTemplateFunc("cmd", helpCommand.Render)
}

// rootLong builds the Description from the projectfile identity text baked in at build time.
func rootLong() string {
	desc := projectfileDescription()
	if desc == "" {
		desc = rootShort
	}
	return wrap80(desc) + "\n\nFile and forge sync (bridge, forge, scan, init): use pf-bridge."
}

// projectfileDescription reads identity.description.en out of the embedded projectfile.
func projectfileDescription() string {
	var doc map[string]any
	if err := yaml.Unmarshal(projectfileYAML, &doc); err != nil {
		return ""
	}
	identity, _ := doc["identity"].(map[string]any)
	description, _ := identity["description"].(map[string]any)
	en, _ := description["en"].(string)
	return strings.TrimSpace(en)
}

// wrap80 folds s to word boundaries so no help line exceeds 80 columns.
func wrap80(s string) string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return ""
	}
	var b strings.Builder
	line := words[0]
	for _, w := range words[1:] {
		if len(line)+1+len(w) > 80 {
			b.WriteString(line + "\n")
			line = w
			continue
		}
		line += " " + w
	}
	b.WriteString(line)
	return b.String()
}

var rootCmd = &cobra.Command{
	Use:   "pf-cli [command]",
	Short: rootShort,
	Long:  rootLong(),
	Example: "  pf-cli get identity.name\n" +
		"  pf-cli set license.spdx MIT\n" +
		"  pf-cli add keywords rust wasm\n" +
		"  pf-cli del keywords[0]\n" +
		"  pf-cli validate\n" +
		"  pf-cli convert yaml toml\n" +
		"  pf-cli optimize\n" +
		"  pf-cli cache status",
	Version:       version,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRun: func(_ *cobra.Command, _ []string) {
		genlog.SetQuiet(quietFlag)
		if !verboseFlag {
			if v, _ := strconv.ParseBool(os.Getenv("PF_CLI_VERBOSE")); v {
				verboseFlag = true
			}
		}
		genlog.SetVerbose(verboseFlag)
		userconfig.SetIgnored(ignoreUserConfigFlag)
		projectfile.SetYAMLOutputSorted(sortedFlag)
		if offlineFlag {
			genlog.Debug("offline mode", "message", "network fetches disabled")
		}
	},
}

// quietFlag wires the root-level --quiet to genlog.Quiet. Persistent so it
// applies to every subcommand without each one redeclaring it.
var quietFlag bool

// verboseFlag wires the root-level --verbose to genlog.Verbose. Also
// activated by PF_CLI_VERBOSE=1. Persistent so it applies to every
// subcommand without each one redeclaring it.
var verboseFlag bool

// ignoreUserConfigFlag bypasses $XDG_CONFIG_HOME/projectfile/cli.* loading
// for this invocation. Persistent so it applies to every subcommand. Use
// case: reproducible CI runs, debugging "is this behavior coming from my
// personal config?", or generating output that matches a teammate's setup.
var ignoreUserConfigFlag bool

// offlineFlag disables all network fetches. Persistent so it applies to every
// subcommand. When set, SPDX lookups skip the upstream fetch, HTTP includes
// use the XDG cache only, and forge push refuses immediately.
var offlineFlag bool

// sortedFlag forces YAML output keys to be written in sorted (alphabetical)
// order. Persistent so it applies to every subcommand that writes YAML.
// When false (default), the key order from the existing file is preserved.
var sortedFlag bool

// failOnFlag is the parsed value of --fail-on. pflag calls Set during parse,
// so an invalid value is rejected at flag-parse time with a clean error before
// any command runs. Default FailOnError: a missing local include is a warning
// and is skipped (partial data > hard stop) so consumers like m6e-sync are not
// blocked while an include is being fixed upstream. Pass --fail-on=warning to
// restore hard-fail strictness.
type failOnFlagValue struct{ level projectfile.IncludeFailLevel }

func (f *failOnFlagValue) String() string {
	switch f.level {
	case projectfile.FailOnWarning:
		return "warning"
	default:
		return "error"
	}
}

func (f *failOnFlagValue) Set(s string) error {
	switch s {
	case "", "error":
		f.level = projectfile.FailOnError
	case "warning":
		f.level = projectfile.FailOnWarning
	default:
		return fmt.Errorf("invalid value %q for --fail-on (must be 'error' or 'warning')", s)
	}
	return nil
}

func (f *failOnFlagValue) Type() string { return "string" }

var failOnFlag = failOnFlagValue{level: projectfile.FailOnError}

// readOpts builds the shared ReadOptions from the persistent include-resolution
// flags (offline, fail-on). Every command that resolves a projectfile uses this
// so the two flags stay in sync across call sites.
func readOpts() projectfile.ReadOptions {
	return projectfile.ReadOptions{Offline: offlineFlag, FailOn: failOnFlag.level}
}

func init() {
	registerHelpPalette()
	orderHelp(rootCmd)
	rootCmd.PersistentFlags().BoolVarP(&quietFlag, "quiet", "q", false,
		"mute info; warnings and errors still print")
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false,
		"show each step; also PF_CLI_VERBOSE=1")
	rootCmd.PersistentFlags().BoolVar(&ignoreUserConfigFlag, "ignore-user-config", false,
		"skip $XDG_CONFIG_HOME/projectfile/cli.* loading")
	rootCmd.PersistentFlags().BoolVar(&offlineFlag, "offline", false,
		"refuse network; use cache and embedded data")
	rootCmd.PersistentFlags().BoolVar(&sortedFlag, "sorted", false,
		"write YAML keys in sorted order")
	rootCmd.PersistentFlags().Var(&failOnFlag, "fail-on",
		"abort includes at error|warning")
}

func Execute() {
	rootCmd.Long = rootLong()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		// Usage mistakes (bad flags/args) exit 2, distinct from a runtime
		// failure's exit 1 — the standard CLI convention, so callers can tell
		// "you invoked me wrong" apart from "the operation failed".
		var ue *usageError
		if errors.As(err, &ue) {
			os.Exit(2)
		}
		genlog.FlushDebug()
		os.Exit(1)
	}
}
