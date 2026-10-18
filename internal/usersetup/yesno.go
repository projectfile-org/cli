// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package usersetup

import (
	"errors"
	"fmt"

	"kiota.ch/projectfile/core/v2/pkg/selector"
)

// yesNo wraps the shared selector with a two-option Yes/No list. The chosen
// default option leads so pressing Enter immediately accepts it — important
// for the no_scan toggle (default = whatever the user already had) and the
// migration-cleanup prompt (default = no, never destroy without intent).
//
// We use selector.Run rather than a custom bubbletea model because:
//   - bubbletea has no native checkbox widget, and reusing the one we built
//     for the format picker keeps the wizard visually coherent.
//   - selector navigation (up/down/j/k/enter/esc) already matches every
//     other interactive prompt in pf-cli — muscle memory carries over.
//
// Cancellation surfaces as errCancelled (same sentinel sections use) so the
// orchestrator's "user aborted, no changes" branch handles both paths
// identically.
// yesNoOption is one row of the two-option list.
type yesNoOption struct {
	label string
	value bool
}

// yesNoOptions orders the rows so the caller's default leads, which is what makes Enter a zero-press accept.
func yesNoOptions(defaultYes bool) []yesNoOption {
	if defaultYes {
		return []yesNoOption{{"yes", true}, {"no", false}}
	}
	return []yesNoOption{{"no", false}, {"yes", true}}
}

func yesNo(title string, defaultYes bool) (bool, error) {
	choice, err := selector.Run(selector.Choices[yesNoOption]{
		Title:  title,
		Items:  yesNoOptions(defaultYes),
		Label:  func(o yesNoOption) string { return o.label },
		Detail: func(_ yesNoOption) string { return "" },
	})
	if err != nil {
		if errors.Is(err, selector.ErrCancelled) {
			return false, errCancelled
		}
		return false, fmt.Errorf("yes/no prompt: %w", err)
	}
	return choice.value, nil
}
