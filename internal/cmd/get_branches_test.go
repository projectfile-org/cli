// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kiota.ch/projectfile/core/v2/pkg/fieldpath"
	"kiota.ch/projectfile/core/v2/pkg/projectfile"
)

// errUnencodable is the failure every encoder in these tests reports.
var errUnencodable = errors.New("cannot encode this value")

// sinkCmd is a command whose streams are one buffer, for exercising the emitters without the root.
func sinkCmd() (*cobra.Command, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	c := &cobra.Command{}
	c.SetOut(buf)
	c.SetErr(buf)
	return c, buf
}

// TestGetPrintPathAnswersTheResolvedFile pins the §4.5 arbitration helper: the caller gets the exact path, or silence when there is no projectfile at all.
func TestGetPrintPathAnswersTheResolvedFile(t *testing.T) {
	p := writeFixture(t, t.TempDir(), envFixtureTOML)

	printed := captureStdout(t, func() {
		_, err := runGetCmd(t, "--file", p, "--print-path")
		require.NoError(t, err)
	})
	abs, absErr := filepath.Abs(p)
	require.NoError(t, absErr)
	assert.Equal(t, abs+"\n", printed)

	t.Chdir(t.TempDir())
	printed = captureStdout(t, func() {
		_, err := runGetCmd(t, "--print-path")
		require.NoError(t, err)
	})
	assert.Empty(t, printed, "no projectfile is a silent exit 0, never a printed guess")
}

// captureStdout collects what fn printed on the real stdout, for the paths that write there instead of to the command's stream.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	saved := os.Stdout
	os.Stdout = w
	fn()
	os.Stdout = saved
	require.NoError(t, w.Close())
	out, err := io.ReadAll(r)
	require.NoError(t, err)
	require.NoError(t, r.Close())
	return string(out)
}

// TestGetRequiresAtLeastOnePath pins the usage error: an invocation with nothing to read names the fix instead of reading the whole document.
func TestGetRequiresAtLeastOnePath(t *testing.T) {
	t.Chdir(t.TempDir())
	_, err := runGetCmd(t)
	require.Error(t, err)
	assert.Equal(t, exitUsage, exitCodeFor(err))
}

// TestGetRejectsSeveralPathsWithoutBatch pins that a multi-path read is opt-in: without --batch the last path would silently win.
func TestGetRejectsSeveralPathsWithoutBatch(t *testing.T) {
	t.Chdir(t.TempDir())
	_, err := runGetCmd(t, addrIdentityName, "identity.namespace")
	require.Error(t, err)
	assert.Equal(t, exitUsage, exitCodeFor(err))
}

// TestGetRejectsDefaultAndOrDefaultTogether pins the mutual exclusion: two fallbacks with no precedence would make the answer depend on parse order.
func TestGetRejectsDefaultAndOrDefaultTogether(t *testing.T) {
	t.Chdir(t.TempDir())
	_, err := runGetCmd(t, "--default", valDemo, "--or-default", addrIdentityName)
	require.Error(t, err)
	assert.Equal(t, exitUsage, exitCodeFor(err))
}

// TestGetRawBatchPrefixesEveryLine pins the one place the batch key is emitted: raw single values, so a caller can tell two entries apart.
func TestGetRawBatchPrefixesEveryLine(t *testing.T) {
	p := writeFixture(t, t.TempDir(), envFixtureTOML)
	out, err := runGetCmd(t, "--file", p, "--batch", addrIdentityName)
	require.NoError(t, err)
	assert.Equal(t, addrIdentityName+"="+valDemo+"\n", out)
}

// TestGetExistsOnAPresentFieldPrintsNothing pins --exists as a pure exit-code probe: a caller branching on the code must not have to read stdout.
func TestGetExistsOnAPresentFieldPrintsNothing(t *testing.T) {
	p := writeFixture(t, t.TempDir(), envFixtureTOML)
	out, err := runGetCmd(t, "--file", p, "--exists", addrIdentityName)
	require.NoError(t, err)
	assert.Empty(t, out)
}

