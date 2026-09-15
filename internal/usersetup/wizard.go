// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

// Package usersetup drives the `pf-cli setup` interactive wizard that
// reads / writes the per-user pf-cli configuration at
// $XDG_CONFIG_HOME/projectfile/cli.{yaml,yml,toml,json}. The wizard is
// purely UI: every persistence and path concern is delegated to
// internal/userconfig — usersetup never touches disk directly.
package usersetup

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"kiota.ch/projectfile/core/v2/pkg/genlog"
	"kiota.ch/projectfile/core/v2/pkg/projectfile"
	"kiota.ch/projectfile/core/v2/pkg/selector"
	"kiota.ch/projectfile/core/v2/pkg/userconfig"
)

const (
	formatYAML = "yaml"
	formatYML  = "yml"
)

// Options controls one run of Run. Format, when set, bypasses the
// interactive format picker — useful for callers that already know the
// answer (`pf-cli setup --format yaml`).
type Options struct {
	Format string
}

// Run executes the wizard end-to-end: pick format → walk every section →
// write. Returns nil on success, errCancelled-wrapped error on user abort,
// any other error from the I/O layer otherwise.
func Run(opts Options) error {
	cfg := userconfig.Load()
	existingPath, _ := userconfig.ExistingPath()
	if existingPath != "" {
		genlog.Debug("setup: editing existing config", "path", existingPath)
	}

	format, err := resolveFormat(opts.Format, existingPath)
	if err != nil {
		if errors.Is(err, errCancelled) {
			genlog.Warn("setup cancelled, no changes written")
			return nil
		}
		return err
	}
	ext := "." + format

	if err := walkSections(cfg); err != nil {
		if errors.Is(err, errCancelled) {
			genlog.Warn("setup cancelled, no changes written")
			return nil
		}
		return err
	}

	path, err := userconfig.Write(cfg, ext)
	if err != nil {
		return fmt.Errorf("write user config: %w", err)
	}
	genlog.Success(fmt.Sprintf("setup: wrote %s", path))
	if existingPath != "" && existingPath != path {
		// Migration case: user previously had cli.toml and just wrote
		// cli.yaml (or similar). The probe order (yaml > yml > toml > json)
		// means the new file wins, but the old one lingers and will confuse
		// any human reader. Offer to delete — destructive, so it's opt-in
		// per the workspace rule "MUST ask before running destructive
		// commands". Default is "no" — a quick Enter keeps both files.
		return offerMigrationCleanup(existingPath, path)
	}
	return nil
}

// offerMigrationCleanup prompts the user to delete the previous-encoding
// config file after a successful write to a new encoding. Skipping the
// prompt (Enter on default "no") leaves both files on disk; user-config
// load order still picks the new file first, so the leftover is cosmetic
// rather than load-shadowing.
func offerMigrationCleanup(oldPath, newPath string) error {
	genlog.Debug("setup: previous config remains on disk", "old", oldPath, "new", newPath)
	del, err := yesNo(fmt.Sprintf("Delete the previous config at %s?", oldPath), false)
	if err != nil {
		if errors.Is(err, errCancelled) {
			// Cancellation during the cleanup prompt is treated as "leave
			// it alone" — same effect as picking "no". The write already
			// happened; the user just opted out of the optional cleanup.
			genlog.Debug("setup: keeping previous config (cancelled)")
			return nil
		}
		return err
	}
	if !del {
		genlog.Debug("setup: keeping previous config")
		return nil
	}
	if err := os.Remove(oldPath); err != nil {
		// Removal failure is non-fatal — the new config wrote successfully;
		// this is just cleanup. Surface as warn so the user knows to remove
		// it by hand if they care.
		genlog.Warn("setup: could not remove previous config", "path", oldPath, "err", err)
		return nil
	}
	genlog.Success(fmt.Sprintf("setup: removed previous config %s", oldPath))
	return nil
}

