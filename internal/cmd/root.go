// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"

	"kiota.ch/projectfile/core/v2/pkg/genlog"
	"kiota.ch/projectfile/core/v2/pkg/netfetch"
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

// helpGroups is the order the root help lists commands in: read, write, validate, maintain; a command outside every group lands under "Other:".
var helpGroups = []*cobra.Group{
	{ID: "read", Title: "Read:"},
	{ID: "write", Title: "Write:"},
	{ID: "validate", Title: "Validate:"},
	{ID: "maintain", Title: "Maintain:"},
}

// helpTemplate orders every help screen as Description, Examples, Commands (grouped), the common flags, then the full flag lists and the doc links.
const helpTemplate = `{{hdr "Usage:"}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{else}}
  {{.UseLine}}{{end}}{{if gt (len .Aliases) 0}}

{{hdr "Aliases:"}}
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

{{hdr "Examples:"}}
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{range $group := .Groups}}

{{hdr $group.Title}}{{range $cmds}}{{if and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help"))}}
  {{cmd (rpad .Name .NamePadding)}} {{.Short}}{{end}}{{end}}{{end}}

{{hdr "Other:"}}{{range $cmds}}{{if and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help"))}}
  {{cmd (rpad .Name .NamePadding)}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

{{hdr "Common flags:"}}
  {{commonFlags .}}{{end}}{{if .HasAvailableLocalFlags}}

{{hdr "Flags:"}}
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

{{hdr "Global Flags:"}}
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if hasDocLinks .}}

{{hdr "Documentation:"}}
  {{docDocs .}}

{{hdr "Report a bug:"}}
  {{docIssues .}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`

// orderHelp pins the Description, Commands, Flags, Examples order onto one command.
func orderHelp(c *cobra.Command) {
	c.SetUsageTemplate(helpTemplate)
}

// usageArgs wraps a cobra arity validator so a rejection reads in the tool's
// own voice: the command, the shape it expects (its own Use line) and the fix,
// routed through errUsage so it exits 2 like every other usage mistake. The
// optional detail names values the validator cannot know (convert's format list).
func usageArgs(fn cobra.PositionalArgs, detail ...string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if err := fn(cmd, args); err != nil {
			shape := strings.TrimSpace(strings.TrimPrefix(cmd.Use, cmd.Name()))
			parts := []string{}
			if shape == "" {
				parts = append(parts, fmt.Sprintf("%s takes no arguments (got %d)", cmd.CommandPath(), len(args)))
			} else {
				parts = append(parts, fmt.Sprintf("%s needs %s (got %d)", cmd.CommandPath(), shape, len(args)))
			}
			if len(detail) > 0 && detail[0] != "" {
				parts = append(parts, detail[0])
			}
			return errUsage(strings.Join(parts, ". ") + fmt.Sprintf(". Run %s --help.", cmd.CommandPath()))
		}
		return nil
	}
}

// flagError rewrites pflag's parse failures in the tool's own voice — the
// command, the library's reason, the fix — and routes them through errUsage so
// they join the usage exit class.
func flagError(cmd *cobra.Command, err error) error {
	return errUsage(fmt.Sprintf("%s: %s. Run %s --help.", cmd.CommandPath(), err, cmd.CommandPath()))
}

// commandError rewrites cobra's own dispatch failure in the tool's own voice, so an unknown command names itself and points at the command list.
func commandError(_ *cobra.Command, err error) error {
	msg := err.Error()
	if strings.HasPrefix(msg, "unknown command") {
		msg = fmt.Sprintf("%s. Run pf-cli --help to see every command.", msg)
		return errUsage(msg)
	}
	return errUsage(fmt.Sprintf("pf-cli: %s. Run pf-cli --help.", msg))
}

