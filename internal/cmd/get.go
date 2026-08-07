// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"kiota.ch/projectfile/core/v2/pkg/fieldpath"
	"kiota.ch/projectfile/core/v2/pkg/projectfile"
)

const formatJSON = "json"

var (
	getDefault    string
	getOrDefault  bool
	getFormat     string
	getBatch      bool
	getExists     bool
	getExpandEnv  bool
	getPrintPath  bool
	getPathFile   string
	getNamedPaths []string
	getLang       string
)

var getCmd = &cobra.Command{
	Use:   "get <path>...",
	Short: "Read one or more projectfile fields",
	Long: "Read field values from the projectfile in the current (or named)\n" +
		"directory. Each path uses the dotted + selector + projection grammar:\n" +
		"  identity.namespace\n" +
		"  keywords[0]            keywords[-1]\n" +
		"  repositories[role=origin].url\n" +
		"  repositories[].url     — projection: one item per line\n" +
		"  ext.example.build.user — extension shortcut\n" +
		"\n" +
		"Synthetic (derived) addresses compute a value from other fields:\n" +
		"  image.basename   org.projectfile.ci.image, else\n" +
		"                   <last-label(identity.namespace)>/<identity.name>\n" +
		"  image.namespace  the basename’s namespace half (before the last /)\n" +
		"  image.name       the basename’s name half (after the last /, no :tag)\n" +
		"\n" +
		"Default output is raw — shell-friendly. Use --format json for a single\n" +
		"JSON value, or --format sh to emit `export KEY=value` lines. --batch\n" +
		"reads many paths from one process; combine with --path NAME=ADDR to\n" +
		"control the export key.\n" +
		"\n" +
		"Localized string fields (stored as {lang: value} maps) are unwrapped\n" +
		"automatically for raw and sh output when only one language is present.\n" +
		"Use --lang to select a specific language when several are available.",
	Args: cobra.ArbitraryArgs,
	RunE: runGet,
}

// pathEntry pairs a user-visible key (for --format sh / json batch) with
// the parsed Path. Built from positional args and --path flags.
type pathEntry struct {
	Key  string
	Path fieldpath.Path
}

// resolvedEntry is the per-path outcome the formatter consumes. Promoted
// to package-level so it can appear in emitJSON / emitSh signatures.
// When isPairs is true the value field is the []fieldpath.Pair carried
// out of a `{}` map projection — the per-formatter emit helpers branch
// on the flag instead of on the value type, keeping the shape switch
// in one place.
type resolvedEntry struct {
	key     string
	path    fieldpath.Path
	value   any
	isList  bool
	isPairs bool
	present bool
}

