// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kiota.ch/projectfile/core/v2/pkg/genlog"
	"kiota.ch/projectfile/core/v2/pkg/projectfile"
)

func TestIncludesPin_WritesMappingAndResolves(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	const body = "keywords: [pinned]\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	dir := t.TempDir()
	path := writeFile(t, dir, "projectfile.yaml", "$schema: https://projectfile.org/schema/v1.json\nidentity:\n  namespace: org.example\n  name: demo\nincludes:\n  - "+srv.URL+"/base.yaml\n")
	genlog.SetOutput(&bytes.Buffer{})
	t.Cleanup(func() { genlog.SetOutput(os.Stderr); includesPinFile = "" })
	rootCmd.SetArgs([]string{cmdIncludes, "pin", flagFile, path})
	require.NoError(t, rootCmd.Execute())
	written, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(written), "sha256: "+projectfile.IncludeDigest([]byte(body)))
	doc, err := projectfile.ReadFromPath(path)
	require.NoError(t, err)
	assert.Equal(t, []string{"pinned"}, doc.Keywords)
}

func TestIncludeRef_BothForms(t *testing.T) {
	assert.Equal(t, includeA, includeRef(includeA))
	assert.Equal(t, includeA, includeRef(map[string]any{"url": includeA, "sha256": "00"}))
	assert.Empty(t, includeRef(42))
}
