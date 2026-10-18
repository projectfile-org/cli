// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kiota.ch/projectfile/core/v2/pkg/genlog"
	"kiota.ch/projectfile/core/v2/pkg/projectfile"
)

// cacheHome redirects the shared pf/ cache slot into a throwaway directory, so a test neither reads nor
// destroys the developer's real warm cache.
func cacheHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", dir)
	return dir
}

func resetCacheFlags(t *testing.T) {
	t.Helper()
	cacheWarmForce = false
	offlineFlag = false
}

// runCacheCmd drives one cache subcommand in isolation.
func runCacheCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	resetCacheFlags(t)
	genlog.SetOutput(&bytes.Buffer{})
	t.Cleanup(func() { genlog.SetOutput(os.Stderr) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs(append([]string{cmdCache}, args...))
	err := rootCmd.Execute()
	return buf.String(), err
}

// TestCacheStatusReportsTheResolvedLocation pins that status names the REAL path on disk, not just the env
// var, since a user debugging a warm cache needs the directory itself.
func TestCacheStatusReportsTheResolvedLocation(t *testing.T) {
	dir := cacheHome(t)
	out, err := runCacheCmd(t, "status")
	require.NoError(t, err)
	assert.Contains(t, out, filepath.Join(dir, "pf"))
}

// TestCacheStatusOnAnEmptyCache pins the zero-case wording, which differs from the populated one so a script
// can tell "nothing cached" from "could not read the cache".
func TestCacheStatusOnAnEmptyCache(t *testing.T) {
	cacheHome(t)
	out, err := runCacheCmd(t, "status")
	require.NoError(t, err)
	assert.Contains(t, out, "includes: 0 cached")
}

// TestCacheStatusCountsWarmIncludes pins that a warmed include shows up in the tally.
func TestCacheStatusCountsWarmIncludes(t *testing.T) {
	cacheHome(t)
	srv := includeServer(t, "keywords: [warm]")
	dir := t.TempDir()
	writeFile(t, dir, "projectfile.yaml", "identity:\n  name: demo\nincludes:\n  - "+srv.URL+"/base.yaml\n")

	require.NoError(t, warmIncludes(dir, false))

	out, err := runCacheCmd(t, "status")
	require.NoError(t, err)
	assert.Contains(t, out, "1 cached")
}

// TestCacheWarmDownloadsTheDeclaredIncludes pins the warm contract: after it runs, an offline read resolves
// from the cache alone, which is the whole reason the command exists.
func TestCacheWarmDownloadsTheDeclaredIncludes(t *testing.T) {
	cacheHome(t)
	srv := includeServer(t, "keywords: [warmed]")
	dir := t.TempDir()
	path := writeFile(t, dir, "projectfile.yaml", "identity:\n  name: demo\nincludes:\n  - "+srv.URL+"/base.yaml\n")

	_, err := runCacheCmd(t, "warm", dir)
	require.NoError(t, err)

	doc, err := projectfile.ReadFromPathWithOptions(path, projectfile.ReadOptions{Offline: true})
	require.NoError(t, err)
	assert.Equal(t, []string{"warmed"}, doc.Keywords)
}

// TestCacheWarmFollowsATransitiveChain pins that warming walks the whole include graph, so an offline read
// finds a fragment that was reached through another remote fragment.
func TestCacheWarmFollowsATransitiveChain(t *testing.T) {
	cacheHome(t)
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/mid.yaml":
			_, _ = w.Write([]byte("includes:\n  - " + srv.URL + "/leaf.yaml\n"))
		default:
			_, _ = w.Write([]byte("keywords: [deep]\n"))
		}
	}))
	t.Cleanup(srv.Close)
	dir := t.TempDir()
	path := writeFile(t, dir, "projectfile.yaml", "identity:\n  name: demo\nincludes:\n  - "+srv.URL+"/mid.yaml\n")

	_, err := runCacheCmd(t, "warm", dir)
	require.NoError(t, err)

	doc, err := projectfile.ReadFromPathWithOptions(path, projectfile.ReadOptions{Offline: true})
	require.NoError(t, err)
	assert.Equal(t, []string{"deep"}, doc.Keywords, "the transitively included fragment must be cached too")
}

// TestCacheWarmOnADirectoryWithNoProjectfileIsANoOp pins that a directory holding nothing to prefetch is a
// success: pf-cli's cache is includes-only, so there is genuinely nothing to do.
func TestCacheWarmOnADirectoryWithNoProjectfileIsANoOp(t *testing.T) {
	cacheHome(t)
	_, err := runCacheCmd(t, "warm", t.TempDir())
	require.NoError(t, err)
}

// TestCacheWarmOnAProjectfileWithNoIncludesIsANoOp pins the same for a document that declares no remote
// includes, which is the common case.
func TestCacheWarmOnAProjectfileWithNoIncludesIsANoOp(t *testing.T) {
	cacheHome(t)
	dir := t.TempDir()
	writeFile(t, dir, "projectfile.yaml", "identity:\n  name: demo\n")
	_, err := runCacheCmd(t, "warm", dir)
	require.NoError(t, err)
}

