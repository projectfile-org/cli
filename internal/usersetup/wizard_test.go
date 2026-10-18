// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package usersetup

import (
	"errors"
	"reflect"
	"testing"

	"kiota.ch/projectfile/core/v2/pkg/userconfig"
)

// TestResolveFormatPrefersTheExplicitFlag pins precedence 1: --format wins over anything on disk, so a
// caller that already knows the encoding is never second-guessed by the picker.
func TestResolveFormatPrefersTheExplicitFlag(t *testing.T) {
	for _, tc := range []struct{ preset, want string }{
		{"yaml", formatYAML},
		{encTOML, encTOML},
		{encJSON, encJSON},
	} {
		t.Run(tc.preset, func(t *testing.T) {
			stubPrompts(t, formatAnswers{format: tc.want})
			got, err := resolveFormat(tc.preset, "/config/projectfile/cli.toml")
			assertNoErr(t, err)
			if got != tc.want {
				t.Errorf("resolveFormat(%q) = %q, want %q", tc.preset, got, tc.want)
			}
		})
	}
}

// TestResolveFormatFoldsYmlOntoYaml pins that the .yml spelling normalises to yaml: the writer derives the
// file extension from the returned name, so passing it through would create a second probe name.
func TestResolveFormatFoldsYmlOntoYaml(t *testing.T) {
	got, err := resolveFormat(formatYML, "")
	assertNoErr(t, err)
	if got != formatYAML {
		t.Errorf("resolveFormat(yml) = %q, want %q", got, formatYAML)
	}
}

// TestResolveFormatRejectsAnUnknownEncoding pins the usage error at its edge: an unsupported spelling must
// name the flag and the accepted set, not silently fall back to the picker.
func TestResolveFormatRejectsAnUnknownEncoding(t *testing.T) {
	for _, preset := range []string{"ini", "YAML", "xml", "yaml "} {
		t.Run(preset, func(t *testing.T) {
			stubPrompts(t, formatAnswers{formatErr: errors.New("picker must not open")})
			_, err := resolveFormat(preset, "")
			if err == nil {
				t.Fatal("an unsupported --format was accepted")
			}
			if want := "--format"; !contains(err.Error(), want) {
				t.Errorf("error %q must name %s", err, want)
			}
			if want := "yaml|toml|json"; !contains(err.Error(), want) {
				t.Errorf("error %q must list the accepted encodings", err)
			}
		})
	}
}

// TestResolveFormatKeepsTheExistingEncoding pins precedence 2: re-prompting a user who already has a config
// is noise, so its extension is carried over — and .yml folds to yaml here too.
func TestResolveFormatKeepsTheExistingEncoding(t *testing.T) {
	for _, tc := range []struct {
		name, existing, want string
	}{
		{"yaml", "/c/projectfile/cli.yaml", formatYAML},
		{"yml folds to yaml", "/c/projectfile/cli.yml", formatYAML},
		{encTOML, "/c/projectfile/cli.toml", encTOML},
		{encJSON, "/c/projectfile/cli.json", encJSON},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stubPrompts(t, formatAnswers{formatErr: errors.New("picker must not open")})
			got, err := resolveFormat("", tc.existing)
			assertNoErr(t, err)
			if got != tc.want {
				t.Errorf("resolveFormat(existing=%q) = %q, want %q", tc.existing, got, tc.want)
			}
		})
	}
}

// TestResolveFormatFallsBackToThePicker pins precedence 3: with no flag and no file the picker decides, and
// its answer is what gets written.
func TestResolveFormatFallsBackToThePicker(t *testing.T) {
	stubPrompts(t, formatAnswers{format: encJSON})
	got, err := resolveFormat("", "")
	assertNoErr(t, err)
	if got != encJSON {
		t.Errorf("resolveFormat = %q, want the picker's json", got)
	}
}

// TestResolveFormatMapsSelectorCancellationToTheWizardSentinel pins that Esc at the picker becomes
// errCancelled, so Run reports a clean abort rather than an error the user cannot act on.
func TestResolveFormatMapsSelectorCancellationToTheWizardSentinel(t *testing.T) {
	stubPrompts(t, formatAnswers{formatErr: errCancelledFromSelector()})
	_, err := resolveFormat("", "")
	if !errors.Is(err, errCancelled) {
		t.Errorf("resolveFormat error = %v, want errCancelled", err)
	}
}