// registerHelpPalette exposes the lipgloss styles to the help template.
func registerHelpPalette() {
	cobra.AddTemplateFunc("hdr", helpHeading.Render)
	cobra.AddTemplateFunc("cmd", helpCommand.Render)
	// commonFlags renders the short "flags worth knowing" block the checklist requires ahead of the full list; the exhaustive set follows.
	cobra.AddTemplateFunc("commonFlags", func(c *cobra.Command) string {
		names := []string{"quiet", "verbose", "offline", "force", "dry-run", "path-file", "format"}
		var b strings.Builder
		for _, n := range names {
			f := c.Flags().Lookup(n)
			if f == nil {
				continue
			}
			label := "--" + f.Name
			if f.Shorthand != "" {
				label = "-" + f.Shorthand + ", --" + f.Name
			}
			fmt.Fprintf(&b, "  %-20s %s\n", label, f.Usage)
		}
		return strings.TrimRight(b.String(), "\n")
	})
	// The three doc-link funcs resolve at RENDER time, because SetProjectfileYAML runs after package init and an eagerly stamped annotation reads empty.
	cobra.AddTemplateFunc("hasDocLinks", func(_ *cobra.Command) bool {
		docs, _ := docLinks()
		return docs != ""
	})
	cobra.AddTemplateFunc("docDocs", func(c *cobra.Command) string {
		// Cobra's own commands are added lazily and never pass through groupCmd, so the name is the fallback page; the root keeps the index and `help` has no page.
		page := c.Annotations["doc-page"]
		if page == "" && c.Parent() != nil && c.Name() != "help" {
			page = c.Name()
		}
		return docsForPage(page)
	})
	cobra.AddTemplateFunc("docIssues", func(_ *cobra.Command) string {
		_, issues := docLinks()
		return issues
	})
}

// linkTypeSourceCode and linkTypeBugs are the projectfile link types the docs renderer reads, named by the projectfile schema.
const (
	linkTypeSourceCode = "source-code"
	linkTypeBugs       = "bugs"
)

// docLinks derives the docs and issue-tracker URLs from the embedded projectfile, so help points at declared addresses rather than a hardcoded host; the PUBLIC source repo is preferred for docs, being the page every reader can reach.
func docLinks() (docs, issues string) {
	var doc struct {
		Links []struct {
			Type      string   `yaml:"type"`
			Tags      []string `yaml:"tags"`
			Preferred bool     `yaml:"preferred"`
			URL       string   `yaml:"url"`
		} `yaml:"links"`
	}
	if err := yaml.Unmarshal(projectfileYAML, &doc); err != nil {
		return "", ""
	}
	for _, l := range doc.Links {
		if l.Type == linkTypeBugs && issues == "" {
			issues = l.URL
		}
		if l.Type != linkTypeSourceCode || docs != "" {
			continue
		}
		for _, t := range l.Tags {
			if t == "public" {
				docs = l.URL
				break
			}
		}
	}
	if docs == "" {
		for _, l := range doc.Links {
			if l.Type == linkTypeSourceCode && l.Preferred {
				docs = l.URL
				break
			}
		}
	}
	if docs == "" {
		for _, l := range doc.Links {
			if l.Type == linkTypeSourceCode {
				docs = l.URL
				break
			}
		}
	}
	if docs != "" {
		docs += "/tree/main/docs"
	}
	return docs, issues
}

// docsForPage turns the docs index into the exact page a command's help links: the root keeps the index, a subcommand gets its USAGE.md heading anchor.
func docsForPage(page string) string {
	docs, _ := docLinks()
	if page == "" {
		return docs
	}
	base := strings.TrimSuffix(docs, "/tree/main/docs")
	return base + "/blob/main/docs/USAGE.md#pf-cli-" + strings.ReplaceAll(page, " ", "-")
}

// groupCmd registers one subcommand under its help group and stamps its docs page, so the root lists it in a named group and its own --help carries the page link.
func groupCmd(c *cobra.Command, groupID, page string) {
	c.GroupID = groupID
	stampDocLinks(c, page)
	rootCmd.AddCommand(c)
}

// stampDocLinks records which docs page a command's help links; the URLs resolve lazily at render time, since SetProjectfileYAML lands after package init.
func stampDocLinks(c *cobra.Command, page string) {
	if c.Annotations == nil {
		c.Annotations = map[string]string{}
	}
	c.Annotations["doc-page"] = page
}

