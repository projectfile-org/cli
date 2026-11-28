// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package validate_test

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"

	"projectfile.org/projectfile/cli/v3/internal/validate"
)

// deployDoc loads the merged specification fixture, a document whose every deploy reference resolves.
func deployDoc(t *testing.T) map[string]any {
	t.Helper()
	data, err := os.ReadFile("testdata/deploy.yaml")
	require.NoError(t, err)
	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(data, &doc))
	return doc
}

// at walks a dotted path below org.projectfile and returns the mapping there.
func at(doc map[string]any, dotted string) map[string]any {
	node := doc["org"].(map[string]any)["projectfile"].(map[string]any)
	for _, key := range strings.Split(dotted, ".") {
		node = node[key].(map[string]any)
	}
	return node
}

func TestDeployReferencesPassTheSpecificationFixture(t *testing.T) {
	doc := deployDoc(t)
	assert.Empty(t, validate.DeployReferences(doc))
	require.NoError(t, validate.Namespaces(doc))
}

func TestDeployReferencesRefuseEachBrokenName(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(doc map[string]any)
		want   string
	}{
		{
			name:   "runs names no artifact",
			mutate: func(doc map[string]any) { at(doc, "deploy.cache")["runs"] = "redis" },
			want:   `org.projectfile.deploy.cache.runs: artifact "redis" is not declared in org.projectfile.artifacts`,
		},
		{
			name: "runs names an artifact with no ports that is not a service",
			mutate: func(doc map[string]any) {
				at(doc, "artifacts.forgejo-prune")["kind"] = "image"
			},
			want: `org.projectfile.deploy.prune.runs: artifact "forgejo-prune" has no ports and is not kind: service`,
		},
		{
			name:   "offer names no port",
			mutate: func(doc map[string]any) { at(doc, "deploy.cache.offers.resp")["port"] = "redis" },
			want:   `org.projectfile.deploy.cache.offers.resp.port: artifact "valkey" has no port named "redis"`,
		},
		{
			name:   "public name on a port that speaks no http",
			mutate: func(doc map[string]any) { at(doc, "deploy.forge.offers.git")["at"] = "ssh.example.org" },
			want:   `org.projectfile.deploy.forge.offers.git.at: port "ssh" of artifact "forgejo" speaks "ssh", a public name needs speaks: http`,
		},
		{
			name: "public name on a port that says nothing about what it speaks",
			mutate: func(doc map[string]any) {
				ports := at(doc, "artifacts.website")["ports"].([]any)
				delete(ports[0].(map[string]any), "speaks")
			},
			want: `org.projectfile.deploy.site.offers.web.at: port "http" of artifact "website" speaks "", a public name needs speaks: http`,
		},
		{
			name:   "need names no unit",
			mutate: func(doc map[string]any) { at(doc, "deploy.forge.needs")["speed"] = "unit:memcached" },
			want:   `org.projectfile.deploy.forge.needs.speed: unit "memcached" is not declared in org.projectfile.deploy`,
		},
		{
			name:   "object need names no unit",
			mutate: func(doc map[string]any) { at(doc, "deploy.forge.needs.records")["is"] = "unit:mysql" },
			want:   `org.projectfile.deploy.forge.needs.records: unit "mysql" is not declared in org.projectfile.deploy`,
		},
		{
			name:   "fill names no setting",
			mutate: func(doc map[string]any) { at(doc, "deploy.forge.needs.records.fills")["host"] = "db-hostname" },
			want:   `org.projectfile.deploy.forge.needs.records.fills.host: artifact "forgejo" has no setting "db-hostname"`,
		},
		{
			name: "health override names no artifact",
			mutate: func(doc map[string]any) {
				at(doc, "health.artifacts")["wireguard"] = map[string]any{"liveness": map[string]any{"command": []any{"true"}}}
			},
			want: `org.projectfile.health.artifacts.wireguard: artifact "wireguard" is not declared in org.projectfile.artifacts`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := deployDoc(t)
			tc.mutate(doc)
			assert.Equal(t, []string{tc.want}, validate.DeployReferences(doc))
		})
	}
}

func TestDeployReferencesAcceptAPortedArtifactOfAnyKind(t *testing.T) {
	doc := deployDoc(t)
	delete(at(doc, "artifacts.valkey"), "kind")
	assert.Empty(t, validate.DeployReferences(doc))
}

func TestDeployReferencesIgnoreADocumentWithoutDeploy(t *testing.T) {
	assert.Empty(t, validate.DeployReferences(minimalValidDoc()))
}

func TestNamespacesRefuseABrokenSubtreeAtItsDocumentPath(t *testing.T) {
	doc := deployDoc(t)
	at(doc, "deploy.prune")["shape"] = "cron"
	err := validate.Namespaces(doc)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "/org/projectfile/deploy/prune/shape")
}

func TestNamespacesRefuseAShellStringHealthCheck(t *testing.T) {
	doc := deployDoc(t)
	at(doc, "health")["liveness"] = map[string]any{"command": "livecheck.d || exit 1"}
	err := validate.Namespaces(doc)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "/org/projectfile/health/liveness/command")
}