func runGet(cmd *cobra.Command, args []string) error {
	// --print-path: emit the resolved projectfile path (explicit --path-file, else
	// DetectPath in cwd) and exit. Lets callers drop their own mtime/stat probe —
	// pf-cli is the authority on the §4.5 multi-file arbitration. No path entries
	// are required in this mode. No projectfile => silent exit 0 (graceful).
	if getPrintPath {
		p := getPathFile
		if p == "" {
			if dp, err := projectfile.DetectPath("."); err == nil {
				p = dp
			}
		}
		if p == "" {
			return nil
		}
		if abs, err := filepath.Abs(p); err == nil {
			p = abs
		}
		fmt.Println(p)
		return nil
	}
	entries, err := collectPathEntries(args, getNamedPaths)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return errUsage("get requires at least one path or --path entry")
	}
	if !getBatch && len(entries) > 1 {
		return errUsage("multiple paths require --batch")
	}
	// --default's presence is detected via Changed, not a non-empty value, so
	// `--default=''` is honoured: a missing field then yields the empty string
	// (exit 0) instead of the not-found soft failure. This is what makes the
	// flag a reliable absent-vs-broken discriminator — a broken document errors
	// back in loadDocument below, before any default is ever consulted.
	defaultSet := cmd.Flags().Changed("default")
	if defaultSet && getOrDefault {
		return errUsage("--default and --or-default are mutually exclusive")
	}

	doc, err := loadDocument(getPathFile)
	if err != nil {
		return err
	}

	// Resolve each entry once, producing (key, value, present) triples so
	// the formatter is encoding-agnostic. Honour --default / --or-default
	// per entry — a missing value remains "missing" only when neither
	// fallback applies.
	out := make([]resolvedEntry, 0, len(entries))
	missingAny := false
	for _, e := range entries {
		val, isList, isPairs, present, err := resolveEntry(doc, e.Path)
		switch {
		case err != nil:
			return err
		case present:
			out = append(out, resolvedEntry{key: e.Key, path: e.Path, value: val, isList: isList, isPairs: isPairs, present: true})
		default:
			if d, ok := defaultValueFor(doc, e.Path, defaultSet); ok {
				out = append(out, resolvedEntry{key: e.Key, path: e.Path, value: d, present: true})
			} else {
				out = append(out, resolvedEntry{key: e.Key, path: e.Path, present: false})
				missingAny = true
			}
		}
	}

	if getExists {
		// --exists reports presence via exit code only; suppress stdout.
		if missingAny {
			os.Exit(1)
		}
		return nil
	}

	switch getFormat {
	case "raw":
		for _, r := range out {
			if !r.present {
				continue
			}
			if getBatch && !r.isPairs {
				// Pair output is line-oriented KEY=VALUE so the leading
				// "key=" batch prefix would clash; suppress it for
				// pairs and let each line stand on its own.
				fmt.Fprintf(cmd.OutOrStdout(), "%s=", r.key)
			}
			emitRaw(cmd, r.value, r.isList, r.isPairs)
		}
	case formatJSON:
		if err := emitJSON(cmd, out, getBatch); err != nil {
			return err
		}
	case "sh":
		emitSh(cmd, out)
	case "flat":
		emitFlat(cmd, out)
	default:
		return errUsage(fmt.Sprintf("unknown --format %q (raw|json|sh|flat)", getFormat))
	}

	if missingAny && !getExists {
		// Missing values without a fallback are a soft failure: stdout
		// already shows what we *could* resolve, but the process exits 1
		// so shell pipelines can detect the partial result.
		os.Exit(1)
	}
	return nil
}

// resolveEntry produces the (value, present) outcome for one path. A SYNTHETIC
// derived field (see derivedFields) wins over document resolution, so a computed
// rule such as image.basename has one home every consumer reads. Otherwise the
// path resolves against the document, with a not-found mapped to present=false
// (the caller then consults --default / --or-default). Any other resolve error
// propagates — notably fieldpath.ErrListOpOnMap, a `[N]`/`[k=v]`/`[]` operator
// aimed at a map of named keys (e.g. `org.projectfile.artifacts[kind=binary]`):
// that is a GRAMMAR mistake, refused LOUDLY here rather than laundered into a
// silent present=false miss that would read as a typo.
func resolveEntry(doc *projectfile.Document, p fieldpath.Path) (value any, isList, isPairs, present bool, err error) {
	if fn, ok := derivedFields[p.String()]; ok {
		v, ok := fn(doc)
		return v, false, false, ok, nil
	}
	r, err := fieldpath.Resolve(doc, p)
	switch {
	case err == nil:
		return singleOrList(r), r.IsList, r.IsPairs, true, nil
	case errors.Is(err, fieldpath.ErrNotFound):
		return nil, false, false, false, nil
	default:
		return nil, false, false, false, err
	}
}

// defaultValueFor consults the explicit --default value first, then the
// spec-defaults registry when --or-default is set. The two flags are
// mutually exclusive (enforced upstream) so the precedence here is the
// only one that can apply. defaultSet reports whether --default was passed
// at all (its own Changed state, not a non-empty value) so an empty
// --default still returns the empty string rather than falling through.
func defaultValueFor(doc *projectfile.Document, p fieldpath.Path, defaultSet bool) (any, bool) {
	if defaultSet {
		return getDefault, true
	}
	if getOrDefault {
		if v, ok := fieldpath.LookupDefault(doc, p.String()); ok {
			return v, true
		}
	}
	return nil, false
}

