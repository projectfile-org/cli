// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/spf13/cobra"

	"kiota.ch/projectfile/core/v2/pkg/genlog"
	"kiota.ch/projectfile/core/v2/pkg/projectfile"
	"projectfile.org/projectfile/cli/internal/validate"
)

// strictIncludesFlag promotes redundant-include findings from warnings to a
// hard failure (exit 1). Off by default so the check is visible everywhere
// without breaking existing valid documents; a CI gate opts in explicitly.
var strictIncludesFlag bool

var validateCmd = &cobra.Command{
	Use:   "validate [directory]",
	Short: "Validate a projectfile against the v1 JSON Schema",
	Long: "Check the document against the embedded v1 schema.\n" +
		"Warns on redundant includes; --strict-includes fails instead.",
	Aliases: []string{"v", "lint"},
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := "."
		if len(args) > 0 {
			dir = args[0]
		}

		raw, path, err := projectfile.ReadRawWithOptions(dir, readOpts())
		if err != nil {
			return err
		}

		// Include-hygiene guardrail: flag any direct include a sibling already
		// provides. It runs on the BASE document (the literal includes list) —
		// the merged `raw` above carries the deduped union and would show
		// nothing. A base-read failure degrades to skipping the check so it can
		// never mask the schema result.
		var redundant []projectfile.RedundantInclude
		if base, berr := projectfile.ReadRawBaseFromPath(path); berr == nil {
			redundant = projectfile.RedundantIncludes(base, filepath.Dir(path), path, readOpts())
		} else {
			genlog.Warn("include check skipped: cannot read base document", "path", path, "error", berr)
		}
		for _, r := range redundant {
			genlog.Warn("redundant include", "include", r.Ref, "already-provided-by", r.Via, "kind", r.Reason)
		}

		schemaErr := validate.Validate(raw)
		strictFail := strictIncludesFlag && len(redundant) > 0

		if schemaErr == nil && !strictFail {
			msg := fmt.Sprintf("%s: valid", path)
			if len(redundant) > 0 {
				msg = fmt.Sprintf("%s: valid (%d redundant include(s) — see warnings)", path, len(redundant))
			}
			genlog.Plain(msg)
			return nil
		}

		// Schema violations print as a flat list of "location: reason" lines so
		// the output is grep-friendly and stable across schema-library upgrades
		// — the nested ValidationError tree is an internal detail.
		if schemaErr != nil {
			ve, ok := validate.AsValidationError(schemaErr)
			if !ok {
				return fmt.Errorf("validate %s: %w", path, schemaErr)
			}
			lines := flattenViolations(ve)
			out := cmd.ErrOrStderr()
			fmt.Fprintf(out, "%s: invalid (%d violation", path, len(lines))
			if len(lines) != 1 {
				fmt.Fprint(out, "s")
			}
			fmt.Fprintln(out, ")")
			for _, line := range lines {
				fmt.Fprintln(out, "  "+line)
			}
		}

		switch {
		case schemaErr != nil && strictFail:
			return fmt.Errorf("schema validation failed; %d redundant include(s)", len(redundant))
		case schemaErr != nil:
			return fmt.Errorf("schema validation failed")
		default:
			return fmt.Errorf("%d redundant include(s) — remove them or drop --strict-includes", len(redundant))
		}
	},
}

// flattenViolations walks the ValidationError tree depth-first and returns
// one entry per leaf cause. For leaves, node.Error() already renders as
// "at '<loc>': <localized message>" via the library's printer — using it
// keeps message formatting consistent with every other tool built on this
// validator.
func flattenViolations(ve *jsonschema.ValidationError) []string {
	var out []string
	var walk func(*jsonschema.ValidationError)
	walk = func(node *jsonschema.ValidationError) {
		if len(node.Causes) == 0 {
			out = append(out, node.Error())
			return
		}
		for _, c := range node.Causes {
			walk(c)
		}
	}
	walk(ve)
	return out
}

func init() {
	validateCmd.Flags().BoolVar(&strictIncludesFlag, "strict-includes", false,
		"fail on redundant includes (default: warn)")
	rootCmd.AddCommand(validateCmd)
}
