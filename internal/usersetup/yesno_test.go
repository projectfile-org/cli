// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package usersetup

import (
	"testing"
)

// TestYesNoOptionsLeadWithTheDefaultOption pins the zero-press contract: the option the user already holds
// is the first row, so Enter accepts it. Leading with the opposite would silently invert every default.
func TestYesNoOptionsLeadWithTheDefaultOption(t *testing.T) {
	for _, tc := range []struct {
		name       string
		defaultYes bool
		wantLabels []string
		wantValues []bool
	}{
		{"default no leads with no", false, []string{"no", "yes"}, []bool{false, true}},
		{"default yes leads with yes", true, []string{"yes", "no"}, []bool{true, false}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			items := yesNoOptions(tc.defaultYes)
			if len(items) != 2 {
				t.Fatalf("offered %d rows, want exactly the two options", len(items))
			}
			for i, want := range tc.wantLabels {
				if items[i].label != want {
					t.Errorf("row %d = %q, want %q", i, items[i].label, want)
				}
				if items[i].value != tc.wantValues[i] {
					t.Errorf("row %q carries value %v, want %v", items[i].label, items[i].value, tc.wantValues[i])
				}
			}
		})
	}
}

// TestYesNoOptionsAreDistinct pins that the two rows are the two distinct answers: a list offering the same
// choice twice, or naming one row twice, makes the prompt ambiguous about what Enter will pick.
func TestYesNoOptionsAreDistinct(t *testing.T) {
	for _, defaultYes := range []bool{false, true} {
		items := yesNoOptions(defaultYes)
		if items[0].label == items[1].label || items[0].value == items[1].value {
			t.Errorf("defaultYes=%v: the two rows must be distinct answers, got %+v", defaultYes, items)
		}
	}
}

// TestKindOptionsLeadWithTheRecordedKind pins that re-running setup pre-selects the kind the config already
// records rather than the package default, so a returning user is not silently asked to flip back.
func TestKindOptionsLeadWithTheRecordedKind(t *testing.T) {
	for _, tc := range []struct {
		name    string
		current bool
		want    []string
	}{
		{"a person config leads with person", false, []string{kindPerson, "organization"}},
		{"an entity config leads with organization", true, []string{"organization", kindPerson}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			items := kindOptions(tc.current)
			if len(items) != 2 {
				t.Fatalf("offered %d rows, want exactly the two kinds", len(items))
			}
			for i, want := range tc.want {
				if items[i].label != want {
					t.Errorf("row %d = %q, want %q", i, items[i].label, want)
				}
			}
			if items[0].value != tc.current {
				t.Errorf("leading row %q carries value %v, want the recorded %v", items[0].label, items[0].value, tc.current)
			}
		})
	}
}

// TestKindOptionsExplainEachChoice pins that both kinds carry a description: "person" / "organization" is an
// ambiguous pair on its own, and the choice decides whether the split-name fields appear at all.
func TestKindOptionsExplainEachChoice(t *testing.T) {
	seen := map[string]bool{}
	for _, o := range kindOptions(false) {
		if o.desc == "" {
			t.Errorf("row %q carries no description", o.label)
		}
		if seen[o.label] {
			t.Errorf("row %q appears twice", o.label)
		}
		seen[o.label] = true
	}
}

// TestFormatOptionsLeadWithYAML pins the zero-press default of the encoding picker: YAML first, so a user who
// just presses Enter through a fresh setup gets the recommended encoding.
func TestFormatOptionsLeadWithYAML(t *testing.T) {
	items := formatOptions()
	want := []string{formatYAML, "toml", "json"}
	if len(items) != len(want) {
		t.Fatalf("offered %d encodings, want %d", len(items), len(want))
	}
	for i, w := range want {
		if items[i].name != w {
			t.Errorf("row %d = %q, want %q", i, items[i].name, w)
		}
	}
	if items[0].desc == "" {
		t.Error("the leading YAML row must say it is the recommended choice")
	}
}
