// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"kiota.ch/projectfile/core/v2/pkg/genlog"
)

// envFixtureTOML carries an `env{}` map under an example extension and
// references an environment variable that --expand-env must resolve.
// dotted-key labels exercise the round-trip path for traefik-style labels.
const envFixtureTOML = `#:schema https://projectfile.org/schema/v1.json
spec_version = "1"
kind = "SoftwareSourceCode"

[identity]
namespace = "org.example"
name = "demo"

[com.example.env]
FOO = "1"
BAR = "two"
TARGET = "${TEST_VAR}"

[com.example.build.labels]
"traefik.enable" = "true"
"traefik.docker.network" = "edge"
`

// resetGetFlags zeroes the package-level cobra globals so a test that
// sets --format / --expand-env / --batch doesn't bleed into the next case.
func resetGetFlags(t *testing.T) {
	t.Helper()
	getDefault = ""
	getOrDefault = false
	getFormat = "raw"
	getBatch = false
	getExists = false
	getExpandEnv = false
	getPathFile = ""
	getNamedPaths = nil
	getLang = ""
	// get now detects --default via its Changed state, which is NOT self-clearing
	// across Execute() calls on the shared rootCmd — reset it or a --default case
	// would bleed presence into the next test.
	if f := getCmd.Flags().Lookup("default"); f != nil {
		f.Changed = false
	}
}