// singleOrList flattens a Result into the raw value the formatter expects:
// scalar Single values pass through, projection/whole-list/map-projection
// results stay as the []any slice (the IsList / IsPairs flags travel
// separately on the resolvedEntry).
func singleOrList(r fieldpath.Result) any {
	if r.IsList || r.IsPairs {
		return r.Values
	}
	if len(r.Values) == 1 {
		return r.Values[0]
	}
	return r.Values
}

func emitRaw(cmd *cobra.Command, v any, isList, isPairs bool) {
	out := cmd.OutOrStdout()
	if isPairs {
		// Map projection: one KEY=VALUE line per pair, matching the
		// standard shell-map format. The value is rendered through
		// FormatScalar so nested scalars (numbers, bools) come out
		// human-readable.
		for _, item := range v.([]any) {
			pr := item.(fieldpath.Pair)
			fmt.Fprintf(out, "%s=%s\n", pr.Key, fieldpath.FormatScalar(coerceLang(pr.Value, getLang)))
		}
		return
	}
	// Any list shape — whether produced by a SegProject projection or by a
	// whole-list address like `get keywords` — emits one item per line.
	// FormatScalar's space-join behaviour is reserved for the `sh` format;
	// raw output stays line-oriented so
	// `pf-cli get list | while read line; do ...` is the natural pattern.
	switch x := v.(type) {
	case []any:
		for _, item := range x {
			fmt.Fprintln(out, fieldpath.FormatScalar(coerceLang(item, getLang)))
		}
		return
	case []string:
		for _, s := range x {
			fmt.Fprintln(out, s)
		}
		return
	case []map[string]any:
		for _, m := range x {
			fmt.Fprintln(out, fieldpath.FormatScalar(coerceLang(m, getLang)))
		}
		return
	case map[string]any:
		fmt.Fprintln(out, fieldpath.FormatScalar(coerceLang(x, getLang)))
		return
	}
	_ = isList
	fmt.Fprintln(out, fieldpath.FormatScalar(coerceLang(v, getLang)))
}

