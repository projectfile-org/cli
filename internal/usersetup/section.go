// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package usersetup

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"kiota.ch/projectfile/core/v2/pkg/selector"
)

// errCancelled is returned by runSection when the user hits Esc / Ctrl+C.
// Sentinel so the orchestrator can distinguish "user aborted" from "form
// failed to render".
var errCancelled = errors.New("cancelled")

// Color palette mirrors scaffold/prompt.go so the two wizards feel like
// the same product. Kept inline rather than imported because both
// scaffold and selector also keep their own copies — a project-wide
// theme refactor is a separate concern.
var (
	highlight = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	dimmed    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	label     = lipgloss.NewStyle().Bold(true)
	heading   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
)

// field describes one prompt inside a section. Current is the pre-filled
// value shown when the user revisits an existing config; Setter is called
// on submit with whatever the user typed (including blank — clearing a
// previously-set field is a legitimate edit).
type field struct {
	Label       string
	Description string
	Current     string
	Setter      func(string)
}

// sectionModel is the bubbletea model behind runSection. View renders one
// labelled input per field, vertically; navigation is Tab / Shift+Tab.
// footer is an optional dimmed line shown after all inputs — used today by
// the trimmed funding step to hint at the providers it doesn't prompt for.
type sectionModel struct {
	title     string
	multi     *selector.MultiInput
	fields    []field
	footer    string
	submitted bool
}

func newSectionModel(title string, fields []field, footer string) sectionModel {
	inputs := make([]textinput.Model, len(fields))
	for i, f := range fields {
		ti := textinput.New()
		ti.Width = 50
		ti.Placeholder = f.Description
		ti.PromptStyle = highlight
		// Pre-fill with the current value so the user sees "what's there"
		// and edits selectively. SetValue does not move the cursor; the
		// CursorEnd call positions it at end-of-text for natural editing.
		if f.Current != "" {
			ti.SetValue(f.Current)
			ti.CursorEnd()
		}
		inputs[i] = ti
	}
	return sectionModel{
		title:  title,
		multi:  selector.NewMultiInput(inputs, nil),
		fields: fields,
		footer: footer,
	}
}

func (m sectionModel) Init() tea.Cmd { return textinput.Blink }

func (m sectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			m.submitted = true
			return m, tea.Quit
		case tea.KeyTab, tea.KeyDown:
			m.multi.Next()
			return m, nil
		case tea.KeyShiftTab, tea.KeyUp:
			m.multi.Prev()
			return m, nil
		}
	}
	cmd := m.multi.Update(msg)
	return m, cmd
}

func (m sectionModel) View() string {
	var b strings.Builder
	fmt.Fprintf(&b, "\n  %s\n", heading.Render(m.title))
	fmt.Fprintf(&b, "  %s\n\n", dimmed.Render("(enter to confirm, tab/↑↓ to navigate, esc to cancel)"))
	focused := m.multi.Focused()
	for i, ti := range m.multi.Inputs() {
		marker := "  "
		if focused == i {
			marker = highlight.Render("> ")
		}
		fmt.Fprintf(&b, "  %s %s\n", marker, label.Render(m.fields[i].Label+":"))
		fmt.Fprintf(&b, "    %s\n", ti.View())
		if d := m.fields[i].Description; d != "" {
			fmt.Fprintf(&b, "    %s\n", dimmed.Render(d))
		}
		b.WriteString("\n")
	}
	if m.footer != "" {
		fmt.Fprintf(&b, "  %s\n\n", dimmed.Render(m.footer))
	}
	return b.String()
}

// runSection drives the bubbletea program and applies field setters on
// successful submit. Returns errCancelled when the user aborts so the
// orchestrator can propagate that as a clean exit, not an error.
//
// footer is variadic so the common no-footer call stays a two-arg signature;
// when supplied, multiple strings are joined with newlines and rendered
// dimmed below the last input — used by the trimmed funding step to point
// at the named providers it doesn't prompt for.
func runSection(title string, fields []field, footer ...string) error {
	if len(fields) == 0 {
		return nil
	}
	foot := strings.Join(footer, "\n")
	prog := tea.NewProgram(newSectionModel(title, fields, foot))
	out, err := prog.Run()
	if err != nil {
		return fmt.Errorf("prompt: %w", err)
	}
	m := out.(sectionModel)
	if !m.submitted {
		return errCancelled
	}
	// The two slices are parallel by construction (newSectionModel builds
	// one input per field), but bound the loop by BOTH lengths so the
	// fields[i] / inputs[i] indexing is provably in range even if a future
	// change breaks that invariant.
	inputs := m.multi.Inputs()
	for i := 0; i < len(fields) && i < len(inputs); i++ {
		fields[i].Setter(strings.TrimSpace(inputs[i].Value()))
	}
	return nil
}
