// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"kiota.ch/projectfile/core/v2/pkg/genlog"
	"kiota.ch/projectfile/core/v2/pkg/projectfile"
	"kiota.ch/projectfile/core/v2/pkg/userconfig"
)

var (
	initFormat         string
	initNonInteractive bool
	initNamespace      string
	initName           string
	initLicense        string
	initNoScan         bool
)

// execBin replaces the current process with another binary. A variable so
// tests can stub the exec and assert the delegation argv instead.
var execBin = syscall.Exec

var initCmd = &cobra.Command{
	Use:     "init [directory]",
	Aliases: []string{"scaffold"},
	Short:   "Scaffold a new projectfile document",
	Long: "Create a new projectfile.yaml (or .toml/.json) in [directory].\n" +
		"With pf-bridge installed this delegates to pf-bridge-init: full\n" +
		"detection from sources (package.json, CITATION.cff, …) plus scanner\n" +
		"gap-fill (git history, stack detection). Without it, a basic local\n" +
		"scaffold prompts for namespace, name and title only — no scanners.",
	Example: "  pf-cli init\n" +
		"  pf-cli init /tmp/demo --namespace org.example --name demo --non-interactive",
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := "."
		if len(args) > 0 {
			dir = args[0]
		}
		if err := delegateInit(); err != nil {
			if !errors.Is(err, errNoBridge) {
				return err
			}
			return basicInit(cmd.OutOrStdout(), os.Stdin, dir, basicOptions{
				Format:         initFormat,
				NonInteractive: initNonInteractive,
				Namespace:      initNamespace,
				Name:           initName,
				License:        initLicense,
			})
		}
		return nil
	},
}

// errNoBridge reports that pf-bridge-init is not installed, so the caller
// falls back to the basic local scaffold.
var errNoBridge = errors.New("pf-bridge-init not found on PATH")

// delegateInit hands the run to pf-bridge-init when it is installed.
func delegateInit() error {
	path, err := exec.LookPath("pf-bridge-init")
	if err != nil {
		return errNoBridge
	}
	argv := append([]string{"pf-bridge-init"}, initTailArgs()...)
	if err := execBin(path, argv, os.Environ()); err != nil {
		return fmt.Errorf("exec pf-bridge-init: %w", err)
	}
	return nil
}

// initTailArgs forwards everything the user typed after `init` (or its
// `scaffold` alias) to pf-bridge-init verbatim, so flag sets can never drift
// apart: new bridge flags keep working without a cli change.
func initTailArgs() []string {
	for i, a := range os.Args {
		if (a == "init" || a == "scaffold") && i+1 < len(os.Args) {
			return os.Args[i+1:]
		}
	}
	return nil
}

func init() {
	initCmd.Flags().StringVarP(&initFormat, "format", "f", "",
		"output format: yaml, toml, json (default: prompt)")
	initCmd.Flags().BoolVar(&initNonInteractive, "non-interactive", false,
		"fail if required fields are missing instead of prompting")
	initCmd.Flags().StringVar(&initNamespace, "namespace", "",
		"identity.namespace (reverse-DNS, e.g. org.example)")
	initCmd.Flags().StringVar(&initName, "name", "",
		"identity.name (project slug)")
	initCmd.Flags().StringVar(&initLicense, "license", "",
		"license SPDX expression (e.g. MIT)")
	initCmd.Flags().BoolVar(&initNoScan, "no-scan", false,
		"skip init-time scanners (accepted for pf-bridge-init parity; the basic scaffold never scans)")
	rootCmd.AddCommand(initCmd)
}

// basicOptions carries the init flag values into the fallback scaffold.
type basicOptions struct {
	Format         string
	NonInteractive bool
	Namespace      string
	Name           string
	License        string
}

// basicInit scaffolds a minimal document when pf-bridge is absent: three
// prompts (namespace, name, title), license and format from flags or user
// config, no source detection and no scanners.
func basicInit(out io.Writer, in io.Reader, dir string, opts basicOptions) error {
	if _, _, err := projectfile.ReadRawWithOptions(dir, readOpts()); err == nil {
		return fmt.Errorf("projectfile already exists in %s", dir)
	}
	ucfg := userconfig.Load()
	rd := bufio.NewReader(in)
	fmt.Fprintln(out, "Basic scaffold (pf-bridge not installed — no source or scanner detection).")
	namespace := opts.Namespace
	if namespace == "" {
		namespace = ucfg.Init.NamespaceDefault
	}
	if !opts.NonInteractive {
		namespace = promptLine(out, rd, "identity.namespace (reverse-DNS owner, e.g. org.example)", namespace)
	}
	if namespace == "" {
		return fmt.Errorf("missing required field: identity.namespace (use --namespace or run interactively)")
	}
	name := opts.Name
	dirBase := ""
	if abs, absErr := filepath.Abs(dir); absErr == nil {
		dirBase = filepath.Base(abs)
	}
	if name == "" {
		name = dirBase
	}
	if !opts.NonInteractive {
		name = promptLine(out, rd, "identity.name (lowercase slug, e.g. my-tool)", name)
	}
	if name == "" {
		return fmt.Errorf("missing required field: identity.name (use --name or run interactively)")
	}
	var title string
	if !opts.NonInteractive {
		title = promptLine(out, rd, "identity.title (human-readable name, optional)", "")
	}
	license := opts.License
	if license == "" {
		license = ucfg.Init.LicenseDefault
	}
	format := opts.Format
	if format == "" {
		format = ucfg.Init.Format
	}
	if format == "" {
		if opts.NonInteractive {
			format = fmtYAML
		} else {
			format = promptLine(out, rd, "format (yaml, toml, json)", fmtYAML)
		}
	}
	switch format {
	case fmtYAML, "yml", "toml", "json":
	default:
		return fmt.Errorf("unsupported format %q (use yaml, toml, or json)", format)
	}
	doc := &projectfile.Document{
		Schema: "https://projectfile.org/schema/v1.json",
		Kind:   "SoftwareSourceCode",
	}
	doc.Identity.Namespace = namespace
	doc.Identity.Name = name
	if title != "" {
		doc.Identity.Title = &projectfile.LocalizedString{Bare: title}
	}
	if license != "" {
		doc.License = &projectfile.License{Spdx: license}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil { // #nosec G301 -- project scaffold dir, world-readable by intent
		return fmt.Errorf("create directory %s: %w", dir, err)
	}
	path := filepath.Join(dir, "projectfile."+format)
	if err := projectfile.Write(doc, path); err != nil {
		return fmt.Errorf("write projectfile: %w", err)
	}
	genlog.Success(fmt.Sprintf("created %s (basic scaffold — install pf-bridge for full detection)", filepath.Base(path)))
	return nil
}

// promptLine prints one "label [default]:" prompt and returns the trimmed
// answer, or the default when the user just hits enter. A closed stdin reads
// as empty so a non-terminal run without flags fails on the required check
// instead of blocking forever.
func promptLine(out io.Writer, rd *bufio.Reader, label, def string) string {
	if def != "" {
		fmt.Fprintf(out, "%s [%s]: ", label, def)
	} else {
		fmt.Fprintf(out, "%s: ", label)
	}
	line, err := rd.ReadString('\n')
	if err != nil {
		return def
	}
	if v := strings.TrimSpace(line); v != "" {
		return v
	}
	return def
}
