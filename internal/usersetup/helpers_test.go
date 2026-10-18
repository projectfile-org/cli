// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package usersetup

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"kiota.ch/projectfile/core/v2/pkg/selector"
)

// Fixture vocabulary shared across the package's tests. goconst counts these across files, so the spellings
// that carry meaning — a key name, an encoding, a person's name — are named once.
const (
	familyNames = "family-names"
	givenNames  = "given-names"
	fieldOrcid  = "orcid"
	fieldName   = "name"
	fieldGitHub = "github"
	fieldHosts  = "private-hosts"
	kindPerson  = "person"
	hostSrc     = "src.example.com"

	encTOML = "toml"
	encJSON = "json"

	fixtureAcme = "acme"

	gitLovelace = "Ada Lovelace"
	gitFamily   = "Lovelace"
	gitGiven    = "Ada"

	keyTab     = "tab"
	keyShiftTb = "shift+tab"
)

func contains(haystack, needle string) bool { return strings.Contains(haystack, needle) }

func containsAny(haystack []string, needle string) bool {
	for _, h := range haystack {
		if strings.Contains(h, needle) {
			return true
		}
	}
	return false
}

// containsLabel reports exact membership, which a field assertion needs: a substring test would call
// "gpg-key" a match for the absent "name" field.
func containsLabel(labels []string, want string) bool {
	return slices.Contains(labels, want)
}

func joinLines(ss []string) string { return strings.Join(ss, "\n") }

// errCancelledFromSelector is what core's picker returns on Esc; the wizard has to recognise it as its own
// abort sentinel rather than as a failure.
func errCancelledFromSelector() error { return selector.ErrCancelled }

func writeConfigPair(t *testing.T, oldExt, newExt string) (oldPath, newPath string) {
	t.Helper()
	dir := t.TempDir()
	oldPath = filepath.Join(dir, "cli."+oldExt)
	newPath = filepath.Join(dir, "cli."+newExt)
	for _, p := range []string{oldPath, newPath} {
		if err := os.WriteFile(p, []byte("identity:\n  name: demo\n"), 0o600); err != nil {
			t.Fatalf("write %s: %v", p, err)
		}
	}
	return oldPath, newPath
}

// replaceWithDir swaps a file for a directory of the same name: os.Remove refuses a non-empty directory, so
// this stands in for a deletion the filesystem will not perform.
func replaceWithDir(t *testing.T, path string) error {
	t.Helper()
	if err := os.Remove(path); err != nil {
		return err
	}
	return os.Mkdir(path, 0o700)
}

func assertExists(t *testing.T, path, why string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("%s: %v", why, err)
	}
}

func assertMissing(t *testing.T, path, why string) {
	t.Helper()
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("%s: %s still present (%v)", why, filepath.Base(path), err)
	}
}
