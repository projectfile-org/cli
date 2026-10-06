// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"kiota.ch/projectfile/core/v2/pkg/projectfile"
)

func TestDropRedundantIncludes_RemovesTransitiveEntry(t *testing.T) {
	raw := map[string]any{"includes": []any{"a.yaml", "b.yaml"}}
	redundant := []projectfile.RedundantInclude{{Ref: "b.yaml", Via: "a.yaml", Reason: "transitive"}}
	pruned := dropRedundantIncludes(raw, redundant)
	assert.Len(t, pruned, 1)
	assert.Equal(t, []any{"a.yaml"}, raw["includes"])
}

func TestDropRedundantIncludes_DuplicateKeepsFirst(t *testing.T) {
	raw := map[string]any{"includes": []any{"a.yaml", "a.yaml"}}
	redundant := []projectfile.RedundantInclude{{Ref: "a.yaml", Via: "a.yaml", Reason: "duplicate"}}
	pruned := dropRedundantIncludes(raw, redundant)
	assert.Len(t, pruned, 1)
	assert.Equal(t, []any{"a.yaml"}, raw["includes"])
}

func TestDropRedundantIncludes_NothingToDo(t *testing.T) {
	raw := map[string]any{"includes": []any{"a.yaml"}}
	assert.Empty(t, dropRedundantIncludes(raw, nil))
	assert.Equal(t, []any{"a.yaml"}, raw["includes"])
	assert.Empty(t, dropRedundantIncludes(map[string]any{}, []projectfile.RedundantInclude{{Ref: "a.yaml"}}))
}
