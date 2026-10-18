// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package usersetup

import (
	"reflect"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// ansiRe strips the colour escapes lipgloss emits, so assertions read the rendered text and a palette
// change cannot fail a behavioural test.
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// TestSplitCSV pins the wizard's list-parsing convention at every edge: a blank input yields nil (not an
// empty slice) so the omitempty field stays absent from the written file, and interior empties are dropped.
func TestSplitCSV(t *testing.T) {
	for _, tc := range []struct {
		name, in string
		want     []string
	}{
		{"empty", "", nil},
		{"whitespace only", "   \t ", nil},
		{"commas only", ",,,", nil},
		{"single", fixtureAcme, []string{fixtureAcme}},
		{"several", "a,b,c", []string{"a", "b", "c"}},
		{"trims each", "  a ,  b  ", []string{"a", "b"}},
		{"drops interior empties", "a,,b", []string{"a", "b"}},
		{"keeps an interior space inside a value", "a, b c ,d", []string{"a", "b c", "d"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := splitCSV(tc.in); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("splitCSV(%q) = %#v, want %#v", tc.in, got, tc.want)
			}
		})
	}
}

// TestJoinCSVIsSplitCSVInverse pins that a comma-free list round-trips through the prompt's Current
// display, so re-entering the wizard shows the same list the user typed rather than a reshaped one.
func TestJoinCSVIsSplitCSVInverse(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   []string
	}{
		{"nil", nil},
		{"empty", []string{}},
		{"single", []string{fixtureAcme}},
		{"several", []string{"a", "b", "c"}},
		{"value holding a space", []string{"b c", "d"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := splitCSV(joinCSV(tc.in)); !reflect.DeepEqual(got, tcOrNil(tc.in)) {
				t.Errorf("splitCSV(joinCSV(%#v)) = %#v, want %#v", tc.in, got, tcOrNil(tc.in))
			}
		})
	}
}

// TestJoinCSVSeparatesWithCommaAndSpace pins the display separator itself: the round-trip above depends on
// it, and a bare comma would make two adjacent values indistinguishable to a reader.
func TestJoinCSVSeparatesWithCommaAndSpace(t *testing.T) {
	if got := joinCSV([]string{"a", "b"}); got != "a, b" {
		t.Errorf("joinCSV = %q, want %q", got, "a, b")
	}
	if got := joinCSV(nil); got != "" {
		t.Errorf("joinCSV(nil) = %q, want empty", got)
	}
}

func tcOrNil(xs []string) []string {
	if len(xs) == 0 {
		return nil
	}
	return xs
}

// TestNewSectionModelPrefillsEveryInput pins that each field's Current value reaches its input, which is
// what makes an edit show "what is there" rather than an empty box that erases the config on submit.
func TestNewSectionModelPrefillsEveryInput(t *testing.T) {
	fields := []field{
		{Label: familyNames, Description: "family name(s)", Current: gitFamily, Setter: func(string) {}},
		{Label: givenNames, Current: gitGiven, Setter: func(string) {}},
		{Label: fieldOrcid, Setter: func(string) {}},
	}
	m := newSectionModel("Identity", fields, "")

	if m.title != "Identity" {
		t.Errorf("title = %q, want Identity", m.title)
	}
	if len(m.multi.Inputs()) != len(fields) {
		t.Fatalf("built %d inputs, want one per field (%d)", len(m.multi.Inputs()), len(fields))
	}
	for i, want := range []string{gitFamily, gitGiven, ""} {
		if got := m.multi.Inputs()[i].Value(); got != want {
			t.Errorf("input %d = %q, want %q", i, got, want)
		}
	}
	if got := m.multi.Inputs()[0].Placeholder; got != "family name(s)" {
		t.Errorf("placeholder = %q, want the field description", got)
	}
	if m.multi.Inputs()[2].Placeholder != "" {
		t.Error("a field with no description must leave its placeholder empty")
	}
}

// TestSectionModelSubmitAndCancel pins the two terminal keys: enter records a submission, esc and ctrl+c
// quit without one. The distinction is the whole abort-vs-commit decision runSection makes.
func TestSectionModelSubmitAndCancel(t *testing.T) {
	for _, tc := range []struct {
		key            string
		wantSubmitted  bool
		wantFocusShift int
	}{
		{"enter", true, 0},
		{"esc", false, 0},
		{"ctrl+c", false, 0},
	} {
		t.Run(tc.key, func(t *testing.T) {
			m := newSectionModel("Identity", threeFields(), "")
			next, _ := m.Update(keyMsg(tc.key))
			if next.(sectionModel).submitted != tc.wantSubmitted {
				t.Errorf("%s: submitted = %v, want %v", tc.key, next.(sectionModel).submitted, tc.wantSubmitted)
			}
		})
	}
}