// resolveFormat returns the chosen encoding ("yaml" | "toml" | "json").
// Precedence:
//
//  1. Explicit --format flag (preset).
//  2. Auto-detect from an existing cli.<ext> on disk — re-prompting in this
//     case is noise; the user clearly wants to keep their format. They can
//     still pass --format to migrate to a different encoding.
//  3. Interactive picker, YAML-led (workspace preference + spec §4.1).
func resolveFormat(preset, existingPath string) (string, error) {
	if preset != "" {
		switch preset {
		case formatYAML, formatYML, "toml", "json":
			if preset == formatYML {
				return formatYAML, nil
			}
			return preset, nil
		default:
			return "", fmt.Errorf("unsupported --format %q (yaml|toml|json)", preset)
		}
	}
	if existingPath != "" {
		ext := strings.TrimPrefix(filepath.Ext(existingPath), ".")
		if ext == formatYML {
			ext = formatYAML
		}
		genlog.Debug("setup: keeping existing format; pass --format to change", "format", ext)
		return ext, nil
	}
	type item struct{ name, desc string }
	choice, err := selector.Run(selector.Choices[item]{
		Title: "Pick config file format:",
		Items: []item{
			{formatYAML, "YAML (recommended)"},
			{"toml", "TOML"},
			{"json", "JSON"},
		},
		Label:  func(i item) string { return i.name },
		Detail: func(i item) string { return i.desc },
	})
	if err != nil {
		// Translate selector cancellation into the wizard's own sentinel so
		// the top-level Run treats it identically to a mid-wizard escape:
		// log "cancelled" and exit clean rather than emit an error.
		if errors.Is(err, selector.ErrCancelled) {
			return "", errCancelled
		}
		return "", fmt.Errorf("format selection: %w", err)
	}
	return choice.name, nil
}

// walkSections runs every wizard section in display order. Each section
// mutates cfg in place via field setters; cancelling one section aborts
// the whole wizard (errCancelled bubbles up). Sections are independent —
// reordering is cosmetic only.
func walkSections(cfg *userconfig.Config) error {
	steps := []func(*userconfig.Config) error{
		identityStep,
		initDefaultsStep,
		fundingStep,
		securityStep,
		contributingStep,
		conventionsStep,
		scanStep,
		copyrightStep,
	}
	for _, step := range steps {
		if err := step(cfg); err != nil {
			return err
		}
	}
	return nil
}

func identityStep(cfg *userconfig.Config) error {
	id := &cfg.Identity
	// Best-effort prefill from `git config` — gap-fill only so a user who
	// blanked a value last run keeps it blanked. Pre-split git's flat
	// user.name into the family/given pair when the person branch wins;
	// the entity branch reuses the flat string verbatim as `name`.
	gitName := gitUserName()
	if id.Email == "" {
		id.Email = gitUserEmail()
	}
	if id.GPGKey == "" {
		id.GPGKey = gitSigningKey()
	}
	if id.FamilyNames == "" && id.GivenNames == "" && id.Name == "" && gitName != "" {
		fam, giv := projectfile.SplitGitName(gitName)
		id.FamilyNames = fam
		id.GivenNames = giv
		// Stash the flat form on Name too so a user who switches to the
		// entity branch sees it pre-filled.
		id.Name = gitName
	}

	// Ask the discriminator up front; default person. Pre-existing configs
	// keep their IsEntity choice and override the default.
	isEntity, err := personOrEntity(id.IsEntity)
	if err != nil {
		return err
	}
	id.IsEntity = isEntity

	if isEntity {
		// Entity branch — clear person-only fields so a person→entity flip
		// does not leak Affiliation / FamilyNames / GivenNames into a record
		// that, per spec §5.5, must not carry them.
		id.FamilyNames = ""
		id.GivenNames = ""
		id.Affiliation = ""
		return runSection("Identity (organization) — auto-attached to an organizations[] entry with matching email", []field{
			{
				Label: "name", Description: "organization name (e.g. Acme Foundation)",
				Current: id.Name, Setter: func(v string) { id.Name = v },
			},
			{
				Label: "email", Description: "contact email; the merge key for ORCID auto-attach",
				Current: id.Email, Setter: func(v string) { id.Email = v },
			},
			{
				Label: "orcid", Description: "ORCID iD (permitted on entities)",
				Current: id.Orcid, Setter: func(v string) { id.Orcid = v },
			},
			{
				Label: "url", Description: "organization homepage",
				Current: id.URL, Setter: func(v string) { id.URL = v },
			},
			{
				Label: "gpg-key", Description: "GPG fingerprint for SECURITY.md (prefilled from `git config user.signingKey`)",
				Current: id.GPGKey, Setter: func(v string) { id.GPGKey = v },
			},
		})
	}

	// Person branch — clear the entity-only Name slot so a flip back to
	// person leaves no stale label. The split-name fields are the
	// canonical form per spec §5.5.5.
	id.Name = ""
	return runSection("Identity (person) — auto-attached to a people[] person whose email matches", []field{
		{
			Label: "family-names", Description: "family name(s) / surname(s) — required for persons",
			Current: id.FamilyNames, Setter: func(v string) { id.FamilyNames = v },
		},
		{
			Label: "given-names", Description: "given name(s) / first name(s)",
			Current: id.GivenNames, Setter: func(v string) { id.GivenNames = v },
		},
		{
			Label: "email", Description: "your contact email; the merge key for ORCID auto-attach",
			Current: id.Email, Setter: func(v string) { id.Email = v },
		},
		{
			Label: "orcid", Description: "ORCID iD (e.g. 0000-0001-2345-6789)",
			Current: id.Orcid, Setter: func(v string) { id.Orcid = v },
		},
		{
			Label: "url", Description: "personal homepage",
			Current: id.URL, Setter: func(v string) { id.URL = v },
		},
		{
			Label: "affiliation", Description: "current institution or employer",
			Current: id.Affiliation, Setter: func(v string) { id.Affiliation = v },
		},
		{
			Label: "gpg-key", Description: "GPG fingerprint for SECURITY.md (prefilled from `git config user.signingKey`)",
			Current: id.GPGKey, Setter: func(v string) { id.GPGKey = v },
		},
	})
}