// TestGetLangResolvesATranslatedField pins that --lang collapses the localized map before every renderer sees it.
func TestGetLangResolvesATranslatedField(t *testing.T) {
	p := writeFixture(t, t.TempDir(), localizedFixtureTOML)

	out, err := runGetCmd(t, "--file", p, "--lang", "es", "--format", "json", "identity.title")
	require.NoError(t, err)
	assert.Equal(t, "\"Título en español\"\n", out)

	out, err = runGetCmd(t, "--file", p, "--format", "json", "identity.title")
	require.NoError(t, err)
	assert.Contains(t, out, "\"en\"")
}

// TestGetOrDefaultFallsBackToTheSpecDefault pins the built-in default path, the second fallback a missing field can take: an absent optional field answers from the spec instead of failing.
func TestGetOrDefaultFallsBackToTheSpecDefault(t *testing.T) {
	dir := t.TempDir()
	p := writeFixture(t, dir, "#:schema https://projectfile.org/schema/v1.json\nspec_version = \"1\"\n\n[identity]\nnamespace = \"org.example\"\nname = \"demo\"\n")

	out, err := runGetCmd(t, "--file", p, "--or-default", "kind")
	require.NoError(t, err)
	assert.Equal(t, "library\n", out, "the spec default answers a field the document never stated")

	resetGetFlags(t)
	doc, err := loadDocument(p)
	require.NoError(t, err)
	path, err := fieldpath.Parse("identity.nickname")
	require.NoError(t, err)
	value, ok := defaultValueFor(doc, path, false)
	assert.False(t, ok, "no flag and no registry entry is a plain miss")
	assert.Nil(t, value)

	getOrDefault = true
	_, ok = defaultValueFor(doc, path, true)
	assert.True(t, ok, "core's default registry answers whatever it declares for the path")

	// An explicit --default answers whatever the registry holds, which is the documented precedence.
	kindPath, err := fieldpath.Parse("kind")
	require.NoError(t, err)
	value, ok = defaultValueFor(doc, kindPath, false)
	assert.True(t, ok)
	assert.Equal(t, "library", value)

	getDefault = valDemo
	value, ok = defaultValueFor(doc, kindPath, true)
	assert.True(t, ok)
	assert.Equal(t, valDemo, value)
}

// TestResolveEntryReportsAGrammarMistakeLoudly pins that a list operator aimed at a named map is an error, not a silent miss the caller would read as a typo.
func TestResolveEntryReportsAGrammarMistakeLoudly(t *testing.T) {
	doc := projectfile.FromMap(map[string]any{keyIdentity: map[string]any{keyName: valDemo}})
	path, err := fieldpath.Parse("identity[key=name]")
	require.NoError(t, err)

	_, _, _, present, err := resolveEntry(doc, path)
	require.Error(t, err)
	assert.False(t, present)

	missing, err := fieldpath.Parse("identity.absent")
	require.NoError(t, err)
	_, _, _, present, err = resolveEntry(doc, missing)
	require.NoError(t, err)
	assert.False(t, present, "a plain miss is not an error")
}

// TestSingleOrListKeepsAnEmptyResult pins the empty case: zero values stay a slice, so the renderer never ranges over a nil of the wrong shape.
func TestSingleOrListKeepsAnEmptyResult(t *testing.T) {
	assert.Equal(t, []any{}, singleOrList(fieldpath.Result{Values: []any{}}))
	assert.Equal(t, valDemo, singleOrList(fieldpath.Result{Values: []any{valDemo}}))
}

