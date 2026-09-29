// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import "testing"

func TestFailOnRejectsEmptyValue(t *testing.T) {
	var f failOnFlagValue
	if err := f.Set(""); err == nil {
		t.Fatal("empty --fail-on was accepted")
	}
}
