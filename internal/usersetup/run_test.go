// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package usersetup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"kiota.ch/projectfile/core/v2/pkg/userconfig"
)

// TestRunWritesTheChosenEncoding pins the end-to-end contract: a completed wizard lands a readable config at
// the path its extension names, with the answers the forms collected.
func TestRunWritesTheChosenEncoding(t *testing.T) {
	for _, ext := range []string{formatYAML, "toml", "json"} {
		t.Run(ext, func(t *testing.T) {
			dir := configHome(t)
			stubPrompts(t, formatAnswers{
				isEntity: false,
				noScan:   true,
				sections: answerSections(map[string]string{
					fieldGitHub: sponsors,
					fieldOrcid:  "0000-0001-2345-6789",
				}),
			})

			assertNoErr(t, Run(Options{Format: ext}))

			path := filepath.Join(dir, "projectfile", "cli."+ext)
			assertExists(t, path, "the wizard must write the config it collected")
			body := readFile(t, path)
			if !strings.Contains(body, "0000-0001-2345-6789") {
				t.Errorf("written config is missing the collected ORCID:\n%s", body)
			}
			if !strings.Contains(body, "bob") {
				t.Errorf("written config is missing the collected funding list:\n%s", body)
			}
		})
	}
}

// TestRunIsolatesFromTheDevelopersOwnConfig pins that a test run reads and writes only the throwaway config
// home, never the config the developer actually uses.
func TestRunIsolatesFromTheDevelopersOwnConfig(t *testing.T) {
	dir := configHome(t)
	realHome := os.Getenv("XDG_CONFIG_HOME")
	t.Cleanup(func() { t.Setenv("XDG_CONFIG_HOME", realHome) })
	if strings.HasPrefix(dir, realHome) {
		t.Skipf("XDG_CONFIG_HOME is not redirectable here (%s)", realHome)
	}
	stubPrompts(t, formatAnswers{isEntity: false})
	assertNoErr(t, Run(Options{Format: formatYAML}))
	if _, err := os.Stat(filepath.Join(realHome, "projectfile", "cli.yaml")); err == nil {
		t.Skip("the real config path is not writable from here")
	}
}

// TestRunWritesNothingWhenTheFormatIsRefused pins that a bad --format is reported as a usage error before any
// prompt or write, so a mistyped flag cannot leave a half-written config behind.
func TestRunWritesNothingWhenTheFormatIsRefused(t *testing.T) {
	dir := configHome(t)
	stubPrompts(t, formatAnswers{formatErr: errCancelled})

	err := Run(Options{Format: "ini"})
	if err == nil {
		t.Fatal("a bad --format was accepted")
	}
	if !strings.Contains(err.Error(), "--format") {
		t.Errorf("error %q must name the flag", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "projectfile")); !os.IsNotExist(statErr) {
		t.Errorf("a refused --format must create no config directory, got %v", statErr)
	}
}

// TestRunWritesNothingWhenAWizardStepIsCancelled pins the abort contract at its highest level: Esc mid-wizard
// is a clean no-op, not a partial write.
func TestRunWritesNothingWhenAWizardStepIsCancelled(t *testing.T) {
	dir := configHome(t)
	stubPrompts(t, formatAnswers{
		isEntity: false,
		sections: func(string, []field) error { return errCancelled },
	})

	assertNoErr(t, Run(Options{Format: formatYAML}))

	if _, err := os.Stat(filepath.Join(dir, "projectfile")); !os.IsNotExist(err) {
		t.Errorf("a cancelled wizard must write nothing, got %v", err)
	}
}

// TestRunWritesNothingWhenTheFormatPickerIsCancelled pins the same for Esc at the very first prompt.
func TestRunWritesNothingWhenTheFormatPickerIsCancelled(t *testing.T) {
	dir := configHome(t)
	stubPrompts(t, formatAnswers{formatErr: errCancelledFromSelector()})

	assertNoErr(t, Run(Options{}))

	if _, err := os.Stat(filepath.Join(dir, "projectfile")); !os.IsNotExist(err) {
		t.Errorf("a cancelled picker must write nothing, got %v", err)
	}
}

// TestRunKeepsTheExistingConfigWhenTheWizardIsCancelled pins the re-run case: a user who opens setup on a
// config they already have and then presses Esc must find their file untouched, not truncated.
func TestRunKeepsTheExistingConfigWhenTheWizardIsCancelled(t *testing.T) {
	dir := configHome(t)
	existing := writeExistingConfig(t, dir, formatYAML)
	before := readFile(t, existing)
	stubPrompts(t, formatAnswers{
		isEntity: false,
		sections: func(string, []field) error { return errCancelled },
	})

	assertNoErr(t, Run(Options{}))

	if got := readFile(t, existing); got != before {
		t.Errorf("a cancelled re-run rewrote the config:\nbefore:\n%s\nafter:\n%s", before, got)
	}
}