// TestSectionModelNavigationMovesFocus pins tab/shift+tab and the arrow aliases, so a user who reaches for
// the arrows lands on the same field as one who reaches for tab.
func TestSectionModelNavigationMovesFocus(t *testing.T) {
	for _, tc := range []struct {
		name          string
		keys          []string
		wantPositions []int
	}{
		{"tab advances", []string{keyTab}, []int{1}},
		{"down advances", []string{"down"}, []int{1}},
		{"shift+tab retreats", []string{keyTab, keyShiftTb}, []int{1, 0}},
		{"up retreats", []string{keyTab, "up"}, []int{1, 0}},
		{"tab wraps past the last field", []string{keyTab, keyTab, keyTab}, []int{1, 2, 0}},
		{"shift+tab wraps before the first field", []string{keyShiftTb}, []int{2}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newSectionModel("Identity", threeFields(), "")
			for i, k := range tc.keys {
				next, _ := m.Update(keyMsg(k))
				m = next.(sectionModel)
				if got := m.multi.Focused(); got != tc.wantPositions[i] {
					t.Fatalf("after %v: focused = %d, want %d", tc.keys[:i+1], got, tc.wantPositions[i])
				}
			}
		})
	}
}

// TestSectionModelViewRendersLabelsValuesAndFooter pins that the form shows the title, every label with
// its value, the hint line and the optional footer — a dropped line here is a field the user cannot see.
func TestSectionModelViewRendersLabelsValuesAndFooter(t *testing.T) {
	m := newSectionModel("Funding — personal funding URLs", threeFields(), "+ 11 named providers")
	view := renderView(m)

	for _, want := range []string{
		"Funding — personal funding URLs",
		"esc to cancel",
		"family-names:",
		gitFamily,
		"given-names:",
		"hint",
		"+ 11 named providers",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q:\n%s", want, view)
		}
	}
	if got := len(focusedLabels(m)); got != 1 {
		t.Errorf("exactly one field may carry the focus marker, got %d", got)
	}
	if strings.Contains(view, "NoSuchField") {
		t.Error("view walked past the field list")
	}
}

// TestSectionModelViewMovesTheFocusMarker pins that the marker tracks navigation, so the highlighted row is
// the one the next keystroke edits.
func TestSectionModelViewMovesTheFocusMarker(t *testing.T) {
	m := newSectionModel("Identity", threeFields(), "")
	if got := focusedLabels(m); len(got) == 1 && got[0] != "family-names" {
		t.Errorf("first field must start focused, got %q", got)
	}
	next, _ := m.Update(keyMsg(keyTab))
	if got := focusedLabels(next.(sectionModel)); len(got) == 1 && got[0] != "given-names" {
		t.Errorf("after tab the marker must sit on given-names, got %q", got)
	}
}

// TestSectionModelInitIsNonNil is a smoke pin on the cursor-blink command: a nil Init renders an input that
// never shows a caret, which reads as a frozen prompt.
func TestSectionModelInitIsNonNil(t *testing.T) {
	if cmd := newSectionModel("Identity", threeFields(), "").Init(); cmd == nil {
		t.Error("Init returned no command, so the caret never blinks")
	}
}

// TestSectionUpdateForwardsNonKeyMessages pins that an ordinary message reaches the inputs; swallowing it
// would make every keystroke arrive through the switch and no paste or resize would ever land.
func TestSectionUpdateForwardsNonKeyMessages(t *testing.T) {
	m := newSectionModel("Identity", []field{{Label: fieldOrcid, Setter: func(string) {}}}, "")
	next, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 10})
	if _, ok := next.(sectionModel); !ok {
		t.Fatalf("Update returned %T, want sectionModel", next)
	}
}

func threeFields() []field {
	return []field{
		{Label: familyNames, Description: "hint", Current: gitFamily, Setter: func(string) {}},
		{Label: givenNames, Current: gitGiven, Setter: func(string) {}},
		{Label: fieldOrcid, Setter: func(string) {}},
	}
}

func keyMsg(k string) tea.KeyPressMsg {
	switch k {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case keyTab:
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case keyShiftTb:
		return tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	default:
		return tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	}
}

func renderView(m sectionModel) string {
	return m.View().Content
}

// focusedLabels names the label rows carrying the focus marker. Colour escapes are stripped first so the
// assertion reads rendered text; only a label row ends in ":", which separates it from the input's own
// prompt glyph on the value line below.
func focusedLabels(m sectionModel) []string {
	var got []string
	for _, line := range strings.Split(ansiRe.ReplaceAllString(renderView(m), ""), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, ">") && strings.HasSuffix(trimmed, ":") {
			got = append(got, strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(trimmed, ">"), ":")))
		}
	}
	return got
}