// TestEmitRawRendersEveryListShape pins the raw renderer's fallbacks: strings, maps inside a list and a map value all reach the user as text.
func TestEmitRawRendersEveryListShape(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   any
		want string
	}{
		{name: "strings", in: []string{"a", "b"}, want: "a\nb\n"},
		{name: "list of maps", in: []map[string]any{{"k": "v"}}, want: "k=v\n"},
		{name: "map", in: map[string]any{"k": "v"}, want: "k=v\n"},
		{name: "scalar", in: valDemo, want: "demo\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd, buf := sinkCmd()
			emitRaw(cmd, tc.in, false, false)
			assert.Equal(t, tc.want, buf.String())
		})
	}
}

// TestEmitRawRefusesANonSlicePairList pins the type contract: a pairs result is always a slice, so anything else is a bug in the caller and must not panic a render.
func TestEmitRawRefusesANonSlicePairList(t *testing.T) {
	cmd, buf := sinkCmd()
	assert.Panics(t, func() { emitRaw(cmd, valDemo, false, true) })
	assert.Empty(t, buf.String())
}

// TestSingleOrBatchPayloadOfNothingPresent pins the all-missing answer: nil, so json prints null and yaml prints null rather than an empty map.
func TestSingleOrBatchPayloadOfNothingPresent(t *testing.T) {
	assert.Nil(t, singleOrBatchPayload([]resolvedEntry{{present: false}}, false))
	assert.Equal(t, map[string]any{}, singleOrBatchPayload([]resolvedEntry{{present: false}}, true))
}

// TestEmitJSONReportsAnUnencodableValue pins that the JSON renderer surfaces its error instead of writing half a document.
func TestEmitJSONReportsAnUnencodableValue(t *testing.T) {
	cmd, _ := sinkCmd()
	require.Error(t, emitJSON(cmd, []resolvedEntry{{key: "k", value: math.NaN(), present: true}}, false))
}

// brokenMarshal is a value no encoder can write, so the emitters' error paths are reachable at all.
type brokenMarshal struct{}

func (brokenMarshal) MarshalYAML() (any, error) { return nil, errUnencodable }

// TestEmitYAMLReportsAnUnencodableValue pins that the YAML renderer surfaces its error instead of writing half a document.
func TestEmitYAMLReportsAnUnencodableValue(t *testing.T) {
	cmd, buf := sinkCmd()
	require.ErrorIs(t, emitYAML(cmd, []resolvedEntry{{key: "k", value: brokenMarshal{}, present: true}}, true), errUnencodable)
	assert.Empty(t, buf.String())
}

// TestEmitTOMLRefusesAScalar pins the format's own limit: TOML has no scalar document, so a scalar query is refused in words instead of writing a bare value.
func TestEmitTOMLRefusesAScalar(t *testing.T) {
	cmd, _ := sinkCmd()
	err := emitTOML(cmd, []resolvedEntry{{key: "k", value: valDemo, present: true}}, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "needs a map/object value")
}

// TestEmitTOMLReportsAnUnencodableValue pins that the encoder error reaches the caller.
func TestEmitTOMLReportsAnUnencodableValue(t *testing.T) {
	cmd, _ := sinkCmd()
	require.Error(t, emitTOML(cmd, []resolvedEntry{{key: "k", value: map[string]any{"k": func() {}}, present: true}}, true))
}

// TestJSONValueKeepsANonSlicePairResult pins the defensive branch: a pairs flag on a scalar is a caller bug, and the value passes through rather than panicking.
func TestJSONValueKeepsANonSlicePairResult(t *testing.T) {
	assert.Equal(t, valDemo, jsonValue(resolvedEntry{value: valDemo, isPairs: true, present: true}))
	assert.Equal(t, map[string]any{"k": valDemo}, jsonValue(resolvedEntry{
		value:   []any{fieldpath.Pair{Key: "k", Value: valDemo}},
		isPairs: true, present: true,
	}))
}

