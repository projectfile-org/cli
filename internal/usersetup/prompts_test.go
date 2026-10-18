// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package usersetup

import (
	"os"
	"testing"

	"kiota.ch/projectfile/core/v2/pkg/userconfig"
)

// requireNoTTY skips when a terminal is attached: the real prompt drivers open it and would wait for keystrokes forever.
func requireNoTTY(t *testing.T) {
	t.Helper()
	f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err == nil {
		f.Close()
		t.Skip("a terminal is attached; driving the real prompt would block on input")
	}
}

// TestRunSectionWithNoFieldsIsANoOp pins the empty form: a section with nothing to ask opens no terminal at all.
func TestRunSectionWithNoFieldsIsANoOp(t *testing.T) {
	assertNoErr(t, runSection("Nothing", nil))
}

// TestRunSectionReportsAMissingTerminal pins that a section which cannot open one reports the failure in its own voice instead of panicking.
func TestRunSectionReportsAMissingTerminal(t *testing.T) {
	requireNoTTY(t)
	err := runSection("Identity", []field{{Label: fieldName, Setter: func(string) {}}})
	if err == nil {
		t.Fatal("a section with no terminal must fail, not answer")
	}
	if !contains(err.Error(), "prompt:") {
		t.Errorf("error %q must name the prompt it came from", err)
	}
}

// TestYesNoReportsAMissingTerminal pins the same contract for the two-option list.
func TestYesNoReportsAMissingTerminal(t *testing.T) {
	requireNoTTY(t)
	if _, err := yesNo("Skip scanners?", true); err == nil {
		t.Fatal("a yes/no prompt with no terminal must fail")
	}
}

// TestPromptFormatReportsAMissingTerminal pins that the encoding picker surfaces the selector's own error unchanged, so the wizard can recognise a cancel.
func TestPromptFormatReportsAMissingTerminal(t *testing.T) {
	requireNoTTY(t)
	format, err := promptFormat()
	if err == nil {
		t.Fatalf("the picker with no terminal must fail, got format %q", format)
	}
	if format != "" {
		t.Errorf("a failed picker answers no format, got %q", format)
	}
}

// TestPersonOrEntityReportsAMissingTerminal pins the discriminator's error, which the identity step propagates as a failed wizard.
func TestPersonOrEntityReportsAMissingTerminal(t *testing.T) {
	requireNoTTY(t)
	isEntity, err := personOrEntity(false)
	if err == nil {
		t.Fatalf("the discriminator with no terminal must fail, got %v", isEntity)
	}
	if !contains(err.Error(), "identity kind prompt") {
		t.Errorf("error %q must name the prompt it came from", err)
	}
}

// TestKindOptionsLeadWithTheStoredChoice pins the re-run affordance: a config that already records an organization opens the list on the organization.
func TestKindOptionsLeadWithTheStoredChoice(t *testing.T) {
	entity := kindOptions(true)
	if len(entity) != 2 || !entity[0].value || entity[1].value {
		t.Errorf("stored entity must lead, got %+v", entity)
	}
	person := kindOptions(false)
	if !person[0].value == false || person[1].value != true {
		t.Errorf("stored person must lead, got %+v", person)
	}
}

// TestFormatOptionsCarryEveryEncoding pins that the picker offers the three encodings the writer can produce.
func TestFormatOptionsCarryEveryEncoding(t *testing.T) {
	names := make([]string, 0, 3)
	for _, o := range formatOptions() {
		names = append(names, o.name)
		if o.desc == "" {
			t.Errorf("encoding %q has no description to show", o.name)
		}
	}
	for _, want := range []string{formatYAML, encTOML, encJSON} {
		if !containsAny(names, want) {
			t.Errorf("the picker is missing %q, got %v", want, names)
		}
	}
}

