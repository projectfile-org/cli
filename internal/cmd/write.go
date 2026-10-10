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

// yamlLineWidth is the physical line a folded block scalar may reach before
// it is re-folded. The fleet's yamllint runs line-length: 120, and a document
// past that cannot be pushed at all.
const yamlLineWidth = 120

// foldedIndicatorRE matches a line whose value is a folded block scalar
// (`>-`, `>`, `>2-`, `>+`), the one style whose single line break folds back
// to a space. A `>` inside a plain scalar (`a > b`) never matches: the
// indicator must end the line, modulo a comment and chomping digits.
var foldedIndicatorRE = regexp.MustCompile(`(^|\s)>[-+]?[0-9]?\s*(#.*)?$`)

// writeProjectfile writes the document and then restores the fold of any
// block scalar the writer collapsed onto one physical line. Every command
// that writes the projectfile goes through here, so one place owns the
// round-trip.
func writeProjectfile(doc *projectfile.Document, path string) error {
	if err := projectfile.Write(doc, path); err != nil {
		return err
	}
	return refoldFoldedScalars(path)
}

// refoldFoldedScalars re-folds a folded (>) block scalar the writer emitted as
// one long line. The YAML emitter's preferred line width is unbounded, so a
// value it holds as one string always comes back as a single physical line —
// the `>-` marker survives but the breaks the author wrote are gone, and a
// long description then exceeds every consumer's line-length rule.
//
// This is a TEXT pass, not a value pass: folding a space into a line break is
// what the author wrote, and the reader folds it straight back, so the value
// is untouched. Only YAML carries block scalars.
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

// refoldFoldedBlocks rewraps every over-long content line of a folded block
// at single spaces. A blank line inside the block is a paragraph break in the
// value and is left alone, and a line with no safe fold point is left alone.
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

// wrapAtSpaces breaks line at isolated spaces so no line exceeds width.
// Reports false when nothing needs folding or no safe point exists.
func wrapAtSpaces(line string, width int) ([]string, bool) {
	if len(line) <= width {
		return nil, false
	}
	indent := line[:indentOf(line)]
	content := line[indentOf(line):]
	// budget is what the content may occupy so the whole line, indent
	// included, stays inside width. A line breaks at the last space whose
	// next word still fits, so no output line exceeds the budget.
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

// isFoldPoint reports whether the space at i is isolated — no space or
// newline on either side — so folding there round-trips to exactly one space.
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

// nextWordLen is the length of the word starting at i, up to the next isolated
// space or the end — the run a line break at the previous space would move
// onto its own line.
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
