// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package validate_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"projectfile.org/projectfile/cli/v3/internal/validate"
)

const (
	testIdentityKey    = "identity"
	testNameKey        = "name"
	testProjectfileKey = "projectfile"
	testTypeKey        = "type"
	testKindKey        = "kind"
	testPublic         = "public"
	testNPM            = "npm"
	testPackage        = "package"
)

// visibilityDoc builds a document with the given identity.visibility plus optional links and one artifact.
func visibilityDoc(vis string, links []any, artifacts map[string]any) map[string]any {
	identity := map[string]any{"namespace": "com.acme", testNameKey: "x"}
	if vis != "" {
		identity["visibility"] = vis
	}
	doc := map[string]any{testIdentityKey: identity}
	if links != nil {
		doc["links"] = links
	}
	if artifacts != nil {
		doc["org"] = map[string]any{testProjectfileKey: map[string]any{"artifacts": artifacts}}
	}
	return doc
}

func sourceCodeLink(url string, tags ...any) any {
	link := map[string]any{testTypeKey: "source-code", testURLKey: url}
	if tags != nil {
		link["tags"] = tags
	}
	return link
}

func TestVisibilityContradiction(t *testing.T) {
	for _, tc := range []struct {
		name string
		doc  map[string]any
		want []string
	}{
		{
			name: "undeclared with a public-tagged forge",
			doc:  visibilityDoc("", []any{sourceCodeLink("https://kiota.ch/a/b", testPublic)}, nil),
			want: []string{`identity.visibility: undeclared (private by default) with a links[type=source-code] entry tagged "public": https://kiota.ch/a/b`},
		},
		{
			name: "explicit private with a public-tagged forge",
			doc:  visibilityDoc("private", []any{sourceCodeLink("https://kiota.ch/a/b", "badges", testPublic)}, nil),
			want: []string{`identity.visibility: private with a links[type=source-code] entry tagged "public": https://kiota.ch/a/b`},
		},
		{
			name: "public is never a contradiction",
			doc:  visibilityDoc(testPublic, []any{sourceCodeLink("https://kiota.ch/a/b", testPublic)}, nil),
		},
		{
			name: "an untagged private forge is fine",
			doc:  visibilityDoc("private", []any{sourceCodeLink("https://forge.example/a/b")}, nil),
		},
		{
			name: "the tag on another link type is not the project's own claim",
			doc:  visibilityDoc("private", []any{map[string]any{testTypeKey: "documentation", testURLKey: "https://spec.example", "tags": []any{testPublic}}}, nil),
		},
		{
			name: "a scalar tags value is a mistake, not a claim",
			doc:  visibilityDoc("private", []any{map[string]any{testTypeKey: "source-code", testURLKey: "https://kiota.ch/a/b", "tags": testPublic}}, nil),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, validate.VisibilityContradiction(tc.doc))
		})
	}
}

func TestVisibilityAdvisories(t *testing.T) {
	registry := map[string]any{testKindKey: testPackage, "registry": testNPM, testNameKey: "x"}
	optIn := map[string]any{testKindKey: testPackage, "registry": testNPM, testNameKey: "x", "visibility": testPublic}
	for _, tc := range []struct {
		name string
		doc  map[string]any
		want []string
	}{
		{
			name: "private project shipping a package nobody opted in",
			doc:  visibilityDoc("private", nil, map[string]any{testNPM: registry}),
			want: []string{`org.projectfile.artifacts.npm declares registry "npm" but identity.visibility is private and the artifact does not opt in: publish sinks skip it`},
		},
		{
			name: "undeclared reads as private and says so",
			doc:  visibilityDoc("", nil, map[string]any{testNPM: registry}),
			want: []string{`org.projectfile.artifacts.npm declares registry "npm" but identity.visibility is undeclared (private by default) and the artifact does not opt in: publish sinks skip it`},
		},
		{
			name: "the artifact opt-in silences it",
			doc:  visibilityDoc("private", nil, map[string]any{testNPM: optIn}),
		},
		{
			name: "an artifact with no registry is not a publication",
			doc:  visibilityDoc("private", nil, map[string]any{testNPM: map[string]any{testKindKey: testPackage, testNameKey: "x"}}),
		},
		{
			name: "a public project has nothing to be advised about",
			doc:  visibilityDoc(testPublic, nil, map[string]any{testNPM: registry}),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, validate.VisibilityAdvisories(tc.doc))
		})
	}
}