// TestYesNoOptionsPutTheDefaultFirst pins the zero-press affordance the toggles depend on.
func TestYesNoOptionsPutTheDefaultFirst(t *testing.T) {
	yes := yesNoOptions(true)
	if !yes[0].value || yes[1].value {
		t.Errorf("a default-yes list must lead with yes, got %+v", yes)
	}
	no := yesNoOptions(false)
	if no[0].value || !no[1].value {
		t.Errorf("a default-no list must lead with no, got %+v", no)
	}
}

// TestIdentityStepStoresEveryPersonField pins that the person branch writes each answer to its own field; a setter wired to the wrong field leaves the answer on screen and out of the file.
func TestIdentityStepStoresEveryPersonField(t *testing.T) {
	stubPrompts(t, formatAnswers{
		isEntity: false,
		sections: answerSections(map[string]string{
			familyNames:      "Lovelace",
			givenNames:       "Ada",
			"email":          "ada@example.org",
			fieldOrcid:       orcidAda,
			"url":            "https://ada.example",
			fieldAffiliation: "Analytical Engine Society",
			"gpg-key":        "ABCDEF0123456789",
		}),
	})

	cfg := &userconfig.Config{}
	assertNoErr(t, identityStep(cfg))
	id := cfg.Identity
	if id.FamilyNames != "Lovelace" || id.GivenNames != gitGiven {
		t.Errorf("split name = %q/%q, want Lovelace/Ada", id.FamilyNames, id.GivenNames)
	}
	if id.Email != "ada@example.org" || id.Orcid != orcidAda || id.URL != "https://ada.example" {
		t.Errorf("contact fields not stored: %+v", id)
	}
	if id.Affiliation != "Analytical Engine Society" || id.GPGKey != gpgAda {
		t.Errorf("affiliation or key not stored: %+v", id)
	}
	if id.Name != "" {
		t.Errorf("the person branch must leave no entity name, got %q", id.Name)
	}
}

// TestIdentityStepStoresEveryEntityField pins the same for the organization branch.
func TestIdentityStepStoresEveryEntityField(t *testing.T) {
	stubPrompts(t, formatAnswers{
		isEntity: true,
		sections: answerSections(map[string]string{
			fieldName:  "Acme Foundation",
			"email":    "hello@acme.example",
			fieldOrcid: "0000-0002-3456-7890",
			"url":      "https://acme.example",
			"gpg-key":  "0123456789ABCDEF",
		}),
	})

	cfg := &userconfig.Config{}
	cfg.Identity.FamilyNames = staleValue
	cfg.Identity.GivenNames = staleValue
	cfg.Identity.Affiliation = staleValue
	assertNoErr(t, identityStep(cfg))
	id := cfg.Identity
	if id.Name != "Acme Foundation" || id.Email != "hello@acme.example" || id.Orcid != "0000-0002-3456-7890" {
		t.Errorf("entity fields not stored: %+v", id)
	}
	if id.URL != "https://acme.example" || id.GPGKey != "0123456789ABCDEF" {
		t.Errorf("entity url or key not stored: %+v", id)
	}
	if id.FamilyNames != "" || id.GivenNames != "" || id.Affiliation != "" {
		t.Errorf("the entity branch must clear the person-only fields, got %+v", id)
	}
}

// TestInitDefaultsStepStoresEveryField pins the init pre-fills, including the boolean toggle the step pulls out of the form.
func TestInitDefaultsStepStoresEveryField(t *testing.T) {
	stubPrompts(t, formatAnswers{
		sections: answerSections(map[string]string{
			"format":            formatYAML,
			"namespace-default": "org.example",
			"license-default":   "MIT",
		}),
		yesNo: func(string, bool) (bool, error) { return true, nil },
	})

	cfg := &userconfig.Config{}
	assertNoErr(t, initDefaultsStep(cfg))
	in := cfg.Init
	if in.Format != formatYAML || in.NamespaceDefault != "org.example" || in.LicenseDefault != "MIT" {
		t.Errorf("init defaults not stored: %+v", in)
	}
	if !in.NoScan {
		t.Error("the no-scan toggle must store the answer the list gave")
	}
}