// TestResolveFormatWrapsAPickerFailure pins that a genuine picker failure keeps its cause and gains the
// wizard's context, so the message names which step failed.
func TestResolveFormatWrapsAPickerFailure(t *testing.T) {
	cause := errors.New("no tty")
	stubPrompts(t, formatAnswers{formatErr: cause})
	_, err := resolveFormat("", "")
	if !errors.Is(err, cause) {
		t.Fatalf("resolveFormat error = %v, want it to wrap %v", err, cause)
	}
	if !contains(err.Error(), "format selection") {
		t.Errorf("error %q must name the step", err)
	}
}

// TestIdentityStepPrefillsFromGitConfig pins the gap-fill contract: email and gpg-key are taken from git
// only when the config left them blank, so a value the user deliberately cleared stays cleared.
func TestIdentityStepPrefillsFromGitConfig(t *testing.T) {
	t.Chdir(repoWithConfig(t, "[user]\n\tname = Ada Lovelace\n\temail = ada@example.com\n\tsigningKey = ABCDEF0123456789\n"))
	cfg := &userconfig.Config{}
	stubPrompts(t, formatAnswers{isEntity: false})

	assertNoErr(t, identityStep(cfg))

	if cfg.Identity.Email != "ada@example.com" {
		t.Errorf("Email = %q, want the git address", cfg.Identity.Email)
	}
	if cfg.Identity.GPGKey != "ABCDEF0123456789" {
		t.Errorf("GPGKey = %q, want the git signing key", cfg.Identity.GPGKey)
	}
	if cfg.Identity.FamilyNames != gitFamily || cfg.Identity.GivenNames != "Ada" {
		t.Errorf("git name split into %q/%q, want Lovelace/Ada", cfg.Identity.FamilyNames, cfg.Identity.GivenNames)
	}
	if cfg.Identity.Name != "" {
		t.Errorf("Name = %q, want the person branch to carry no entity label", cfg.Identity.Name)
	}
}

// TestIdentityStepEntityBranchTakesTheFlatGitName pins the entity branch reads the flat git name straight
// through, without the family/given split.
func TestIdentityStepEntityBranchTakesTheFlatGitName(t *testing.T) {
	t.Chdir(repoWithConfig(t, "[user]\n\tname = Ada Lovelace\n\temail = ada@example.com\n"))
	cfg := &userconfig.Config{}
	stubPrompts(t, formatAnswers{isEntity: true, sections: keepPrefilled(cfg)})

	assertNoErr(t, identityStep(cfg))

	if cfg.Identity.Name != gitLovelace {
		t.Errorf("Name = %q, want the flat git form", cfg.Identity.Name)
	}
}

// TestIdentityStepKeepsAValueTheUserCleared pins that the prefill is gap-fill only, not an overwrite: a
// config carrying its own email keeps it even when git names a different one.
func TestIdentityStepKeepsAValueTheUserCleared(t *testing.T) {
	t.Chdir(repoWithConfig(t, "[user]\n\tname = Ada Lovelace\n\temail = ada@example.com\n\tsigningKey = ABCDEF0123456789\n"))
	cfg := &userconfig.Config{}
	cfg.Identity.Email = "mine@example.org"
	stubPrompts(t, formatAnswers{isEntity: false})

	assertNoErr(t, identityStep(cfg))

	if cfg.Identity.Email != "mine@example.org" {
		t.Errorf("Email = %q, want the configured address to survive the git prefill", cfg.Identity.Email)
	}
}

