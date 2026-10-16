// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package validate

import (
	"fmt"
	"slices"
	"sort"

	"kiota.ch/projectfile/core/v2/pkg/genlog"
)

// excludeAxes pairs each exclude cell key with the list it filters (spec §4.8a).
var excludeAxes = []struct{ key, list string }{
	{"os", "operating-system"},
	{"arch", "architecture"},
}

// ArtifactExcludes reports every org.projectfile.artifacts exclude cell naming a value outside the list it filters.
func ArtifactExcludes(doc map[string]any) []string {
	projectfile, _ := child(child(doc, "org"), "projectfile").(map[string]any)
	artifacts, _ := projectfile["artifacts"].(map[string]any)
	names := make([]string, 0, len(artifacts))
	for name := range artifacts {
		names = append(names, name)
	}
	sort.Strings(names)
	var out []string
	for _, name := range names {
		artifact, _ := artifacts[name].(map[string]any)
		cells, _ := artifact["exclude"].([]any)
		genlog.Debug("artifact exclude check", "artifact", name, "cells", len(cells))
		for i, item := range cells {
			cell, _ := item.(map[string]any)
			for _, axis := range excludeAxes {
				value, ok := cell[axis.key].(string)
				if !ok {
					continue
				}
				list, source := effectiveList(artifact, projectfile, axis.list)
				if !slices.Contains(list, value) {
					out = append(out, fmt.Sprintf("org.projectfile.artifacts.%s.exclude[%d]: %s %q is not in %s %v",
						name, i, axis.key, value, source, list))
				}
			}
		}
	}
	return out
}

// effectiveList returns the artifact's own list, else the project-level one, and names which it read.
func effectiveList(artifact, projectfile map[string]any, key string) ([]string, string) {
	if list, ok := artifact[key].([]any); ok {
		return stringItems(list), key
	}
	list, _ := projectfile[key].([]any)
	return stringItems(list), "org.projectfile." + key
}

func stringItems(list []any) []string {
	out := make([]string, 0, len(list))
	for _, v := range list {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func child(node any, key string) any {
	m, _ := node.(map[string]any)
	return m[key]
}
