// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"strings"

	"kiota.ch/projectfile/core/v2/pkg/projectfile"
)

// derivedFields are SYNTHETIC `get` addresses: values COMPUTED from real
// document fields rather than read from one. They are the single home of a
// rule that several consumers would otherwise each re-derive — so a project's
// container-image basename, for instance, is owned by projectfile.ImageBasename
// and read identically by m6e (M6E_IMAGE_BASENAME via `projectfile get
// image.basename`) and ci-resolver (pf-ci, via the pkg façade). runGet consults
// this map BEFORE document resolution, so a synthetic address wins; the function
// returns (value, ok=false) to signal "derived but not determinable" (treated as
// a normal missing value, so --default / --or-default still apply). Keys are the
// canonical dotted address.
// Canonical dotted addresses of the synthetic image.* fields — the keys
// derivedFields is indexed by, and the strings the get help text names.
const (
	addrImageBasename  = "image.basename"
	addrImageNamespace = "image.namespace"
	addrImageName      = "image.name"
)

var derivedFields = map[string]func(*projectfile.Document) (string, bool){
	addrImageBasename:  projectfile.ImageBasename,
	addrImageNamespace: imageNamespace,
	addrImageName:      imageName,
}

// imageNamespace / imageName split the synthetic image.basename into its two
// identity halves — the last `/`-separated label as the namespace, the rest
// (minus any `:tag`) as the name. They are the make-plane home of the
// `${image.namespace}` / `${image.name}` references the m6e reader interpolates
// (via `pf-cli get`), mirroring the ci-resolver interp split byte-for-byte so a
// build-arg identity value can never disagree across the two lowerings. Present
// iff the basename itself is determinable; a bare (namespace-less) basename
// yields an EMPTY-but-present namespace (an empty identity arg is valid), matching
// the ci-resolver behaviour.
func imageNamespace(doc *projectfile.Document) (string, bool) {
	base, ok := projectfile.ImageBasename(doc)
	if !ok {
		return "", false
	}
	ns, _ := splitBasename(base)
	return ns, true
}

func imageName(doc *projectfile.Document) (string, bool) {
	base, ok := projectfile.ImageBasename(doc)
	if !ok {
		return "", false
	}
	_, name := splitBasename(base)
	return name, true
}

// splitBasename splits an image basename into (namespace, name): the last `/`
// label is the namespace, the remainder (minus any `:tag`) the name. `b19/ubuntu`
// → (b19, ubuntu); a bare `ubuntu` → ("", ubuntu). The basename RULE itself lives
// in core (projectfile.ImageBasename); only this trivial split is local (kept in
// sync with ci-resolver's basenameParts).
func splitBasename(image string) (namespace, name string) {
	name = image
	if i := strings.LastIndex(name, "/"); i >= 0 {
		namespace, name = name[:i], name[i+1:]
	}
	if i := strings.LastIndex(name, ":"); i >= 0 {
		name = name[:i]
	}
	return namespace, name
}
