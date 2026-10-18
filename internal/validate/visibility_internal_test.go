// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package validate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// privateHostConfig stages a user config that marks one host private, which is the only evidence the advisory fires on.
func privateHostConfig(t *testing.T, host string) {
	t.Helper()
	dir := t.TempDir()
	body := "scan:\n  private-hosts:\n    - " + host + "\n"
	if err := os.MkdirAll(filepath.Join(dir, "projectfile"), 0o700); err != nil {
		t.Fatalf("stage config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "projectfile", "cli.yaml"), []byte(body), 0o600); err != nil {
		t.Fatalf("stage config: %v", err)
	}
	t.Setenv("XDG_CONFIG_HOME", dir)
}

// TestHostOfReadsTheHostPart pins the three answers the advisory relies on: a lowercased host, an unparsable URL's silence and an absent one.
func TestHostOfReadsTheHostPart(t *testing.T) {
	assert.Equal(t, "kiota.ch", hostOf("https://Kiota.CH/projectfile/cli"))
	assert.Equal(t, "src.example.com", hostOf("ssh://git@src.example.com:2222/projectfile/cli.git"))
	assert.Empty(t, hostOf("://no-scheme"))
	assert.Empty(t, hostOf(""))
}

// TestVisibilityAdvisoriesIgnoreANonSourceLink pins the type filter: a documentation link pointing at a private host is not a contradiction about visibility.
func TestVisibilityAdvisoriesIgnoreANonSourceLink(t *testing.T) {
	privateHostConfig(t, "git.internal.example")
	doc := map[string]any{
		keyIdentity: map[string]any{keyVisibility: visPrivate},
		keyLinks:    []any{map[string]any{keyType: "documentation", keyURL: "https://git.internal.example/docs"}},
	}
	assert.Empty(t, VisibilityAdvisories(doc))
}

// TestVisibilityAdvisoriesIgnoreAPublicHost pins that the advisory needs configured evidence; an ordinary forge is not a finding.
func TestVisibilityAdvisoriesIgnoreAPublicHost(t *testing.T) {
	privateHostConfig(t, "git.internal.example")
	doc := map[string]any{
		keyIdentity: map[string]any{keyVisibility: visPrivate},
		keyLinks:    []any{map[string]any{keyType: linkSource, keyURL: "https://github.com/projectfile-org/cli"}},
	}
	assert.Empty(t, VisibilityAdvisories(doc))
}

// TestVisibilityAdvisoriesIgnoreALinkWithoutAURL pins the empty-host skip, which is also what a hand-edited link missing its url produces.
func TestVisibilityAdvisoriesIgnoreALinkWithoutAURL(t *testing.T) {
	privateHostConfig(t, "git.internal.example")
	doc := map[string]any{
		keyIdentity: map[string]any{keyVisibility: visPrivate},
		keyLinks:    []any{map[string]any{keyType: linkSource}},
	}
	assert.Empty(t, VisibilityAdvisories(doc))
}

// TestVisibilityAdvisoriesNameAPrivateForge pins the finding itself: a private project whose source link points at a host the user configured as private did not mean to.
func TestVisibilityAdvisoriesNameAPrivateForge(t *testing.T) {
	privateHostConfig(t, "git.internal.example")
	doc := map[string]any{
		keyIdentity: map[string]any{keyVisibility: visPrivate},
		keyLinks:    []any{map[string]any{keyType: linkSource, keyURL: "https://git.internal.example/projectfile/cli"}},
	}
	got := VisibilityAdvisories(doc)
	if len(got) != 1 {
		t.Fatalf("want one advisory, got %v", got)
	}
	assert.Contains(t, got[0], "git.internal.example")
	assert.Contains(t, got[0], "identity.visibility is private")
}