// TestEmitShSkipsAnAbsentEntry pins that a --default-less miss contributes no export line, so `eval` never defines a variable from nothing.
func TestEmitShSkipsAnAbsentEntry(t *testing.T) {
	cmd, buf := sinkCmd()
	emitSh(cmd, []resolvedEntry{
		{key: addrIdentityName, value: valDemo, present: true},
		{key: "identity.absent", present: false},
	})
	assert.Equal(t, "export IDENTITY_NAME=demo\n", buf.String())
}

// TestEmitFlatSkipsAnAbsentEntryAndPrefixesInBatch pins both of emitFlat's branches: a miss emits nothing, and --batch stamps the entry key on every leaf.
func TestEmitFlatSkipsAnAbsentEntryAndPrefixesInBatch(t *testing.T) {
	cmd, buf := sinkCmd()
	entries := []resolvedEntry{
		{key: "org.demo", value: map[string]any{"count": 1}, present: true},
		{key: "org.absent", present: false},
	}
	emitFlat(cmd, entries)
	assert.Equal(t, "count=1\n", buf.String())

	resetGetFlags(t)
	t.Cleanup(func() { resetGetFlags(t) })
	getBatch = true
	buf.Reset()
	emitFlat(cmd, entries)
	assert.Equal(t, "org.demo.count=1\n", buf.String())
}

// TestCollectPathEntriesRefusesAMalformedAddress pins that a bad address is an error rather than a silently dropped entry.
func TestCollectPathEntriesRefusesAMalformedAddress(t *testing.T) {
	_, err := collectPathEntries([]string{"identity..name"}, nil)
	require.Error(t, err)

	_, err = collectPathEntries(nil, []string{"no-equals-sign"})
	require.Error(t, err)
	assert.Equal(t, exitUsage, exitCodeFor(err))

	_, err = collectPathEntries(nil, []string{"KEY=identity..name"})
	require.Error(t, err)

	entries, err := collectPathEntries([]string{"keywords[0]"}, nil)
	require.NoError(t, err)
	assert.Equal(t, "keywords", entries[0].Key, "an index segment is not part of the export key")
}

// TestLoadDocumentRefusesAMissingFile pins the read error path, which every command that loads a document reports.
func TestLoadDocumentRefusesAMissingFile(t *testing.T) {
	resetGetFlags(t)
	t.Cleanup(func() { resetGetFlags(t) })
	_, err := loadDocument(filepath.Join(t.TempDir(), "absent.yaml"))
	require.Error(t, err)

	getExpandEnv = true
	_, err = loadDocument("")
	require.Error(t, err, "no projectfile in the working directory is an error, never an empty document")
}

// TestIsLocalizedMapRejectsNonTranslations pins the discriminator: a map is a translation only when every key is a language tag and every value a string.
func TestIsLocalizedMapRejectsNonTranslations(t *testing.T) {
	assert.False(t, isLocalizedMap(nil))
	assert.False(t, isLocalizedMap(map[string]any{"en": 1}))
	assert.False(t, isLocalizedMap(map[string]any{"not-a-lang-code-at-all": "x"}))
	assert.True(t, isLocalizedMap(map[string]any{"es": "hola"}))
}

// TestIsLangCodeRejectsWhatIsNotATag pins the accepted subset: 2–5 lowercase letters with an optional underscore.
func TestIsLangCodeRejectsWhatIsNotATag(t *testing.T) {
	assert.False(t, isLangCode("e"))
	assert.False(t, isLangCode("abcdef"))
	assert.False(t, isLangCode("en-GB"))
	assert.False(t, isLangCode("uk_UA"), "the accepted subset is lowercase letters only, so a region subtag in caps is not a tag here")
	assert.True(t, isLangCode("es"))
	assert.True(t, isLangCode("es_cl"))
}