// runGetCmd drives the get command in isolation. Same Cobra-root pattern
// as runConvertCmd, so subcommand dispatch reaches our handler.
func runGetCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	resetGetFlags(t)
	genlog.SetOutput(&bytes.Buffer{})
	t.Cleanup(func() { genlog.SetOutput(os.Stderr) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs(append([]string{"get"}, args...))
	err := rootCmd.Execute()
	return buf.String(), err
}

func writeFixture(t *testing.T, dir, body string) string {
	t.Helper()
	p := filepath.Join(dir, "projectfile.toml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
	return p
}

// TestGetMapProjectRaw confirms the env{} grammar fans the map out as one
// KEY=VALUE line per pair in --format raw. Order is deterministic (sorted
// keys per the walker fallback) so the assertion can compare verbatim.
func TestGetMapProjectRaw(t *testing.T) {
	dir := t.TempDir()
	path := writeFixture(t, dir, envFixtureTOML)

	out, err := runGetCmd(t, "ext.com.example.env{}", "--path-file", path)
	if err != nil {
		t.Fatalf("get env{}: %v", err)
	}
	want := "BAR=two\nFOO=1\nTARGET=${TEST_VAR}\n"
	if out != want {
		t.Fatalf("raw env{} = %q, want %q", out, want)
	}
}

// TestGetMapProjectDottedLabels makes sure dotted labels (the traefik.*
// pattern) survive the {} fan-out without being navigated into. This is
// the regression guard for the buildah `.build.labels` consumer.
func TestGetMapProjectDottedLabels(t *testing.T) {
	dir := t.TempDir()
	path := writeFixture(t, dir, envFixtureTOML)

	out, err := runGetCmd(t, "ext.com.example.build.labels{}", "--path-file", path)
	if err != nil {
		t.Fatalf("get labels{}: %v", err)
	}
	if !strings.Contains(out, "traefik.enable=true") ||
		!strings.Contains(out, "traefik.docker.network=edge") {
		t.Fatalf("labels{} missing dotted entries: %q", out)
	}
}

// TestGetMapProjectSh confirms --format=sh emits one export per pair with
// shellKey normalisation applied (dots → underscores, uppercase). This is
// what a caller would `eval` to get the build-config block in scope.
func TestGetMapProjectSh(t *testing.T) {
	dir := t.TempDir()
	path := writeFixture(t, dir, envFixtureTOML)

	out, err := runGetCmd(t, "ext.com.example.build.labels{}", "--path-file", path, "--format", "sh")
	if err != nil {
		t.Fatalf("get labels{} sh: %v", err)
	}
	// shellKey upper-cases and replaces '.' with '_' — traefik.enable →
	// TRAEFIK_ENABLE. Values that contain no shell metacharacters come out
	// without quoting (see shellQuote / needsShellQuote).
	if !strings.Contains(out, "export TRAEFIK_ENABLE=true\n") {
		t.Fatalf("sh export missing TRAEFIK_ENABLE: %q", out)
	}
	if !strings.Contains(out, "export TRAEFIK_DOCKER_NETWORK=edge\n") {
		t.Fatalf("sh export missing TRAEFIK_DOCKER_NETWORK: %q", out)
	}
}

// TestGetMapProjectJSON confirms --format=json renders pair results back
// to a JSON object — round-trips to the same shape the projectfile had.
func TestGetMapProjectJSON(t *testing.T) {
	dir := t.TempDir()
	path := writeFixture(t, dir, envFixtureTOML)

	out, err := runGetCmd(t, "ext.com.example.env{}", "--path-file", path, "--format", "json")
	if err != nil {
		t.Fatalf("get env{} json: %v", err)
	}
	// The encoder json-renders a map[string]any; key order in maps is not
	// guaranteed, so check each entry individually instead of full equality.
	for _, want := range []string{`"BAR": "two"`, `"FOO": "1"`, `"TARGET": "${TEST_VAR}"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("json env{} missing %s: %q", want, out)
		}
	}
}

// TestGetFlatSubtree confirms --format=flat walks a whole subtree into one
// `<dotted.path>=<scalar>` line per leaf, paths VERBATIM (no shellKey
// upper-casing) and relative to the queried address, key-sorted. This is the
// single-call subtree dump the m6e DAG reader consumes in place of hundreds of
// per-leaf `get` spawns.
func TestGetFlatSubtree(t *testing.T) {
	dir := t.TempDir()
	path := writeFixture(t, dir, envFixtureTOML)

	out, err := runGetCmd(t, "ext.com.example.env", "--path-file", path, "--format", "flat")
	if err != nil {
		t.Fatalf("get env flat: %v", err)
	}
	want := "BAR=two\nFOO=1\nTARGET=${TEST_VAR}\n"
	if out != want {
		t.Fatalf("flat env = %q, want %q", out, want)
	}
}

// TestGetFlatNested checks that a deeper subtree keeps the full relative path
// per leaf so attribution survives — the property the reader relies on to map
// `tools.<t>.image` / `nodes.<n>.needs.<t>` back to their owner.
func TestGetFlatNested(t *testing.T) {
	dir := t.TempDir()
	path := writeFixture(t, dir, envFixtureTOML)

	out, err := runGetCmd(t, "ext.com.example", "--path-file", path, "--format", "flat")
	if err != nil {
		t.Fatalf("get subtree flat: %v", err)
	}
	for _, want := range []string{
		"build.labels.traefik.enable=true\n",
		"build.labels.traefik.docker.network=edge\n",
		"env.FOO=1\n",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("flat subtree missing %q in %q", want, out)
		}
	}
}

// listOfMapsFixtureTOML carries an array-of-tables (mounts:, the shape m6e's CI
// tools use) plus a scalar list, to pin the two flat-list behaviours apart.
const listOfMapsFixtureTOML = `#:schema https://projectfile.org/schema/v1.json
spec_version = "1"
kind = "SoftwareSourceCode"

[identity]
namespace = "org.example"
name = "demo"

[com.example]
tags = ["a", "b", "c"]

[[com.example.mounts]]
from = "grype-db"
to = "/app/.cache/grype"

[[com.example.mounts]]
from = "/var/run/docker.sock"
to = "/var/run/docker.sock"
`

// TestGetFlatListOfMaps pins the lossless flat lowering of a list-of-maps: each
// element recurses under an indexed `[i]` segment so every field keeps an
// unambiguous path (the property the m6e DAG reader relies on to drop its
// per-tool `mounts[].from/.to` spawns). A scalar list must still space-join.
func TestGetFlatListOfMaps(t *testing.T) {
	dir := t.TempDir()
	path := writeFixture(t, dir, listOfMapsFixtureTOML)

	out, err := runGetCmd(t, "ext.com.example", "--path-file", path, "--format", "flat")
	if err != nil {
		t.Fatalf("get subtree flat: %v", err)
	}
	for _, want := range []string{
		"mounts[0].from=grype-db\n",
		"mounts[0].to=/app/.cache/grype\n",
		"mounts[1].from=/var/run/docker.sock\n",
		"mounts[1].to=/var/run/docker.sock\n",
		"tags=a b c\n", // scalar list stays space-joined (no regression)
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("flat list-of-maps missing %q in %q", want, out)
		}
	}
}

// imageFixtureTOML carries a namespaced identity so the image.* synthetics
// have both halves to split. The ci.image override is exercised separately with
// an inline document (it changes the whole basename, tag included).
const imageFixtureTOML = `#:schema https://projectfile.org/schema/v1.json
spec_version = "1"
kind = "SoftwareSourceCode"

[identity]
namespace = "org.example.b19"
name = "ubuntu"
`

// TestGetImageSynthetics pins the three derived image addresses — the make-plane
// home of the ${image.basename/namespace/name} references the m6e reader
// interpolates. The split MUST match ci-resolver's basenameParts byte-for-byte
// (last `/` is the namespace boundary), so a build-arg identity value can never
// disagree across the make and cloud lowerings.
func TestGetImageSynthetics(t *testing.T) {
	dir := t.TempDir()
	path := writeFixture(t, dir, imageFixtureTOML)

	cases := map[string]string{
		addrImageBasename:  "b19/ubuntu", // last-label(namespace)/name
		addrImageNamespace: "b19",        // before the last /
		addrImageName:      "ubuntu",     // after the last /, no :tag
	}
	for addr, want := range cases {
		out, err := runGetCmd(t, addr, "--path-file", path)
		if err != nil {
			t.Fatalf("get %s: %v", addr, err)
		}
		if got := strings.TrimSpace(out); got != want {
			t.Fatalf("get %s = %q, want %q", addr, got, want)
		}
	}
}

// TestGetImageSyntheticsBareName covers the namespace-less identity: the
// namespace half is EMPTY-but-PRESENT (an empty identity arg is valid), so the
// command exits 0 with a blank value rather than treating it as missing (which
// would let --or-default fire). Mirrors the ci-resolver behaviour.
func TestGetImageSyntheticsBareName(t *testing.T) {
	dir := t.TempDir()
	path := writeFixture(t, dir, strings.Replace(imageFixtureTOML,
		"namespace = \"org.example.b19\"\nname = \"ubuntu\"", "name = \"loner\"", 1))

	out, err := runGetCmd(t, addrImageNamespace, "--path-file", path)
	if err != nil {
		t.Fatalf("get image.namespace (bare): %v", err)
	}
	if got := strings.TrimSpace(out); got != "" {
		t.Fatalf("bare-name image.namespace = %q, want empty", got)
	}
	nameOut, err := runGetCmd(t, addrImageName, "--path-file", path)
	if err != nil {
		t.Fatalf("get image.name (bare): %v", err)
	}
	if got := strings.TrimSpace(nameOut); got != "loner" {
		t.Fatalf("bare-name image.name = %q, want loner", got)
	}
}

// TestGetImageSyntheticsCIOverride confirms org.projectfile.ci.image wins
// verbatim for the basename (tag and all), and the namespace/name split still
// strips the :tag from the name half only — the property a tagged override
// relies on so M6E_PROJECT never carries a version tag.
func TestGetImageSyntheticsCIOverride(t *testing.T) {
	dir := t.TempDir()
	path := writeFixture(t, dir, imageFixtureTOML+`
[org.projectfile.ci]
image = "custom/thing:v2"
`)

	cases := map[string]string{
		addrImageBasename:  "custom/thing:v2", // override wins verbatim, tag kept
		addrImageNamespace: "custom",
		addrImageName:      "thing", // :tag stripped from the name half
	}
	for addr, want := range cases {
		out, err := runGetCmd(t, addr, "--path-file", path)
		if err != nil {
			t.Fatalf("get %s (override): %v", addr, err)
		}
		if got := strings.TrimSpace(out); got != want {
			t.Fatalf("override get %s = %q, want %q", addr, got, want)
		}
	}
}

// localizedFixtureTOML has identity.summary with a single language (en) and
// identity.title with two languages (en + es) to exercise both auto-unwrap
// and --lang selection.
const localizedFixtureTOML = `#:schema https://projectfile.org/schema/v1.json
spec_version = "1"
kind = "SoftwareSourceCode"

[identity]
namespace = "org.example"
name = "demo"

[identity.summary]
en = "English summary"

[identity.title]
en = "English title"
es = "Título en español"
`

// TestGetLocalizedSingleLangRaw checks that a single-language map is
// unwrapped automatically in --format raw without requiring --lang.
func TestGetLocalizedSingleLangRaw(t *testing.T) {
	dir := t.TempDir()
	path := writeFixture(t, dir, localizedFixtureTOML)

	out, err := runGetCmd(t, "identity.summary", "--path-file", path)
	if err != nil {
		t.Fatalf("get identity.summary: %v", err)
	}
	if strings.TrimSpace(out) != "English summary" {
		t.Fatalf("single-lang raw = %q, want %q", out, "English summary")
	}
}

// TestGetLocalizedSingleLangSh checks that identity{} map projection unwraps
// single-language values in --format sh (no en= prefix).
func TestGetLocalizedSingleLangSh(t *testing.T) {
	dir := t.TempDir()
	path := writeFixture(t, dir, localizedFixtureTOML)

	out, err := runGetCmd(t, "identity{}", "--path-file", path, "--format", "sh")
	if err != nil {
		t.Fatalf("get identity{} sh: %v", err)
	}
	if !strings.Contains(out, "export SUMMARY='English summary'\n") {
		t.Fatalf("sh summary missing unwrapped value: %q", out)
	}
}

// TestGetLocalizedLangFlag checks that --lang selects the right value from a
// multi-language map and leaves single-string fields untouched.
func TestGetLocalizedLangFlag(t *testing.T) {
	dir := t.TempDir()
	path := writeFixture(t, dir, localizedFixtureTOML)

	out, err := runGetCmd(t, "identity.title", "--path-file", path, "--lang", "es")
	if err != nil {
		t.Fatalf("get identity.title --lang es: %v", err)
	}
	if strings.TrimSpace(out) != "Título en español" {
		t.Fatalf("--lang es title = %q, want Spanish value", out)
	}
}

// TestGetLocalizedLangFlagJSON checks that --lang resolves localized maps in
// --format json output as well.
func TestGetLocalizedLangFlagJSON(t *testing.T) {
	dir := t.TempDir()
	path := writeFixture(t, dir, localizedFixtureTOML)

	out, err := runGetCmd(t, "identity{}", "--path-file", path, "--format", "json", "--lang", "en")
	if err != nil {
		t.Fatalf("get identity{} json --lang en: %v", err)
	}
	if !strings.Contains(out, `"summary": "English summary"`) {
		t.Fatalf("json summary missing unwrapped value: %q", out)
	}
	if !strings.Contains(out, `"title": "English title"`) {
		t.Fatalf("json title missing --lang en value: %q", out)
	}
}

// TestGetExpandEnv runs --expand-env against the fixture's TARGET = ${TEST_VAR}
// entry and checks the value resolves before parse. Without --expand-env the
// literal `${TEST_VAR}` stays in the value, so the two assertions in one test
// document both modes.
func TestGetExpandEnv(t *testing.T) {
	dir := t.TempDir()
	path := writeFixture(t, dir, envFixtureTOML)

	t.Setenv("TEST_VAR", "expanded-value")

	// Without --expand-env: literal placeholder survives.
	noExpand, err := runGetCmd(t, "ext.com.example.env.TARGET", "--path-file", path)
	if err != nil {
		t.Fatalf("get TARGET (no expand): %v", err)
	}
	if strings.TrimSpace(noExpand) != "${TEST_VAR}" {
		t.Fatalf("no-expand TARGET = %q, want literal ${TEST_VAR}", noExpand)
	}

	// With --expand-env: value resolves to the env-var contents.
	expanded, err := runGetCmd(t, "ext.com.example.env.TARGET", "--path-file", path, "--expand-env")
	if err != nil {
		t.Fatalf("get TARGET (expand): %v", err)
	}
	if strings.TrimSpace(expanded) != "expanded-value" {
		t.Fatalf("expand TARGET = %q, want expanded-value", expanded)
	}
}

// TestGetDefaultEmptyString pins the discriminator the --default flag exists to
// provide: a VALID projectfile missing the field, queried with an empty
// --default, yields the empty string and exits 0 — NOT the not-found soft
// failure. Presence is detected by the flag being set (Changed), so an empty
// override still counts. This is the common shell idiom `X=$(pf-cli get k
// --default=)`, which the old `getDefault != ""` sentinel silently broke.
func TestGetDefaultEmptyString(t *testing.T) {
	dir := t.TempDir()
	path := writeFixture(t, dir, envFixtureTOML)

	out, err := runGetCmd(t, "identity.summary", "--path-file", path, "--default", "")
	if err != nil {
		t.Fatalf("get missing --default='': %v", err)
	}
	if strings.TrimSpace(out) != "" {
		t.Fatalf("empty-default missing = %q, want empty", out)
	}
}

// TestGetDefaultValueUsed confirms a non-empty --default is emitted verbatim for
// an absent field, exit 0.
func TestGetDefaultValueUsed(t *testing.T) {
	dir := t.TempDir()
	path := writeFixture(t, dir, envFixtureTOML)

	out, err := runGetCmd(t, "identity.summary", "--path-file", path, "--default", "fallback")
	if err != nil {
		t.Fatalf("get missing --default=fallback: %v", err)
	}
	if strings.TrimSpace(out) != "fallback" {
		t.Fatalf("default value = %q, want fallback", out)
	}
}

// TestGetDefaultIgnoredWhenPresent guards that --default never shadows a real
// value: a present field returns its own value even with --default set.
func TestGetDefaultIgnoredWhenPresent(t *testing.T) {
	dir := t.TempDir()
	path := writeFixture(t, dir, envFixtureTOML)

	out, err := runGetCmd(t, "identity.name", "--path-file", path, "--default", "fallback")
	if err != nil {
		t.Fatalf("get present --default: %v", err)
	}
	if strings.TrimSpace(out) != "demo" {
		t.Fatalf("present-with-default = %q, want demo", out)
	}
}

// TestGetDefaultBrokenStillErrors is the safety guarantee: --default rescues an
// ABSENT field, never a BROKEN document. An unparseable projectfile must still
// exit non-zero even with --default set, so a non-zero exit unambiguously means
// "broken", not "missing" — the whole reason the user reaches for --default.
func TestGetDefaultBrokenStillErrors(t *testing.T) {
	dir := t.TempDir()
	// Unterminated table header: fails the TOML parse in loadDocument, so the
	// path lookup (and its default) is never reached.
	path := writeFixture(t, dir, "spec_version = \"1\"\n[identity\nname = \"x\"\n")

	if _, err := runGetCmd(t, "identity.name", "--path-file", path, "--default", "fallback"); err == nil {
		t.Fatalf("broken projectfile with --default should error, got nil")
	}
}

// TestGetUsageErrorType pins the type Execute() keys the exit-2 path on: a usage
// mistake (unknown --format) must surface as *usageError, while a runtime failure
// (broken document) must NOT — so the two map to distinct exit codes (2 vs 1).
func TestGetUsageErrorType(t *testing.T) {
	dir := t.TempDir()
	good := writeFixture(t, dir, envFixtureTOML)
	broken := filepath.Join(dir, "broken.toml")
	if err := os.WriteFile(broken, []byte("spec_version = \"1\"\n[identity\n"), 0o644); err != nil {
		t.Fatalf("write broken: %v", err)
	}

	_, uErr := runGetCmd(t, "identity.name", "--path-file", good, "--format", "bogus")
	var ue *usageError
	if !errors.As(uErr, &ue) {
		t.Fatalf("unknown --format should be a *usageError (exit 2), got %T: %v", uErr, uErr)
	}

	_, rErr := runGetCmd(t, "identity.name", "--path-file", broken)
	if rErr == nil || errors.As(rErr, &ue) {
		t.Fatalf("broken doc should be a non-usage runtime error (exit 1), got %v", rErr)
	}
}