func emitJSON(cmd *cobra.Command, entries []resolvedEntry, batch bool) error {
	var payload any
	if batch {
		m := map[string]any{}
		for _, r := range entries {
			if r.present {
				m[r.key] = jsonValue(r)
			}
		}
		payload = m
	} else {
		for _, r := range entries {
			if r.present {
				payload = jsonValue(r)
				break
			}
		}
	}
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

// jsonValue collapses a resolvedEntry into a value json.Encoder will print
// in the encoding the user expects: pair results round-trip to a JSON
// object (same shape the projectfile carries them as), everything else
// passes through unchanged. When --lang is set, localized string maps are
// resolved to a scalar before encoding.
func jsonValue(r resolvedEntry) any {
	if !r.isPairs {
		if getLang != "" {
			return coerceLang(r.value, getLang)
		}
		return r.value
	}
	pairs, ok := r.value.([]any)
	if !ok {
		return r.value
	}
	m := make(map[string]any, len(pairs))
	for _, item := range pairs {
		if p, ok := item.(fieldpath.Pair); ok {
			val := p.Value
			if getLang != "" {
				val = coerceLang(p.Value, getLang)
			}
			m[p.Key] = val
		}
	}
	return m
}

// emitSh writes one `export KEY=VALUE` line per resolved entry. Key
// derivation mirrors the convention projectfile-read.sh uses (uppercase,
// `.`/`-` → `_`, plus the org_projectfile_ prefix strip) so downstream
// Makefiles can drop dasel without renaming a single variable.
// Lists become space-joined strings; map-projection (`env{}`) results
// fan out into one `export KEY=VALUE`
// line per pair, with the user-supplied --path key used only as the
// implicit "group" header in --batch (no enclosing variable).
func emitSh(cmd *cobra.Command, entries []resolvedEntry) {
	for _, r := range entries {
		if !r.present {
			continue
		}
		if r.isPairs {
			for _, item := range r.value.([]any) {
				p := item.(fieldpath.Pair)
				key := shellKey(p.Key)
				val := shellQuote(fieldpath.FormatScalar(coerceLang(p.Value, getLang)))
				fmt.Fprintf(cmd.OutOrStdout(), "export %s=%s\n", key, val)
			}
			continue
		}
		key := shellKey(r.key)
		val := shellValue(r.value)
		fmt.Fprintf(cmd.OutOrStdout(), "export %s=%s\n", key, val)
	}
}

// emitFlat recursively flattens each resolved subtree into one
// `<dotted.path>=<scalar>` line per leaf, the path RELATIVE to the queried
// address (e.g. `get org.projectfile.ci` yields `nodes.<n>.needs.<t>=true`).
// Unlike `sh`, keys are kept VERBATIM (dashes preserved) and the full path is
// retained, so a consumer can read a whole subtree in ONE pf-cli call and
// recover every leaf's attribution unambiguously — path SEGMENTS join on `.`
// and projectfile key names never contain `.`. Lists space-join (FormatScalar);
// maps recurse (key-sorted, matching the `{}` projection's stable order). The
// shell-flatten reader (m6e ci/select.mk) consumes this to collapse hundreds of
// per-leaf `get` spawns into one. In --batch the entry key prefixes each path.
func emitFlat(cmd *cobra.Command, entries []resolvedEntry) {
	for _, r := range entries {
		if !r.present {
			continue
		}
		prefix := ""
		if getBatch {
			prefix = r.key
		}
		flatWalk(cmd, prefix, r.value)
	}
}

// flatWalk emits v under prefix: a map recurses one segment deeper per key, a
// list of maps/lists recurses under an indexed `prefix[i]` segment, and a
// scalar (or scalar-only list) is a leaf rendered through FormatScalar.
//
// The indexed recursion keeps the flat format LOSSLESS for a list-of-maps
// (mounts:, matrix.overrides): FormatScalar renders a map leaf with embedded
// newlines, which collide with the record separator and orphan every field
// past the first. Emitting `mounts[0].from=…` / `mounts[0].to=…` instead gives
// each leaf an unambiguous path — matching the `key[N]` index grammar the
// get/set addressing already speaks. A pure scalar list still space-joins
// (`keywords=a b c`): shell consumers word-split it and must not regress.
func flatWalk(cmd *cobra.Command, prefix string, v any) {
	switch t := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			child := k
			if prefix != "" {
				child = prefix + "." + k
			}
			flatWalk(cmd, child, t[k])
		}
		return
	case []any:
		if listHasComposite(t) {
			for i, item := range t {
				flatWalk(cmd, fmt.Sprintf("%s[%d]", prefix, i), item)
			}
			return
		}
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s=%s\n", prefix, fieldpath.FormatScalar(v))
}

// listHasComposite reports whether any element is a map or nested list — the
// signal that space-joining would lose structure and indexed recursion is
// required. A list of scalars returns false and space-joins as before.
func listHasComposite(xs []any) bool {
	for _, x := range xs {
		switch x.(type) {
		case map[string]any, []any:
			return true
		}
	}
	return false
}

func shellKey(in string) string {
	k := strings.ToUpper(in)
	k = strings.NewReplacer(".", "_", "-", "_").Replace(k)
	for _, prefix := range []string{"ORG_PROJECTFILE_", "EXT_"} {
		k = strings.TrimPrefix(k, prefix)
	}
	return k
}

func shellValue(v any) string {
	v = coerceLang(v, getLang)
	switch x := v.(type) {
	case []any:
		parts := make([]string, len(x))
		for i, item := range x {
			parts[i] = fieldpath.FormatScalar(item)
		}
		return shellQuote(strings.Join(parts, " "))
	case []string:
		return shellQuote(strings.Join(x, " "))
	}
	return shellQuote(fieldpath.FormatScalar(v))
}