// TestIdentityStepKeepsAnAlreadySplitName pins that a config the user split by hand is not re-split from git
// — that would overwrite a deliberate family/given split with the flat string.
func TestIdentityStepKeepsAnAlreadySplitName(t *testing.T) {
	t.Chdir(repoWithConfig(t, "[user]\n\tname = Grace Hopper\n"))
	cfg := &userconfig.Config{}
	cfg.Identity.FamilyNames = "Hopper"
	cfg.Identity.GivenNames = "Grace"
	stubPrompts(t, formatAnswers{isEntity: false})

	assertNoErr(t, identityStep(cfg))

	if cfg.Identity.FamilyNames != "Hopper" || cfg.Identity.GivenNames != "Grace" {
		t.Errorf("name = %q/%q, want the configured split preserved", cfg.Identity.FamilyNames, cfg.Identity.GivenNames)
	}
}

// TestIdentityStepPersonBranchClearsTheEntityName pins the flip-back contract: the person branch must not
// leave a stale entity label in the Name slot, which would then be written beside the split names.
func TestIdentityStepPersonBranchClearsTheEntityName(t *testing.T) {
	isolateGit(t)
	cfg := &userconfig.Config{}
	cfg.Identity.Name = "Acme Foundation"
	stubPrompts(t, formatAnswers{isEntity: false})

	assertNoErr(t, identityStep(cfg))

	if cfg.Identity.Name != "" {
		t.Errorf("Name = %q, want the entity label cleared on the person branch", cfg.Identity.Name)
	}
}

// TestIdentityStepEntityBranchClearsThePersonFields pins the flip-forward contract: per spec §5.5 an entity
// record must not carry Affiliation / FamilyNames / GivenNames, so a person→entity flip has to drop them.
func TestIdentityStepEntityBranchClearsThePersonFields(t *testing.T) {
	isolateGit(t)
	cfg := &userconfig.Config{}
	cfg.Identity.FamilyNames = gitFamily
	cfg.Identity.GivenNames = "Ada"
	cfg.Identity.Affiliation = "Analytical Engine"
	stubPrompts(t, formatAnswers{isEntity: true})

	assertNoErr(t, identityStep(cfg))

	id := cfg.Identity
	if id.FamilyNames != "" || id.GivenNames != "" || id.Affiliation != "" {
		t.Errorf("entity record kept person fields: family=%q given=%q affiliation=%q", id.FamilyNames, id.GivenNames, id.Affiliation)
	}
	if !id.IsEntity {
		t.Error("IsEntity must record the chosen kind")
	}
}

// TestIdentityStepPersonBranchOffersThePersonFields pins the field set each branch exposes: the person form
// is where the split names live, the entity form is where the single Name does.
func TestIdentityStepPersonBranchOffersThePersonFields(t *testing.T) {
	isolateGit(t)
	for _, tc := range []struct {
		name         string
		isEntity     bool
		want, absent []string
	}{
		{kindPerson, false, []string{familyNames, givenNames, "affiliation"}, []string{fieldName}},
		{"entity", true, []string{fieldName, fieldOrcid}, []string{familyNames, givenNames, "affiliation"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var offered []string
			stubPrompts(t, formatAnswers{
				isEntity: tc.isEntity,
				sections: func(_ string, fields []field) error {
					offered = labelsOf(fields)
					return nil
				},
			})
			assertNoErr(t, identityStep(&userconfig.Config{}))
			assertLabels(t, offered, tc.want, tc.absent)
		})
	}
}

// TestIdentityStepPropagatesCancellation pins that Esc at the discriminator aborts the wizard with the
// sentinel, so Run writes nothing.
func TestIdentityStepPropagatesCancellation(t *testing.T) {
	isolateGit(t)
	stubPrompts(t, formatAnswers{entityErr: errCancelled})
	if err := identityStep(&userconfig.Config{}); !errors.Is(err, errCancelled) {
		t.Errorf("identityStep error = %v, want errCancelled", err)
	}
}

// TestIdentityStepPropagatesAPromptFailure pins that a picker failure is not swallowed — it is the wizard's
// only signal that the terminal went away.
func TestIdentityStepPropagatesAPromptFailure(t *testing.T) {
	isolateGit(t)
	cause := errors.New("no tty")
	stubPrompts(t, formatAnswers{entityErr: cause})
	if err := identityStep(&userconfig.Config{}); !errors.Is(err, cause) {
		t.Errorf("identityStep error = %v, want %v", err, cause)
	}
}

