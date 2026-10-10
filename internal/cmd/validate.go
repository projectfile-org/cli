// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

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
	Long: "Check that the projectfile is well-formed.\n" +
		"Warns about repeated includes.",
	Example: "  pf-cli validate\n" +
		"  pf-cli validate /tmp/demo\n" +
		"  pf-cli validate --strict-includes",
	Aliases: []string{"lint"},
	Args:    usageArgs(cobra.MaximumNArgs(1)),
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
			if r.Reason == "duplicate" {
				genlog.Warn("include listed twice", "include", r.Ref)
				continue
			}
			genlog.Warn("redundant include", "include", r.Ref, "already-provided-by", r.Via, "kind", r.Reason)
		}

		schemaErr := validate.Validate(raw)
		strictFail := strictIncludesFlag && len(redundant) > 0

		// Violations accumulate so checks the schema cannot express contribute
		// their own line: the TOML discriminator (spec §3.4) is keyed on the
		// file extension because anyOf cannot tell encodings apart.
		violations := make([]string, 0, 4)
		if line, bad := tomlDiscriminator(path, raw); bad {
			violations = append(violations, line)
		}

		// Schema violations print as a flat list of "location: reason" lines so
		// the output is grep-friendly and stable across schema-library upgrades
		// — the nested ValidationError tree is an internal detail. The
		// origin-cardinality oneOf over the whole list is replaced by a message
		// naming the entries, since the schema never says which they are.
		if schemaErr != nil {
			ve, ok := validate.AsValidationError(schemaErr)
			if !ok {
				return fmt.Errorf("validate %s: %w", path, schemaErr)
			}
			named, bad := repositoriesOriginViolation(raw)
			for _, line := range flattenViolations(ve) {
				if bad && isRepositoriesViolation(line) {
					continue
				}
				violations = append(violations, line)
			}
			if bad {
				violations = append(violations, named)
			}
		}

		if len(violations) == 0 && !strictFail {
			msg := fmt.Sprintf("%s: valid", path)
			if len(redundant) > 0 {
				msg = fmt.Sprintf("%s: valid (%d redundant include(s) — see warnings)", path, len(redundant))
			}
			genlog.Plain(msg)
			return nil
		}

		if len(violations) > 0 {
			out := cmd.ErrOrStderr()
			fmt.Fprintf(out, "%s: invalid (%d violation", path, len(violations))
			if len(violations) != 1 {
				fmt.Fprint(out, "s")
			}
			fmt.Fprintln(out, ")")
			for _, line := range violations {
				fmt.Fprintln(out, "  "+line)
			}
		}

		switch {
		case schemaErr != nil && strictFail:
			return fmt.Errorf("schema validation failed; %d redundant include(s)", len(redundant))
		case schemaErr != nil:
			return fmt.Errorf("schema validation failed")
		case len(violations) > 0:
			return fmt.Errorf("TOML documents require spec_version = %q", specVersion)
		default:
			return fmt.Errorf("%d redundant include(s) — remove them or drop --strict-includes", len(redundant))
		}
	},
}

// specVersion is the value the TOML discriminator must carry (spec §3.4).
const specVersion = "1"

// tomlDiscriminator enforces the encoding-specific discriminator the schema's
// anyOf cannot check: a .toml document MUST carry spec_version = "1" even when
// it also has "$schema" (spec §3.4/§4.1). Keyed on the file extension.
func tomlDiscriminator(path string, raw map[string]any) (string, bool) {
	if !strings.EqualFold(filepath.Ext(path), ".toml") {
		return "", false
	}
	if v, _ := raw["spec_version"].(string); v == specVersion {
		return "", false
	}
	return "TOML carries its discriminator in spec_version — add spec_version = \"1\" at the top level", true
}

// repositoriesPrefixes match every leaf location the origin-cardinality oneOf
// reports under, from the list itself down to a single entry's role.
var repositoriesPrefixes = []string{"at '/repositories'", "at '/repositories/"}

func isRepositoriesViolation(line string) bool {
	for _, p := range repositoriesPrefixes {
		if strings.HasPrefix(line, p) {
			return true
		}
	}
	return false
}

// repositoriesOriginViolation rewrites the schema's opaque oneOf failure over
// the whole list (spec §4.3a) into a message naming the entries: with more than
// one repository, exactly one MUST carry role: origin — a sole entry is
// implicitly origin and unconstrained. It only reports when the rule is
// actually broken, so genuine per-entry failures keep their own lines.
func repositoriesOriginViolation(raw map[string]any) (string, bool) {
	list, ok := raw["repositories"].([]any)
	if !ok || len(list) < 2 {
		return "", false
	}
	origins, urls := 0, make([]string, 0, len(list))
	for _, item := range list {
		entry, ok := item.(map[string]any)
		if !ok {
			return "", false
		}
		if role, _ := entry["role"].(string); role == "origin" {
			origins++
		}
		if url, _ := entry["url"].(string); url != "" {
			urls = append(urls, url)
		}
	}
	if origins == 1 {
		return "", false
	}
	return fmt.Sprintf("repositories: %d entries, %d marked role: origin, exactly one required. URLs: %s",
		len(list), origins, strings.Join(urls, " ")), true
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
		"fail on repeated includes (default: warn)")
	rootCmd.AddCommand(validateCmd)
}