// conciseHelp is the bare-invocation screen: one line on what the tool does, worked examples, the flags worth knowing, and the pointer to --help.
func conciseHelp(out io.Writer, docs, issues string) {
	fmt.Fprintln(out, rootShort+" — read, write and validate projectfile documents.")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Try one:")
	fmt.Fprintln(out, "  pf-cli get identity.name      read a field")
	fmt.Fprintln(out, "  pf-cli set license.spdx MIT   write a field")
	fmt.Fprintln(out, "  pf-cli validate               check the document against the v1 schema")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Common flags:")
	fmt.Fprintln(out, "  -q, --quiet       mute info lines")
	fmt.Fprintln(out, "  -v, --verbose     show each step")
	fmt.Fprintln(out, "      --offline     refuse network; use cache and embedded data")
	fmt.Fprintln(out, "  -f, --force       overwrite the output (convert, cache warm)")
	if docs != "" {
		fmt.Fprintln(out)
		fmt.Fprintf(out, "Docs:   %s\n", docs)
	}
	if issues != "" {
		fmt.Fprintf(out, "Issues: %s\n", issues)
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Run pf-cli --help for every command and flag.")
}

// rootLong builds the Description from the projectfile identity text baked in at build time.
func rootLong() string {
	desc := projectfileDescription()
	if desc == "" {
		desc = rootShort
	}
	return wrap80(desc) + "\n\nFile and forge sync (bridge, forge, scan): use pf-bridge."
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

// Exit codes, in one table so the mapping cannot drift between the call sites and the docs that render it; a caller branches on the CLASS of failure, and every nonzero that is not 2 is "the tool ran and could not finish".
const (
	exitOK          = 0 // success
	exitFailure     = 1 // runtime/IO failure: unreadable file, bad include, refused write
	exitUsage       = 2 // the invocation itself: unknown flag/command, wrong arity, bad value
	exitAbsent      = 3 // the requested field does not exist
	exitInvalidated = 4 // the document was read and fails the v1 schema
)

// exitCodeFor maps a returned error onto its exit class; Execute applies it.
func exitCodeFor(err error) int {
	var ve *validationError
	var ue *usageError
	var ae *absentError
	switch {
	case errors.As(err, &ve):
		return exitInvalidated
	case errors.As(err, &ue):
		return exitUsage
	case errors.As(err, &ae):
		return exitAbsent
	}
	return exitFailure
}

// usageError marks an invocation mistake — a bad flag, wrong arity, a value
// the command cannot accept.
type usageError struct{ msg string }

func (e *usageError) Error() string { return e.msg }

// absentError marks a requested field that does not exist, so a script tells
// a mistyped address apart from a broken document.
type absentError struct{ msg string }

func (e *absentError) Error() string { return e.msg }

// validationError marks a document that violates the v1 schema.
type validationError struct{ msg string }

func (e *validationError) Error() string { return e.msg }

func errUsage(msg string) error      { return &usageError{msg: msg} }
func errAbsent(msg string) error     { return &absentError{msg: msg} }
func errValidation(msg string) error { return &validationError{msg: msg} }

var rootCmd = &cobra.Command{
	Use:   "pf-cli [command]",
	Short: rootShort,
	Long:  rootLong(),
	Example: "  pf-cli init --namespace org.example --name demo\n" +
		"  pf-cli get identity.name\n" +
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
	// A bare invocation prints the concise summary; --help still renders the
	// full screen, and any subcommand dispatches past this RunE untouched.
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return commandError(cmd, fmt.Errorf("unknown command %q for %q", args[0], cmd.CommandPath()))
		}
		docs, issues := docLinks()
		conciseHelp(cmd.OutOrStdout(), docs, issues)
		return nil
	},
	PersistentPreRunE: func(c *cobra.Command, _ []string) error {
		if err := applyColors(c); err != nil {
			return err
		}
		genlog.SetResultOutput(c.OutOrStdout())
		netfetch.SetTimeout(timeoutFlag)
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
		genlog.Debug("output", "colors", colorsFlag, "timeout", netfetch.Timeout())
		return nil
	},
}