// TestInitDefaultsStepRoundTripsTheToggle pins that no_scan is a visible yes/no toggle carrying the existing
// value as its default, rather than a free-text field the user could not answer consistently.
func TestInitDefaultsStepRoundTripsTheToggle(t *testing.T) {
	for _, current := range []bool{false, true} {
		cfg := &userconfig.Config{}
		cfg.Init.NoScan = current
		var asked string
		stubPrompts(t, formatAnswers{
			noScan: !current,
			yesNo: func(title string, _ bool) (bool, error) {
				asked = title
				return !current, nil
			},
		})

		assertNoErr(t, initDefaultsStep(cfg))

		if cfg.Init.NoScan == current {
			t.Errorf("NoScan stayed %v, want the answer %v", current, !current)
		}
		if !contains(asked, "scanners") {
			t.Errorf("toggle prompt %q must name what it skips", asked)
		}
	}
}

// TestInitDefaultsStepPropagatesCancellationFromTheToggle pins that Esc at the toggle aborts before the
// value is written back, so a half-answered section cannot reach the file.
func TestInitDefaultsStepPropagatesCancellationFromTheToggle(t *testing.T) {
	stubPrompts(t, formatAnswers{yesNo: func(string, bool) (bool, error) { return false, errCancelled }})
	if err := initDefaultsStep(&userconfig.Config{}); !errors.Is(err, errCancelled) {
		t.Errorf("initDefaultsStep error = %v, want errCancelled", err)
	}
}

// TestInitDefaultsStepPropagatesCancellationFromTheForm pins the same for the text form above the toggle.
func TestInitDefaultsStepPropagatesCancellationFromTheForm(t *testing.T) {
	stubPrompts(t, formatAnswers{sections: func(string, []field) error { return errCancelled }})
	if err := initDefaultsStep(&userconfig.Config{}); !errors.Is(err, errCancelled) {
		t.Errorf("initDefaultsStep error = %v, want errCancelled", err)
	}
}

