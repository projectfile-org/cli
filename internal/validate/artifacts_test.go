// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package validate_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"projectfile.org/projectfile/cli/v3/internal/validate"
)

// excludeDoc builds a document whose one artifact carries extra fields plus the given exclude cells.
func excludeDoc(project, artifact map[string]any, cells ...map[string]any) map[string]any {
	list := make([]any, 0, len(cells))
	for _, c := range cells {
		list = append(list, c)
	}
	artifact["exclude"] = list
	project["artifacts"] = map[string]any{"cli": artifact}
	return map[string]any{"org": map[string]any{"projectfile": project}}
}

func platforms() map[string]any {
	return map[string]any{
		"operating-system": []any{"linux", "darwin", "windows"},
		"architecture":     []any{"amd64", "arm64", "riscv64"},
	}
}

func TestArtifactExcludes(t *testing.T) {
	for _, tc := range []struct {
		name string
		doc  map[string]any
		want []string
	}{
		{
			name: "declared cells",
			doc:  excludeDoc(platforms(), map[string]any{}, map[string]any{"os": "darwin", "arch": "riscv64"}, map[string]any{"os": "windows"}),
		},
		{
			name: "undeclared project value",
			doc:  excludeDoc(platforms(), map[string]any{}, map[string]any{"os": "freebsd"}),
			want: []string{`org.projectfile.artifacts.cli.exclude[0]: os "freebsd" is not in org.projectfile.operating-system [linux darwin windows]`},
		},
		{
			name: "artifact own list",
			doc:  excludeDoc(platforms(), map[string]any{"architecture": []any{"amd64"}}, map[string]any{"arch": "arm64"}),
			want: []string{`org.projectfile.artifacts.cli.exclude[0]: arch "arm64" is not in architecture [amd64]`},
		},
		{
			name: "no list declared",
			doc:  excludeDoc(map[string]any{}, map[string]any{}, map[string]any{"os": "linux"}),
			want: []string{`org.projectfile.artifacts.cli.exclude[0]: os "linux" is not in org.projectfile.operating-system []`},
		},
		{
			name: "no artifacts",
			doc:  minimalValidDoc(),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, validate.ArtifactExcludes(tc.doc))
		})
	}
}
