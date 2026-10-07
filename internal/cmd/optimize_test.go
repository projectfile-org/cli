// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"kiota.ch/projectfile/core/v2/pkg/projectfile"
)

const includeA = "a.yaml"

func TestDropRedundantIncludes_RemovesTransitiveEntry(t *testing.T) {
	raw := map[string]any{includesKey: []any{includeA, "b.yaml"}}
	redundant := []projectfile.RedundantInclude{{Ref: "b.yaml", Via: includeA, Reason: "transitive"}}
	pruned := dropRedundantIncludes(raw, redundant)
	assert.Len(t, pruned, 1)
	assert.Equal(t, []any{includeA}, raw[includesKey])
}

func TestDropRedundantIncludes_DuplicateKeepsFirst(t *testing.T) {
	raw := map[string]any{includesKey: []any{includeA, includeA}}
	redundant := []projectfile.RedundantInclude{{Ref: includeA, Via: includeA, Reason: "duplicate"}}
	pruned := dropRedundantIncludes(raw, redundant)
	assert.Len(t, pruned, 1)
	assert.Equal(t, []any{includeA}, raw[includesKey])
}

func TestDropRedundantIncludes_NothingToDo(t *testing.T) {
	raw := map[string]any{includesKey: []any{includeA}}
	assert.Empty(t, dropRedundantIncludes(raw, nil))
	assert.Equal(t, []any{includeA}, raw[includesKey])
	assert.Empty(t, dropRedundantIncludes(map[string]any{}, []projectfile.RedundantInclude{{Ref: includeA}}))
}