// shellQuote wraps s in single quotes after escaping any literal single
// quotes. POSIX-compatible — no $/backtick interpolation, no backslash
// escapes inside the quoted span — so the output is safe even when the
// value contains shell metacharacters.
func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if !needsShellQuote(s) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func needsShellQuote(s string) bool {
	for _, c := range s {
		if c == ' ' || c == '\t' || c == '\n' || c == '"' || c == '\'' ||
			c == '$' || c == '`' || c == '\\' || c == '*' || c == '?' ||
			c == '[' || c == ']' || c == '(' || c == ')' || c == '<' ||
			c == '>' || c == '|' || c == '&' || c == ';' || c == '!' ||
			c == '#' {
			return true
		}
	}
	return false
}

// collectPathEntries builds the (key, path) list from positional args
// and --path KEY=ADDR flags. Positional paths get a key derived from the
// last meaningful segment; --path entries take their key verbatim. Order
// in the output matches command-line order with positionals first.
func collectPathEntries(args, named []string) ([]pathEntry, error) {
	entries := make([]pathEntry, 0, len(args)+len(named))
	for _, a := range args {
		p, err := fieldpath.Parse(a)
		if err != nil {
			return nil, err
		}
		entries = append(entries, pathEntry{Key: deriveKey(a), Path: p})
	}
	for _, n := range named {
		eq := strings.IndexByte(n, '=')
		if eq <= 0 {
			return nil, errUsage(fmt.Sprintf("--path requires KEY=ADDR, got %q", n))
		}
		k := strings.TrimSpace(n[:eq])
		addr := strings.TrimSpace(n[eq+1:])
		p, err := fieldpath.Parse(addr)
		if err != nil {
			return nil, err
		}
		entries = append(entries, pathEntry{Key: k, Path: p})
	}
	// Stable order: positionals as given, then --path flags as given.
	// No sort — the user dictates the order, which `sh` output relies on.
	sort.SliceStable(entries, func(_, _ int) bool { return false })
	return entries, nil
}

// deriveKey turns an address string into the default export-key for the
// `--format sh` case when no explicit --path KEY= override was given. The
// rule: take the trailing dotted segments, strip brackets, uppercase.
func deriveKey(addr string) string {
	s := addr
	// strip everything after the first '['
	if i := strings.IndexByte(s, '['); i >= 0 {
		s = s[:i]
	}
	return s
}

// loadDocument reads the projectfile from explicitPath when set, otherwise
// from the current working directory via DetectPath. When --expand-env is
// set the source bytes are run through a brace-only env-var expansion
// before parsing — the expansion is in-memory (no temp file).
//
// Brace-only is a deliberate divergence from `envsubst`/`os.Expand`: those
// also expand bare `$IDENTIFIER` references, which would obliterate
// `$schema` (the YAML/JSON projectfile discriminator) and break the
// subsequent parse on every YAML/JSON file. Only `${VAR}` triggers
// substitution here; bare `$something` passes through verbatim.
func loadDocument(explicit string) (*projectfile.Document, error) {
	if !getExpandEnv {
		if explicit != "" {
			return projectfile.ReadFromPathWithOptions(explicit, readOpts())
		}
		doc, _, err := projectfile.ReadWithOptions(".", readOpts())
		return doc, err
	}
	path := explicit
	if path == "" {
		p, err := projectfile.DetectPath(".")
		if err != nil {
			return nil, err
		}
		path = p
	}
	data, err := os.ReadFile(path) // #nosec G304 -- user-supplied path
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	expanded := expandBracedEnv(string(data))
	raw, err := projectfile.ReadRawFromBytes(path, []byte(expanded))
	if err != nil {
		return nil, err
	}
	return projectfile.FromMap(raw), nil
}

