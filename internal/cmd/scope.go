// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"kiota.ch/projectfile/core/v2/pkg/fieldpath"
	"kiota.ch/projectfile/core/v2/pkg/genlog"
	"kiota.ch/projectfile/core/v2/pkg/interp"
	"kiota.ch/projectfile/core/v2/pkg/projectfile"
)

// What we are trying to do: let a shell read a value that the document COMPOSES
// rather than states. A destination reference is a template over declared parts —
//
//	image: {org: b19, name: ${identity.name}, series: resolute, tag: latest}
//	sinks:
//	  kiota: {ref: "kiota.ch/${org}/${name}-${series}:${tag}"}
//
// — so `get org.projectfile.sinks.kiota.ref --scope org.projectfile.image`
// answers `kiota.ch/b19/ubuntu-resolute:latest` in one spawn. Without a scope,
// the same address answers the template verbatim, because `${org}` is not a
// document address and nothing here invents one.
//
// --scope is what turns expansion ON. A plain `get` resolves exactly what it
// always did, so no existing caller changes behaviour on the day this ships.

// resolveScoped resolves p under the scopes, then the document root — the same
// order interp uses, so a template and a shell reading the same address can
// never disagree about which scope answered.
//
// present is false only when NO scope and not the root answers. A miss inside a
// scope is ordinary: it is how the fall-through works, and it is what lets a
// caller pass an optional scope without first testing that it exists.
func resolveScoped(doc *projectfile.Document, p fieldpath.Path, scopes []string) (value any, isList, isPairs, present bool, err error) {
	for _, scope := range scopes {
		scoped, parseErr := fieldpath.Parse(scope + "." + p.String())
		if parseErr != nil {
			genlog.Warn("get: scope is not a field address — skipped", "scope", scope, "error", parseErr.Error())
			continue
		}
		r, resolveErr := fieldpath.Resolve(doc, scoped)
		if resolveErr != nil {
			continue
		}
		genlog.Decision("get_scope", p.String(), scope, "")
		return singleOrList(r), r.IsList, r.IsPairs, true, nil
	}
	return resolveEntry(doc, p)
}

// expandScoped expands every `${…}` in a resolved value against the scopes.
//
// It runs on the VALUE, not on the address, because that is where a template
// lives: `ref` holds the string, and reading it without expanding would hand the
// shell a reference with holes in it. Only strings are touched — a number or a
// list element that is not a string has no references to resolve.
//
// A value that does not fully resolve is returned VERBATIM rather than
// half-composed, and reported. A reference that silently lost a segment is worse
// than one that visibly did not resolve: the first is a push to the wrong place,
// the second stops the build.
func expandScoped(value any, doc *projectfile.Document, scopes []string) any {
	if len(scopes) == 0 {
		return value
	}
	switch v := value.(type) {
	case string:
		out, resolved := interp.ExpandIn(doc, v, scopes...)
		if !resolved {
			genlog.Warn("get: value did not fully resolve under the given scopes",
				"template", v, "composed", out, "scopes", len(scopes))
			return v
		}
		return out
	case []any:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, expandScoped(item, doc, scopes))
		}
		return out
	case fieldpath.Pair:
		return fieldpath.Pair{Key: v.Key, Value: expandScoped(v.Value, doc, scopes)}
	case map[string]any:
		// A map projection (`sinks{}.values`) hands back whole ENTRIES, and the
		// template lives on a key inside one. Descending is what lets ONE spawn
		// compose EVERY declared destination: without it the caller reads the
		// templates back verbatim and must spawn again per entry to resolve them.
		out := make(map[string]any, len(v))
		for k, item := range v {
			out[k] = expandScoped(item, doc, scopes)
		}
		return out
	default:
		return value
	}
}