// TestSimpleStepsWriteEveryFieldTheyOffer pins that each flat section actually stores what it asked for; a
// setter wired to the wrong field would leave the answer on screen and out of the file.
func TestSimpleStepsWriteEveryFieldTheyOffer(t *testing.T) {
	for _, tc := range []struct {
		name   string
		step   func(*userconfig.Config) error
		answer map[string]string
		assert func(*testing.T, *userconfig.Config)
	}{
		{
			name: "security", step: securityStep,
			answer: map[string]string{
				"report-url":        "https://hackerone.com/report",
				"disclosure-window": "90 days",
				"bug-bounty-url":    "https://bugbounty.example",
			},
			assert: func(t *testing.T, cfg *userconfig.Config) {
				s := cfg.Security
				if s.ReportURL != "https://hackerone.com/report" || s.DisclosureWindow != "90 days" || s.BugBountyURL != "https://bugbounty.example" {
					t.Errorf("security = %+v, want every field stored", s)
				}
			},
		},
		{
			name: "contributing", step: contributingStep,
			answer: map[string]string{"cla-url": "https://cla.example", "chat-url": "https://chat.example"},
			assert: func(t *testing.T, cfg *userconfig.Config) {
				c := cfg.Contributing
				if c.CLAURL != "https://cla.example" || c.ChatURL != "https://chat.example" {
					t.Errorf("contributing = %+v, want every field stored", c)
				}
			},
		},
		{
			name: "conventions", step: conventionsStep,
			answer: map[string]string{
				"commit-style":    "conventional",
				"workflow":        "trunk-based",
				"style-guide-url": "https://style.example",
			},
			assert: func(t *testing.T, cfg *userconfig.Config) {
				c := cfg.Conventions
				if c.CommitStyle != "conventional" || c.Workflow != "trunk-based" || c.StyleGuideURL != "https://style.example" {
					t.Errorf("conventions = %+v, want every field stored", c)
				}
			},
		},
		{
			name: "copyright", step: copyrightStep,
			answer: map[string]string{"year-strategy": "fixed"},
			assert: func(t *testing.T, cfg *userconfig.Config) {
				if cfg.Copyright.YearStrategy != "fixed" {
					t.Errorf("year strategy = %q, want fixed", cfg.Copyright.YearStrategy)
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &userconfig.Config{}
			stubPrompts(t, formatAnswers{sections: answerSections(tc.answer)})
			assertNoErr(t, tc.step(cfg))
			tc.assert(t, cfg)
		})
	}
}

// TestSimpleStepsPropagateCancellation pins that Esc at any flat section aborts it rather than writing a
// partially-answered form.
func TestSimpleStepsPropagateCancellation(t *testing.T) {
	for name, step := range map[string]func(*userconfig.Config) error{
		"security": securityStep, "contributing": contributingStep,
		"conventions": conventionsStep, "scan": scanStep, "copyright": copyrightStep,
		"funding": fundingStep,
	} {
		t.Run(name, func(t *testing.T) {
			stubPrompts(t, formatAnswers{sections: func(string, []field) error { return errCancelled }})
			if err := step(&userconfig.Config{}); !errors.Is(err, errCancelled) {
				t.Errorf("%s error = %v, want errCancelled", name, err)
			}
		})
	}
}

// TestListStepsSplitTheirAnswers pins the comma convention on the three list fields, including that a blank
// answer clears the list to nil so the omitempty field disappears from the written file.
func TestListStepsSplitTheirAnswers(t *testing.T) {
	for _, tc := range []struct {
		name   string
		step   func(*userconfig.Config) error
		label  string
		answer string
		get    func(*userconfig.Config) []string
		want   []string
	}{
		{"funding github", fundingStep, fieldGitHub, sponsors, func(c *userconfig.Config) []string { return c.Funding.GitHub }, []string{"alice", "bob"}},
		{"funding custom", fundingStep, "custom", "https://a.example, https://b.example", func(c *userconfig.Config) []string { return c.Funding.Custom }, []string{"https://a.example", "https://b.example"}},
		{"scan private hosts", scanStep, fieldHosts, "src.example.com, db.example.com", func(c *userconfig.Config) []string { return c.Scan.PrivateHosts }, []string{hostSrc, "db.example.com"}},
		{"scan cleared", scanStep, fieldHosts, "   ", func(c *userconfig.Config) []string { return c.Scan.PrivateHosts }, nil},
		{"funding cleared", fundingStep, fieldGitHub, "", func(c *userconfig.Config) []string { return c.Funding.GitHub }, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &userconfig.Config{}
			stubPrompts(t, formatAnswers{sections: answerSections(map[string]string{tc.label: tc.answer})})
			assertNoErr(t, tc.step(cfg))
			if got := tc.get(cfg); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("%s = %#v, want %#v", tc.label, got, tc.want)
			}
		})
	}
}

// TestListStepsShowTheStoredListAsTheCurrentValue pins that an existing list is rendered back for editing
// rather than presented empty, which would hide what the config already holds.
func TestListStepsShowTheStoredListAsTheCurrentValue(t *testing.T) {
	for _, tc := range []struct {
		name string
		step func(*userconfig.Config) error
		fill func(*userconfig.Config)
		want map[string]string
	}{
		{
			name: "funding github", step: fundingStep,
			fill: func(c *userconfig.Config) { c.Funding.GitHub = []string{"alice", "bob"} },
			want: map[string]string{fieldGitHub: sponsors, "custom": ""},
		},
		{
			name: "scan private hosts", step: scanStep,
			fill: func(c *userconfig.Config) { c.Scan.PrivateHosts = []string{hostSrc} },
			want: map[string]string{fieldHosts: hostSrc},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &userconfig.Config{}
			tc.fill(cfg)
			var offered map[string]string
			stubPrompts(t, formatAnswers{sections: func(_ string, fields []field) error {
				offered = map[string]string{}
				for _, f := range fields {
					offered[f.Label] = f.Current
				}
				return nil
			}})
			assertNoErr(t, tc.step(cfg))
			for label, want := range tc.want {
				if got := offered[label]; got != want {
					t.Errorf("field %q prefilled %q, want %q", label, got, want)
				}
			}
		})
	}
}

