// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package validate

import (
	"fmt"
	"sort"
	"strings"

	"kiota.ch/projectfile/core/v2/pkg/genlog"
)

// unitPrefix marks a need that names another unit of the same deployment.
const unitPrefix = "unit:"

// DeployReferences reports every org.projectfile.deploy and org.projectfile.health name that resolves to nothing.
func DeployReferences(doc map[string]any) []string {
	projectfile, _ := child(child(doc, "org"), "projectfile").(map[string]any)
	artifacts, _ := projectfile["artifacts"].(map[string]any)
	units, _ := projectfile["deploy"].(map[string]any)
	var out []string
	for _, unit := range sortedKeys(units) {
		at := "org.projectfile.deploy." + unit
		body, _ := units[unit].(map[string]any)
		runs, _ := body["runs"].(string)
		artifact, declared := artifacts[runs].(map[string]any)
		genlog.Debug("deploy reference check", "unit", unit, "runs", runs, "declared", declared)
		switch {
		case runs == "":
			continue
		case !declared:
			out = append(out, fmt.Sprintf("%s.runs: artifact %q is not declared in org.projectfile.artifacts", at, runs))
			continue
		case !runnable(artifact):
			out = append(out, fmt.Sprintf("%s.runs: artifact %q has no ports and is not kind: service", at, runs))
		}
		out = append(out, offerReferences(at, runs, artifact, body)...)
		out = append(out, needReferences(at, runs, artifact, units, body)...)
	}
	health, _ := projectfile["health"].(map[string]any)
	overrides, _ := health["artifacts"].(map[string]any)
	for _, name := range sortedKeys(overrides) {
		if _, ok := artifacts[name]; !ok {
			out = append(out, fmt.Sprintf("org.projectfile.health.artifacts.%s: artifact %q is not declared in org.projectfile.artifacts", name, name))
		}
	}
	return out
}

// runnable reports whether an artifact is something a unit can run.
func runnable(artifact map[string]any) bool {
	_, ports := artifact["ports"]
	return ports || artifact["kind"] == "service"
}

// offerReferences checks each offer's port against the artifact's named ports and its public name against `speaks: http`.
func offerReferences(at, runs string, artifact, unit map[string]any) []string {
	offers, _ := unit["offers"].(map[string]any)
	ports := namedPorts(artifact)
	var out []string
	for _, offer := range sortedKeys(offers) {
		body, _ := offers[offer].(map[string]any)
		port, _ := body["port"].(string)
		speaks, named := ports[port]
		genlog.Debug("deploy offer check", "offer", offer, "port", port, "named", named, "speaks", speaks)
		if !named {
			out = append(out, fmt.Sprintf("%s.offers.%s.port: artifact %q has no port named %q", at, offer, runs, port))
			continue
		}
		if _, public := body["at"]; public && speaks != "http" {
			out = append(out, fmt.Sprintf("%s.offers.%s.at: port %q of artifact %q speaks %q, a public name needs speaks: http", at, offer, port, runs, speaks))
		}
	}
	return out
}

// namedPorts maps each named port of an artifact to what it speaks.
func namedPorts(artifact map[string]any) map[string]string {
	list, _ := artifact["ports"].([]any)
	out := map[string]string{}
	for _, item := range list {
		port, _ := item.(map[string]any)
		if name, ok := port["name"].(string); ok {
			out[name], _ = port["speaks"].(string)
		}
	}
	return out
}

// needReferences checks each need's unit name against the deployment and each filled setting against the artifact.
func needReferences(at, runs string, artifact, units, unit map[string]any) []string {
	needs, _ := unit["needs"].(map[string]any)
	settings, _ := artifact["settings"].(map[string]any)
	var out []string
	for _, need := range sortedKeys(needs) {
		capability, _ := needs[need].(string)
		body, _ := needs[need].(map[string]any)
		if is, ok := body["is"].(string); ok {
			capability = is
		}
		genlog.Debug("deploy need check", "need", need, "is", capability)
		if target, ok := strings.CutPrefix(capability, unitPrefix); ok {
			if _, found := units[target]; !found {
				out = append(out, fmt.Sprintf("%s.needs.%s: unit %q is not declared in org.projectfile.deploy", at, need, target))
			}
		}
		fills, _ := body["fills"].(map[string]any)
		for _, part := range sortedKeys(fills) {
			setting, _ := fills[part].(string)
			if _, found := settings[setting]; !found {
				out = append(out, fmt.Sprintf("%s.needs.%s.fills.%s: artifact %q has no setting %q", at, need, part, runs, setting))
			}
		}
	}
	return out
}

func sortedKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