// personOrEntity asks the discriminator. Wrapping selector.Run rather than
// reusing yesNo keeps the labels meaningful — "person" / "organization"
// reads less ambiguously than "yes" / "no" on a question of identity kind.
func personOrEntity(currentIsEntity bool) (bool, error) {
	type opt struct {
		label, desc string
		value       bool
	}
	items := []opt{
		{"person", "natural person — split into family / given names", false},
		{"organization", "company, team, project — single name field", true},
	}
	if currentIsEntity {
		items = []opt{items[1], items[0]}
	}
	choice, err := selector.Run(selector.Choices[opt]{
		Title:  "Are you setting up your identity as a person or an organization?",
		Items:  items,
		Label:  func(o opt) string { return o.label },
		Detail: func(o opt) string { return o.desc },
	})
	if err != nil {
		if errors.Is(err, selector.ErrCancelled) {
			return false, errCancelled
		}
		return false, fmt.Errorf("identity kind prompt: %w", err)
	}
	return choice.value, nil
}

func initDefaultsStep(cfg *userconfig.Config) error {
	in := &cfg.Init
	// Text fields run through the standard section form. no_scan is a real
	// boolean — pulled out into its own yes/no selector below so the user
	// gets a visible toggle rather than typing "yes"/"no" into a text box.
	if err := runSection("Init defaults — pre-fill answers for `pf-cli init`", []field{
		{
			Label: "format", Description: "default projectfile format (yaml|toml|json)",
			Current: in.Format, Setter: func(v string) { in.Format = v },
		},
		{
			Label: "namespace-default", Description: "reverse-DNS namespace pre-fill (e.g. org.example)",
			Current: in.NamespaceDefault, Setter: func(v string) { in.NamespaceDefault = v },
		},
		{
			Label: "license-default", Description: "default SPDX expression (e.g. MIT)",
			Current: in.LicenseDefault, Setter: func(v string) { in.LicenseDefault = v },
		},
	}); err != nil {
		return err
	}
	noScan, err := yesNo("Init defaults — skip git/stack scanners by default?", in.NoScan)
	if err != nil {
		return err
	}
	in.NoScan = noScan
	return nil
}