// expandBracedEnv replaces every `${VAR}` occurrence in s with the value
// of VAR in the process environment, or the empty string when unset.
// Bare `$VAR` is left alone — see loadDocument's comment for the rationale
// (avoiding `$schema` collateral damage on YAML/JSON inputs).
func expandBracedEnv(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		if i+1 < len(s) && s[i] == '$' && s[i+1] == '{' {
			end := strings.IndexByte(s[i+2:], '}')
			if end >= 0 {
				name := s[i+2 : i+2+end]
				if v, ok := os.LookupEnv(name); ok {
					b.WriteString(v)
				}
				i = i + 2 + end + 1
				continue
			}
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// coerceLang resolves a localized string map to a single string value.
// A map qualifies when every key is a language code (2–5 lowercase ASCII
// letters with an optional _ separator, e.g. "en", "es_CL") and every
// value is a string. Two rules apply:
//   - If lang is non-empty and present as a key, return that value.
//   - If the map has exactly one key, return that value regardless of lang.
//
// All other inputs pass through unchanged.
func coerceLang(v any, lang string) any {
	m, ok := v.(map[string]any)
	if !ok || !isLocalizedMap(m) {
		return v
	}
	if lang != "" {
		if s, ok := m[lang]; ok {
			return s
		}
	}
	if len(m) == 1 {
		for _, s := range m {
			return s
		}
	}
	return v
}

// isLocalizedMap returns true when m is non-empty and every key is a
// language code and every value is a string.
func isLocalizedMap(m map[string]any) bool {
	if len(m) == 0 {
		return false
	}
	for k, v := range m {
		if !isLangCode(k) {
			return false
		}
		if _, ok := v.(string); !ok {
			return false
		}
	}
	return true
}

// isLangCode returns true for IETF-style language tags in the subset used
// by projectfile: 2–5 lowercase ASCII letters with an optional _ separator
// (e.g. "en", "es", "uk", "es_CL", "uk_UA").
func isLangCode(s string) bool {
	if len(s) < 2 || len(s) > 5 {
		return false
	}
	for _, c := range s {
		if c != '_' && (c < 'a' || c > 'z') {
			return false
		}
	}
	return true
}

// errUsage signals a usage problem so the CLI exits 2 (see Execute's
// usageError branch) while still surfacing a readable error message.
func errUsage(msg string) error {
	return &usageError{msg: msg}
}

type usageError struct{ msg string }

func (e *usageError) Error() string { return e.msg }

func init() {
	getCmd.Flags().StringVar(&getDefault, "default", "",
		"fallback value emitted when the path is absent (exit 0). Detected by the\n"+
			"flag being set, so --default='' is honoured and yields the empty string.\n"+
			"A broken or unreadable projectfile still errors (exit 1), so a non-zero\n"+
			"exit tells an absent field apart from a broken document.")
	getCmd.Flags().BoolVar(&getOrDefault, "or-default", false, "emit the spec-defined default when the path is absent")
	getCmd.Flags().StringVar(&getFormat, "format", "raw", "output format: raw, json, sh, flat")
	getCmd.Flags().BoolVar(&getBatch, "batch", false, "read multiple paths in one invocation")
	getCmd.Flags().BoolVar(&getExists, "exists", false, "exit 0 if path exists, 1 if missing (no stdout)")
	getCmd.Flags().BoolVar(&getPrintPath, "print-path", false, "emit the resolved projectfile path and exit (no field query needed)")
	getCmd.Flags().BoolVar(&getExpandEnv, "expand-env", false,
		"expand ${VAR} (braced form only) against the environment before parsing;\n"+
			"unknown variables become the empty string. Bare $VAR is left alone so the\n"+
			"YAML/JSON `$schema` discriminator survives. Expanded values that contain\n"+
			"unquoted format-control characters (newline, TOML/YAML quote chars) can\n"+
			"break the subsequent parse — caller responsibility.")
	getCmd.Flags().StringVarP(&getPathFile, "path-file", "f", "", "explicit projectfile path (skips detection)")
	getCmd.Flags().StringArrayVar(&getNamedPaths, "path", nil, "named path KEY=ADDR (repeatable, batch mode)")
	getCmd.Flags().StringVar(&getLang, "lang", "", "language to select from localized string maps (e.g. en, es, uk)")
	rootCmd.AddCommand(getCmd)
}