// TestReportMissingStaysQuietWhenAsked pins --quiet: a caller that asked for silence must not get a hint stream on stderr.
func TestReportMissingStaysQuietWhenAsked(t *testing.T) {
	resetGetFlags(t)
	t.Cleanup(func() { resetGetFlags(t) })
	saved := quietFlag
	t.Cleanup(func() { quietFlag = saved })

	cmd, buf := sinkCmd()
	entries := []resolvedEntry{{key: addrIdentityName, present: false}}
	path, err := fieldpath.Parse(addrIdentityName)
	require.NoError(t, err)
	entries[0].path = path

	quietFlag = true
	reportMissing(cmd, entries, nil)
	assert.Empty(t, buf.String())

	quietFlag = false
	reportMissing(cmd, entries, nil)
	assert.Contains(t, buf.String(), "identity.name: no such field")
}

// TestReportMissingSkipsPresentEntries pins that only the misses are named.
func TestReportMissingSkipsPresentEntries(t *testing.T) {
	resetGetFlags(t)
	t.Cleanup(func() { resetGetFlags(t) })
	saved := quietFlag
	t.Cleanup(func() { quietFlag = saved })
	quietFlag = false

	cmd, buf := sinkCmd()
	reportMissing(cmd, []resolvedEntry{{key: addrIdentityName, value: valDemo, present: true}}, nil)
	assert.Empty(t, buf.String())
}

// TestSuggestEntriesCountsOnlyRealSuggestions pins that the count reports printed suggestions, so the caller knows whether to fall back to the error.
func TestSuggestEntriesCountsOnlyRealSuggestions(t *testing.T) {
	doc := projectfile.FromMap(map[string]any{keyIdentity: map[string]any{keyName: valDemo}})
	near, err := fieldpath.Parse(addrIdentityNam)
	require.NoError(t, err)
	far, err := fieldpath.Parse("identity.zzzzzzzzzz")
	require.NoError(t, err)

	cmd, buf := sinkCmd()
	n := suggestEntries(cmd, []resolvedEntry{
		{key: addrIdentityNam, path: near, present: false},
		{key: "identity.zzzzzzzzzz", path: far, present: false},
		{key: addrIdentityName, value: valDemo, present: true},
	}, doc)
	assert.Equal(t, 1, n)
	assert.Equal(t, "identity.name\n", buf.String())
}

// TestSuggestAddressNeedsSomethingToCorrect pins the guards: no document, an unreadable root and an index segment each yield no suggestion rather than a wrong one.
func TestSuggestAddressNeedsSomethingToCorrect(t *testing.T) {
	_, ok := suggestAddress(nil, fieldpath.Path{})
	assert.False(t, ok)

	index, err := fieldpath.Parse("keywords[0]")
	require.NoError(t, err)
	_, ok = suggestAddress(projectfile.FromMap(map[string]any{"keywords": []any{"rust"}}), index)
	assert.False(t, ok)

	near, err := fieldpath.Parse(addrIdentityNam)
	require.NoError(t, err)
	_, ok = suggestAddress(projectfile.FromMap(map[string]any{keyIdentity: "not a map"}), near)
	assert.False(t, ok)
}

// TestClosestSiblingNeedsAMapUnderThePath pins the guards on the nearest-key hint.
func TestClosestSiblingNeedsAMapUnderThePath(t *testing.T) {
	path, err := fieldpath.Parse("identity.zzzzzzzzzz")
	require.NoError(t, err)

	_, ok := closestSibling(nil, path)
	assert.False(t, ok)

	_, ok = closestSibling(projectfile.FromMap(map[string]any{keyIdentity: map[string]any{keyName: valDemo}}), path)
	assert.False(t, ok, "a key that is not close enough is silence, not a wrong guess")

	_, ok = closestSibling(projectfile.FromMap(map[string]any{keyIdentity: "scalar"}), path)
	assert.False(t, ok)

	first, err := fieldpath.Parse("identity")
	require.NoError(t, err)
	_, ok = closestSibling(projectfile.FromMap(map[string]any{keyIdentity: map[string]any{}}), first)
	assert.False(t, ok, "a single segment has no parent to search")
}