// TestFundingStepNamesTheProvidersItDoesNotPromptFor pins the footer hint: the trimmed funding step must tell
// the user where the other 11 providers live, or those fields read as unsupported.
func TestFundingStepNamesTheProvidersItDoesNotPromptFor(t *testing.T) {
	var footer string
	stubPrompts(t, formatAnswers{sections: func(string, []field) error { return nil }, footer: func(f string) { footer = f }})
	assertNoErr(t, fundingStep(&userconfig.Config{}))
	for _, want := range []string{"patreon", "ko-fi", "open-collective", "cli.yaml"} {
		if !contains(footer, want) {
			t.Errorf("footer must mention %q, got %q", want, footer)
		}
	}
}

// TestWalkSectionsRunsEveryStepInOrder pins that one full pass reaches all eight sections, which is what makes
// a `setup` run actually populate the config rather than only its first page.
func TestWalkSectionsRunsEveryStepInOrder(t *testing.T) {
	var titles []string
	stubPrompts(t, formatAnswers{
		isEntity: false,
		noScan:   false,
		sections: func(title string, _ []field) error {
			titles = append(titles, title)
			return nil
		},
	})
	assertNoErr(t, walkSections(&userconfig.Config{}))
	for _, want := range []string{"Identity (person)", "Init defaults", "Funding", "Security", "Contributing", "Conventions", "Scan", "Copyright"} {
		if !containsAny(titles, want) {
			t.Errorf("walkSections never reached %q, saw %v", want, titles)
		}
	}
	if len(titles) != 8 {
		t.Errorf("walkSections ran %d sections, want 8: %v", len(titles), titles)
	}
}

// TestWalkSectionsStopsAtTheFirstCancellation pins that a cancel aborts the remaining sections: continuing
// would prompt the user for more answers after they asked to stop.
func TestWalkSectionsStopsAtTheFirstCancellation(t *testing.T) {
	seen := 0
	stubPrompts(t, formatAnswers{
		isEntity: false,
		sections: func(string, []field) error {
			seen++
			return errCancelled
		},
	})
	if err := walkSections(&userconfig.Config{}); !errors.Is(err, errCancelled) {
		t.Errorf("walkSections error = %v, want errCancelled", err)
	}
	if seen != 1 {
		t.Errorf("walkSections ran %d sections after a cancel, want it to stop at the first", seen)
	}
}

// TestOfferMigrationCleanupKeepsBothFilesByDefault pins the destructive-by-default rule: a quick Enter must
// leave the previous encoding in place, since the load order already prefers the new file.
func TestOfferMigrationCleanupKeepsBothFilesByDefault(t *testing.T) {
	oldPath, newPath := writeConfigPair(t, encTOML, "yaml")
	stubPrompts(t, formatAnswers{yesNo: func(_ string, defaultYes bool) (bool, error) {
		if defaultYes {
			t.Error("the cleanup prompt must default to no")
		}
		return false, nil
	}})

	assertNoErr(t, offerMigrationCleanup(oldPath, newPath))

	assertExists(t, oldPath, "the previous config must survive the default answer")
	assertExists(t, newPath, "the new config must survive")
}

// TestOfferMigrationCleanupRemovesTheOldFileWhenConfirmed pins the opt-in delete.
func TestOfferMigrationCleanupRemovesTheOldFileWhenConfirmed(t *testing.T) {
	oldPath, newPath := writeConfigPair(t, encTOML, "yaml")
	stubPrompts(t, formatAnswers{yesNo: func(string, bool) (bool, error) { return true, nil }})

	assertNoErr(t, offerMigrationCleanup(oldPath, newPath))

	assertMissing(t, oldPath, "confirming must remove the previous config")
	assertExists(t, newPath, "the new config must survive the cleanup")
}

// TestOfferMigrationCleanupTreatsCancellationAsKeep pins that Esc at the cleanup prompt is "leave it alone",
// not a failure — the write already succeeded by that point.
func TestOfferMigrationCleanupTreatsCancellationAsKeep(t *testing.T) {
	oldPath, newPath := writeConfigPair(t, encTOML, "yaml")
	stubPrompts(t, formatAnswers{yesNo: func(string, bool) (bool, error) { return false, errCancelled }})

	assertNoErr(t, offerMigrationCleanup(oldPath, newPath))

	assertExists(t, oldPath, "a cancelled cleanup must keep the previous config")
}