// TestRunMigratingEncodingOffersTheCleanup pins the migration path: writing a new encoding over an existing one
// is exactly when the previous file lingers and must be offered for deletion.
func TestRunMigratingEncodingOffersTheCleanup(t *testing.T) {
	dir := configHome(t)
	oldPath := writeExistingConfig(t, dir, "toml")
	var promptTitle string
	stubPrompts(t, formatAnswers{
		isEntity: false,
		sections: func(string, []field) error { return nil },
		yesNo: func(title string, defaultYes bool) (bool, error) {
			promptTitle = title
			if defaultYes {
				t.Error("the migration cleanup must default to no")
			}
			return true, nil
		},
	})

	assertNoErr(t, Run(Options{Format: formatYAML}))

	if !strings.Contains(promptTitle, oldPath) {
		t.Errorf("cleanup prompt %q must name the file it would delete (%s)", promptTitle, oldPath)
	}
	assertMissing(t, oldPath, "confirming the cleanup must remove the previous encoding")
	assertExists(t, filepath.Join(dir, "projectfile", "cli.yaml"), "the new encoding must survive")
}

// TestRunReusingTheSameEncodingOffersNoCleanup pins the inverse: re-running setup over the SAME encoding has no
// leftover to clean up, so asking would be a pointless prompt.
func TestRunReusingTheSameEncodingOffersNoCleanup(t *testing.T) {
	dir := configHome(t)
	oldPath := writeExistingConfig(t, dir, formatYAML)
	asked := false
	stubPrompts(t, formatAnswers{
		isEntity: false,
		sections: func(string, []field) error { return nil },
		// The no_scan toggle also asks, so the cleanup is identified by its title.
		yesNo: func(title string, _ bool) (bool, error) {
			if strings.Contains(title, "Delete the previous config") {
				asked = true
			}
			return false, nil
		},
	})

	assertNoErr(t, Run(Options{}))

	if asked {
		t.Error("re-running over the same encoding must not offer a cleanup")
	}
	assertExists(t, oldPath, "the config must survive an in-place re-run")
}

// TestRunReportsAWriteFailure pins that an unwritable config home surfaces as an error naming the write,
// rather than a silent success the user would only discover on the next run.
func TestRunReportsAWriteFailure(t *testing.T) {
	dir := configHome(t)
	// A regular file where the config directory must go: MkdirAll refuses it.
	blocker := filepath.Join(dir, "projectfile")
	if err := os.WriteFile(blocker, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("stage blocker: %v", err)
	}
	stubPrompts(t, formatAnswers{
		isEntity: false,
		sections: func(string, []field) error { return nil },
	})

	err := Run(Options{Format: formatYAML})
	if err == nil {
		t.Fatal("an unwritable config home was reported as a success")
	}
	if !strings.Contains(err.Error(), "write user config") {
		t.Errorf("error %q must name the failing step", err)
	}
}

// TestRunRecordsTheEntityChoiceInTheWrittenConfig pins that the discriminator answer reaches the file; a
// wizard that asked and then discarded it would flip the user back to person on every re-run.
func TestRunRecordsTheEntityChoiceInTheWrittenConfig(t *testing.T) {
	for _, isEntity := range []bool{false, true} {
		dir := configHome(t)
		stubPrompts(t, formatAnswers{
			isEntity: isEntity,
			sections: func(string, []field) error { return nil },
		})

		assertNoErr(t, Run(Options{Format: formatYAML}))

		path := filepath.Join(dir, "projectfile", "cli.yaml")
		userconfig.SetIgnored(false) // read the file the run just wrote, not the isolated empty cache
		written := userconfig.Load()
		if written.Identity.IsEntity != isEntity {
			t.Errorf("reloaded IsEntity = %v, want %v (written:\n%s)", written.Identity.IsEntity, isEntity, readFile(t, path))
		}
	}
}

// configHome points XDG_CONFIG_HOME at a throwaway directory for the rest of the test and hands back its path.
// userconfig caches the loaded document for the process lifetime, so the ignore toggle is flipped and the
// cache reset with it — otherwise one test's config would answer the next one's Load. Git is isolated too, or
// the developer's own identity lands in the fixture the test then asserts against.
func configHome(t *testing.T) string {
	t.Helper()
	isolateGit(t)
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	userconfig.SetIgnored(true)
	t.Cleanup(func() { userconfig.SetIgnored(false) })
	return dir
}

// writeExistingConfig plants a cli.<ext> in the throwaway config home so a test starts from the state a
// returning user is in.
func writeExistingConfig(t *testing.T, dir, ext string) string {
	t.Helper()
	base := filepath.Join(dir, "projectfile")
	if err := os.MkdirAll(base, 0o750); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	path := filepath.Join(base, "cli."+ext)
	if err := os.WriteFile(path, []byte("identity:\n  family-names: Ada\n"), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}