func fundingStep(cfg *userconfig.Config) error {
	f := &cfg.Funding
	// Two slots cover the 95% case: GitHub Sponsors (most-used named provider)
	// and `custom` (escape hatch — any URL). The remaining 11 FUNDING.yml-
	// known providers are still honoured at load time; users who want them
	// edit cli.yaml directly. The footer line keeps that discoverable
	// without forcing every user through 13 prompts on first run.
	return runSection(
		"Funding — personal funding URLs (FUNDING.yml fallback)", []field{
			{
				Label: "github", Description: "GitHub Sponsors usernames (comma-separated)",
				Current: joinCSV(f.GitHub), Setter: func(v string) { f.GitHub = splitCSV(v) },
			},
			{
				Label: "custom", Description: "arbitrary URLs (comma-separated) — Patreon, Ko-fi, etc.",
				Current: joinCSV(f.Custom), Setter: func(v string) { f.Custom = splitCSV(v) },
			},
		},
		"+ 11 named providers (patreon, ko-fi, liberapay, open-collective, polar, buy-me-a-coffee, thanks-dev, tidelift, community-bridge, lfx-crowdfunding, issuehunt) — edit cli.yaml directly",
	)
}

func securityStep(cfg *userconfig.Config) error {
	s := &cfg.Security
	return runSection("Security — SECURITY.md defaults", []field{
		{
			Label: "report-url", Description: "vulnerability-report URL (hackerone, etc.)",
			Current: s.ReportURL, Setter: func(v string) { s.ReportURL = v },
		},
		{
			Label: "disclosure-window", Description: "e.g. 90 days",
			Current: s.DisclosureWindow, Setter: func(v string) { s.DisclosureWindow = v },
		},
		{
			Label: "bug-bounty-url", Description: "program landing page",
			Current: s.BugBountyURL, Setter: func(v string) { s.BugBountyURL = v },
		},
	})
}

func contributingStep(cfg *userconfig.Config) error {
	c := &cfg.Contributing
	return runSection("Contributing — CONTRIBUTING.md defaults", []field{
		{
			Label: "cla-url", Description: "Contributor License Agreement URL",
			Current: c.CLAURL, Setter: func(v string) { c.CLAURL = v },
		},
		{
			Label: "chat-url", Description: "community chat (Discord, Matrix, …)",
			Current: c.ChatURL, Setter: func(v string) { c.ChatURL = v },
		},
	})
}

func conventionsStep(cfg *userconfig.Config) error {
	c := &cfg.Conventions
	return runSection("Conventions — cross-cutting workflow defaults", []field{
		{
			Label: "commit-style", Description: "commit message style (conventional | gitmoji | free)",
			Current: c.CommitStyle, Setter: func(v string) { c.CommitStyle = v },
		},
		{
			Label: "workflow", Description: "branching workflow (github-flow | git-flow | gitlab-flow | trunk-based | centralized)",
			Current: c.Workflow, Setter: func(v string) { c.Workflow = v },
		},
		{
			Label: "style-guide-url", Description: "code style guide URL",
			Current: c.StyleGuideURL, Setter: func(v string) { c.StyleGuideURL = v },
		},
	})
}

func scanStep(cfg *userconfig.Config) error {
	s := &cfg.Scan
	return runSection("Scan — private hosts omitted from any scanner output", []field{
		{
			Label: "private-hosts", Description: "hostnames to redact (comma-separated, e.g. src.example.com)",
			Current: joinCSV(s.PrivateHosts), Setter: func(v string) { s.PrivateHosts = splitCSV(v) },
		},
	})
}

func copyrightStep(cfg *userconfig.Config) error {
	c := &cfg.Copyright
	return runSection("Copyright — LICENSE year strategy", []field{
		{
			Label: "year-strategy", Description: "current | fixed (empty = current)",
			Current: c.YearStrategy, Setter: func(v string) { c.YearStrategy = v },
		},
	})
}

// splitCSV is the wizard's list-parsing convention: split on commas, trim
// whitespace, drop empties. Returns nil (not []string{}) on empty input so
// the omitempty-tagged field stays absent from the written file.
func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// joinCSV is splitCSV's inverse for display in the prompt's Current value.
func joinCSV(xs []string) string { return strings.Join(xs, ", ") }