// TestCacheWarmForceRefetches pins that --force bypasses the cache TTL, which is what a user reaches for when
// an upstream include was corrected.
func TestCacheWarmForceRefetches(t *testing.T) {
	cacheHome(t)
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		_, _ = w.Write([]byte("keywords: [forced]\n"))
	}))
	t.Cleanup(srv.Close)
	dir := t.TempDir()
	writeFile(t, dir, "projectfile.yaml", "identity:\n  name: demo\nincludes:\n  - "+srv.URL+"/base.yaml\n")

	_, err := runCacheCmd(t, "warm", dir)
	require.NoError(t, err)
	before := hits
	_, err = runCacheCmd(t, "warm", dir, "--force")
	require.NoError(t, err)
	assert.Greater(t, hits, before, "--force must reach the origin again")
}

// TestCacheWarmSurvivesAnUnreachableInclude pins that one failing include does not abort the run: the rest
// still warm, and the failure is reported as a warning.
func TestCacheWarmSurvivesAnUnreachableInclude(t *testing.T) {
	cacheHome(t)
	dir := t.TempDir()
	writeFile(t, dir, "projectfile.yaml", "identity:\n  name: demo\nincludes:\n  - http://127.0.0.1:1/absent.yaml\n")
	_, err := runCacheCmd(t, "warm", dir)
	require.NoError(t, err, "one unreachable include must not fail the whole warm")
}

// TestCacheWarmAndRefreshRefuseOffline pins that --offline turns both warm commands into a clear refusal
// rather than a silent no-op the user would read as success.
func TestCacheWarmAndRefreshRefuseOffline(t *testing.T) {
	cacheHome(t)
	for _, sub := range []string{"warm", "refresh"} {
		t.Run(sub, func(t *testing.T) {
			_, err := runCacheCmd(t, sub, "--offline")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "offline")
		})
	}
}

// TestCacheRefreshIsTheForcedWarm pins the alias: refresh is warm --force, so it must reach the origin even
// when the cache is still fresh.
func TestCacheRefreshIsTheForcedWarm(t *testing.T) {
	cacheHome(t)
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		_, _ = w.Write([]byte("keywords: [refreshed]\n"))
	}))
	t.Cleanup(srv.Close)
	dir := t.TempDir()
	writeFile(t, dir, "projectfile.yaml", "identity:\n  name: demo\nincludes:\n  - "+srv.URL+"/base.yaml\n")

	_, err := runCacheCmd(t, "warm", dir)
	require.NoError(t, err)
	before := hits
	_, err = runCacheCmd(t, "refresh", dir)
	require.NoError(t, err)
	assert.Greater(t, hits, before, "refresh must refetch a fresh cache entry")
}

// TestCachePurgeClearsEverything pins that a bare purge empties the slot and reports the location it emptied.
func TestCachePurgeClearsEverything(t *testing.T) {
	dir := cacheHome(t)
	srv := includeServer(t, "keywords: [purgeable]")
	project := t.TempDir()
	writeFile(t, project, "projectfile.yaml", "identity:\n  name: demo\nincludes:\n  - "+srv.URL+"/base.yaml\n")
	_, err := runCacheCmd(t, "warm", project)
	require.NoError(t, err)

	out, err := runCacheCmd(t, "purge")
	require.NoError(t, err)
	assert.Contains(t, out, "purged 1")
	assert.Contains(t, out, filepath.Join(dir, "pf"))
}

// TestCachePurgeOneURLPinsTheRemovalToThatURL pins that the single-URL form leaves every other entry cached.
func TestCachePurgeOneURLPinsTheRemovalToThatURL(t *testing.T) {
	cacheHome(t)
	srv := includeServer(t, "keywords: [kept]")
	other := includeServer(t, "keywords: [other]")
	project := t.TempDir()
	writeFile(t, project, "projectfile.yaml", "identity:\n  name: demo\nincludes:\n  - "+srv.URL+"/base.yaml\n  - "+other.URL+"/other.yaml\n")
	_, err := runCacheCmd(t, "warm", project)
	require.NoError(t, err)

	out, err := runCacheCmd(t, "purge", srv.URL+"/base.yaml")
	require.NoError(t, err)
	assert.Contains(t, out, "purged "+srv.URL+"/base.yaml")

	status, err := runCacheCmd(t, "status")
	require.NoError(t, err)
	assert.Contains(t, status, "1 cached", "the sibling include must survive a single-URL purge")
}

// TestCachePurgeOfAnUncachedURLSaysSo pins the distinct wording, so a script can tell "removed" from
// "there was nothing there".
func TestCachePurgeOfAnUncachedURLSaysSo(t *testing.T) {
	cacheHome(t)
	out, err := runCacheCmd(t, "purge", "https://example.invalid/absent.yaml")
	require.NoError(t, err)
	assert.Contains(t, out, "no cached entry")
}

// TestCacheRootIsTheSharedPfSlot pins the location contract: the slot is <cache>/pf, shared with pf-bridge
// and pf-ci, so a user clearing one tool's cache clears all of them by design.
func TestCacheRootIsTheSharedPfSlot(t *testing.T) {
	dir := cacheHome(t)
	root, err := cacheRoot()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "pf"), root)
}

func includeServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}
