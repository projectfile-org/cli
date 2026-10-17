// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package validate

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"kiota.ch/projectfile/core/v2/pkg/genlog"
	"kiota.ch/projectfile/core/v2/pkg/userconfig"
)

const (
	keyVisibility = "visibility"
	keyLinks      = "links"
	keyType       = "type"
	keyRegistry   = "registry"
	keyTags       = "tags"
	linkSource    = "source-code"
	tagPublic     = "public"
)

// VisibilityContradiction reports a document that resolves to private while
// claiming its own forge landing page is anonymously reachable. Only a
// links[type=source-code] entry counts: the same `public` tag legitimately rides
// any other link type, because a private project can and does point at a public
// upstream or a public documentation site.
func VisibilityContradiction(doc map[string]any) []string {
	if !resolvesPrivate(doc) {
		return nil
	}
	var out []string
	for _, l := range typedLinks(doc) {
		if l["type"] != linkSource || !carriesTag(l, tagPublic) {
			continue
		}
		out = append(out, fmt.Sprintf("identity.visibility: %s with a links[type=source-code] entry tagged %q: %s",
			declaredVisibility(doc), tagPublic, stringOr(l["url"], "(no url)")))
	}
	return out
}

// VisibilityAdvisories reports evidence that a project resolving to private did
// not mean to. Both findings need evidence: warning on mere absence would fire
// forever on a project that is genuinely private.
func VisibilityAdvisories(doc map[string]any) []string {
	if !resolvesPrivate(doc) {
		return nil
	}
	var out []string
	for _, name := range artifactNames(doc) {
		artifact, _ := artifactNode(doc)[name].(map[string]any)
		registry, _ := artifact[keyRegistry].(string)
		if registry == "" {
			continue
		}
		if own, _ := artifact[keyVisibility].(string); own == "public" {
			continue
		}
		out = append(out, fmt.Sprintf("org.projectfile.artifacts.%s declares registry %q but identity.visibility is %s and the artifact does not opt in: publish sinks skip it",
			name, registry, declaredVisibility(doc)))
	}
	for _, l := range typedLinks(doc) {
		if l["type"] != linkSource {
			continue
		}
		host := hostOf(stringOr(l["url"], ""))
		if host == "" || !userconfig.IsPrivateHost(host) {
			continue
		}
		out = append(out, fmt.Sprintf("links[type=source-code] points at %s, a configured private host, while identity.visibility is %s",
			host, declaredVisibility(doc)))
	}
	return out
}

// declaredVisibility names the field's value as the document reads it, with the
// spec's default spelled out so the line says which state it is reporting.
func declaredVisibility(doc map[string]any) string {
	if v := stringOr(identityNode(doc)[keyVisibility], ""); v != "" {
		return v
	}
	return "undeclared (private by default)"
}

// resolvesPrivate applies the §4.3 default: only an explicit public is public.
func resolvesPrivate(doc map[string]any) bool {
	return stringOr(identityNode(doc)[keyVisibility], "") != "public"
}

func identityNode(doc map[string]any) map[string]any {
	m, _ := doc["identity"].(map[string]any)
	return m
}

// typedLinks returns the links[] entries that are mappings, in document order.
func typedLinks(doc map[string]any) []map[string]any {
	raw, _ := doc[keyLinks].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

// carriesTag reads the §139 tags channel as a LIST; a bare string is a
// hand-editing mistake the alias deriver ignores, so the validator does too.
func carriesTag(link map[string]any, tag string) bool {
	items, _ := link[keyTags].([]any)
	for _, item := range items {
		if s, ok := item.(string); ok && s == tag {
			return true
		}
	}
	return false
}

func artifactNode(doc map[string]any) map[string]any {
	m, _ := child(child(doc, "org"), "projectfile").(map[string]any)
	artifacts, _ := m["artifacts"].(map[string]any)
	return artifacts
}

func artifactNames(doc map[string]any) []string {
	artifacts := artifactNode(doc)
	names := make([]string, 0, len(artifacts))
	for name := range artifacts {
		names = append(names, name)
	}
	sort.Strings(names)
	genlog.Debug("visibility artifact scan", "artifacts", len(names))
	return names
}

func hostOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}

func stringOr(v any, fallback string) string {
	s, ok := v.(string)
	if !ok || s == "" {
		return fallback
	}
	return s
}