// colorsFlag is --colors: auto, always or never; PF_CLI_NO_COLOR=1 means never.
var colorsFlag = genlog.ColorAuto

// applyColors hands --colors, or PF_CLI_NO_COLOR=1 when the flag is unset, to core's one colour decision.
func applyColors(c *cobra.Command) error {
	if v, _ := strconv.ParseBool(os.Getenv("PF_CLI_NO_COLOR")); v && !c.Flags().Changed("colors") {
		colorsFlag = genlog.ColorNever
	}
	return genlog.SetColor(colorsFlag)
}

// timeoutFlag bounds each network attempt; retries and backoff come from core.
var timeoutFlag = netfetch.DefaultTimeout

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
	case "error":
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
	rootCmd.SetFlagErrorFunc(flagError)
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
	rootCmd.PersistentFlags().StringVar(&colorsFlag, "colors", genlog.ColorAuto,
		"colour output: auto|always|never; also PF_CLI_NO_COLOR=1")
	rootCmd.PersistentFlags().DurationVar(&timeoutFlag, "timeout", netfetch.DefaultTimeout,
		"per-attempt network timeout")
	rootCmd.SetFlagErrorFunc(flagError)
	rootCmd.SetOut(genlog.Styled(os.Stdout))
	rootCmd.SetErr(genlog.Styled(os.Stderr))
	defaultHelp := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(c *cobra.Command, args []string) {
		if err := applyColors(c); err != nil {
			genlog.Warn("colors", "err", err.Error())
		}
		defaultHelp(c, args)
	})
	rootCmd.Flags().BoolP("version", "V", false, "print the version")
	rootCmd.AddGroup(helpGroups...)
	stampDocLinks(rootCmd, "")
}

// classifyDispatchError re-classes cobra's own dispatch failures (unknown command, unknown shorthand) into the usage class, so `pf-cli bogus` exits 2 like every other bad invocation; a subcommand's error never reaches here.
func classifyDispatchError(cmd *cobra.Command, err error) error {
	msg := err.Error()
	isDispatch := strings.HasPrefix(msg, "unknown command") ||
		strings.HasPrefix(msg, "unknown shorthand flag") ||
		strings.HasPrefix(msg, "unknown flag")
	if !isDispatch {
		return err
	}
	return commandError(cmd, err)
}

// resolveVersion falls back to the Go build info when no release stamp was linked in.
func resolveVersion() string {
	info, ok := debug.ReadBuildInfo()
	if version != "unknown" || !ok {
		return version
	}
	if v := info.Main.Version; v != "" && v != "(devel)" {
		return v
	}
	rev, dirty := "", ""
	for _, s := range info.Settings {
		switch {
		case s.Key == "vcs.revision" && len(s.Value) >= 12:
			rev = s.Value[:12]
		case s.Key == "vcs.modified" && s.Value == "true":
			dirty = "-dirty"
		}
	}
	if rev == "" {
		return version + " (built without a release stamp or VCS metadata)"
	}
	return "dev-" + rev + dirty
}

func Execute() {
	rootCmd.Long = rootLong()
	rootCmd.Version = resolveVersion()
	err := rootCmd.Execute()
	if err != nil {
		// Cobra's own dispatch failures (unknown command, unknown shorthand)
		// are usage mistakes, not runtime failures; re-class before printing
		// so they take the exit-2 class.
		err = classifyDispatchError(rootCmd, err)
		code := exitCodeFor(err)
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		// Exit classes let a caller tell "you invoked me wrong" (2), "the
		// field you asked for is absent" (3) and "the document fails the
		// schema" (4) apart from a runtime failure (1). Old-to-new: a
		// failure that used to exit 1 and now exits 3 or 4 is still a
		// nonzero, so every existing caller keeps its branch.
		genlog.FlushDebug()
		os.Exit(code)
	}
}
