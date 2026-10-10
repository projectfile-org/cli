// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"kiota.ch/projectfile/core/v2/pkg/projectfile"
)

// yamlLineWidth caps a folded block scalar's physical line: the fleet's yamllint runs line-length: 120, and a document past that cannot be pushed.
const yamlLineWidth = 120

// foldedIndicatorRE matches a folded block scalar (`>-`, `>`, `>2-`, `>+`) — the one style whose single line break folds back to a space; a `>` inside a plain scalar (`a > b`) never matches, since the indicator ends the line.
var foldedIndicatorRE = regexp.MustCompile(`(^|\s)>[-+]?[0-9]?\s*(#.*)?$`)

// writeProjectfile writes the document then restores the fold of any block scalar the writer collapsed onto one line; every mutating command writes through here.
func writeProjectfile(doc *projectfile.Document, path string) error {
	if err := projectfile.Write(doc, path); err != nil {
		return err
	}
	return refoldFoldedScalars(path)
}

// refoldFoldedScalars re-folds a folded (>) block the writer emitted as one long line: the YAML emitter's preferred width is unbounded, so the `>-` marker survives but the author's breaks are gone, and every consumer's line-length rule then fails. The pass is TEXT-level, not a value pass — folding a space into a break is what the author wrote and the reader folds it straight back — and YAML-only, the one encoding carrying block scalars.
func refoldFoldedScalars(path string) error {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
	default:
		return nil
	}
	data, err := os.ReadFile(path) // #nosec G304 -- path validated by Read/DetectPath
	if err != nil {
		return err
	}
	out, changed := refoldFoldedBlocks(string(data))
	if !changed {
		return nil
	}
	return os.WriteFile(path, []byte(out), 0o600) // #nosec G306,G703 -- metadata file, not a secret; path validated upstream
}

// refoldFoldedBlocks rewraps every over-long content line of a folded block at single spaces; an empty line inside the block is a paragraph break in the value, and a line with no safe fold point is left alone.
func refoldFoldedBlocks(text string) (string, bool) {
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	changed := false
	for i := 0; i < len(lines); {
		line := lines[i]
		out = append(out, line)
		i++
		if !foldedIndicatorRE.MatchString(line) {
			continue
		}
		indicatorIndent := indentOf(line)
		for i < len(lines) {
			body := lines[i]
			if strings.TrimSpace(body) == "" {
				out = append(out, body)
				i++
				continue
			}
			if indentOf(body) <= indicatorIndent {
				break // the block ended at a line back at the key's indent
			}
			if wrapped, ok := wrapAtSpaces(body, yamlLineWidth); ok {
				out = append(out, wrapped...)
				changed = true
			} else {
				out = append(out, body)
			}
			i++
		}
	}
	return strings.Join(out, "\n"), changed
}

// wrapAtSpaces breaks line at isolated spaces so no line exceeds width; false when nothing needs folding or no safe point exists.
func wrapAtSpaces(line string, width int) ([]string, bool) {
	if len(line) <= width {
		return nil, false
	}
	indent := line[:indentOf(line)]
	content := line[indentOf(line):]
	// budget is the content length that keeps the whole line inside width; a line breaks at the last space whose next word still fits, so no output line exceeds it.
	budget := width - len(indent)
	var out []string
	var b strings.Builder
	folded := false
	for i := range len(content) {
		c := content[i]
		if c == ' ' && isFoldPoint(content, i) {
			next := nextWordLen(content, i+1)
			if b.Len()+1+next > budget {
				out = append(out, indent+strings.TrimRight(b.String(), " "))
				b.Reset()
				folded = true
				continue
			}
		}
		b.WriteByte(c)
	}
	if !folded {
		return nil, false
	}
	out = append(out, indent+strings.TrimRight(b.String(), " "))
	return out, true
}

// isFoldPoint reports whether the space at i is isolated — no space or newline on either side — so folding there round-trips to exactly one space.
func isFoldPoint(s string, i int) bool {
	if i > 0 {
		if c := s[i-1]; c == ' ' || c == '\n' {
			return false
		}
	}
	if i+1 < len(s) {
		if c := s[i+1]; c == ' ' || c == '\n' {
			return false
		}
	}
	return true
}

// nextWordLen is the word length starting at i, up to the next isolated space or the end — the run a break at the previous space moves onto its own line.
func nextWordLen(s string, i int) int {
	n := 0
	for i+n < len(s) && s[i+n] != ' ' && s[i+n] != '\n' {
		n++
	}
	return n
}

// indentOf is the number of leading spaces on one line.
func indentOf(line string) int {
	return len(line) - len(strings.TrimLeft(line, " "))
}