// TestOfferMigrationCleanupSurvivesAFailedRemoval pins that a delete the filesystem refuses is a warning, not
// a failed setup: the new config is already written and correct.
func TestOfferMigrationCleanupSurvivesAFailedRemoval(t *testing.T) {
	oldPath, newPath := writeConfigPair(t, encTOML, "yaml")
	stubPrompts(t, formatAnswers{yesNo: func(string, bool) (bool, error) { return true, nil }})
	// A directory where the file is expected: Remove refuses it, standing in for a permissions failure.
	if err := replaceWithDir(t, oldPath); err != nil {
		t.Fatalf("stage undeletable path: %v", err)
	}

	assertNoErr(t, offerMigrationCleanup(oldPath, newPath))

	assertExists(t, newPath, "the new config must survive a failed cleanup")
}

// TestOfferMigrationCleanupPropagatesAPromptFailure pins that a genuine prompt failure is returned rather than
// swallowed into a silent keep.
func TestOfferMigrationCleanupPropagatesAPromptFailure(t *testing.T) {
	oldPath, newPath := writeConfigPair(t, encTOML, "yaml")
	cause := errors.New("no tty")
	stubPrompts(t, formatAnswers{yesNo: func(string, bool) (bool, error) { return false, cause }})
	if err := offerMigrationCleanup(oldPath, newPath); !errors.Is(err, cause) {
		t.Errorf("offerMigrationCleanup error = %v, want %v", err, cause)
	}
}

// --- prompt stubs -------------------------------------------------------------------------------

// formatAnswers scripts every interactive prompt the wizard can open. A nil func means "fail the test if
// this prompt is reached", which is how a precedence rule proves it never opened the picker.
type formatAnswers struct {
	format    string
	formatErr error
	isEntity  bool
	entityErr error
	noScan    bool
	yesNo     func(title string, defaultYes bool) (bool, error)
	sections  func(title string, fields []field) error
	footer    func(footer string)
}

// stubPrompts installs the scripted answers for the rest of the test and restores the real drivers after.
func stubPrompts(t *testing.T, a formatAnswers) {
	t.Helper()
	runSectionPrompt = func(title string, fields []field, footer ...string) error {
		if a.footer != nil {
			a.footer(joinLines(footer))
		}
		if a.sections == nil {
			return nil // accept every answer as-is: the test is not about the form's contents
		}
		return a.sections(title, fields)
	}
	yesNoPrompt = func(title string, defaultYes bool) (bool, error) {
		if a.yesNo != nil {
			return a.yesNo(title, defaultYes)
		}
		return a.noScan, nil
	}
	pickFormat = func() (string, error) { return a.format, a.formatErr }
	pickEntityKind = func(bool) (bool, error) { return a.isEntity, a.entityErr }
	t.Cleanup(func() {
		runSectionPrompt = runSection
		yesNoPrompt = yesNo
		pickFormat = promptFormat
		pickEntityKind = personOrEntity
	})
}

// answerSections builds a section driver that types answers into the fields it is offered, by label.
func answerSections(answers map[string]string) func(string, []field) error {
	return func(_ string, fields []field) error {
		for _, f := range fields {
			if v, ok := answers[f.Label]; ok {
				f.Setter(v)
			}
		}
		return nil
	}
}

// keepPrefilled accepts the form as displayed, which is what a user pressing Enter straight through does.
func keepPrefilled(_ *userconfig.Config) func(string, []field) error {
	return func(_ string, _ []field) error { return nil }
}

func labelsOf(fields []field) []string {
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		out = append(out, f.Label)
	}
	return out
}

func assertLabels(t *testing.T, offered, want, absent []string) {
	t.Helper()
	for _, w := range want {
		if !containsLabel(offered, w) {
			t.Errorf("field %q must be offered, saw %v", w, offered)
		}
	}
	for _, a := range absent {
		if containsLabel(offered, a) {
			t.Errorf("field %q must NOT be offered, saw %v", a, offered)
		}
	}
}

func assertNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
