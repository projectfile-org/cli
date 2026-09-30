// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetRefusesAValueTheFieldCannotHold(t *testing.T) {
	path := filepath.Join(t.TempDir(), "projectfile.yaml")
	before := []byte("identity:\n  name: demo\n")
	require.NoError(t, os.WriteFile(path, before, 0o600))

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"set", "--path-file", path, "identity.name", "true"})
	err := rootCmd.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), `quote it: pf-cli set identity.name '"true"'`)
	after, readErr := os.ReadFile(path)
	require.NoError(t, readErr)
	assert.Equal(t, before, after, "a refused set leaves the file byte-identical")
}
