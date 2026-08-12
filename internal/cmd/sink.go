// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"kiota.ch/projectfile/core/v2/pkg/genlog"
	"kiota.ch/projectfile/core/v2/pkg/projectfile"
	"kiota.ch/projectfile/core/v2/pkg/sink"
)

// Flags shared by both subcommands. The two coordinate overrides are what make
// this verb usable for a FOREIGN image: a build resolving its base image knows
// the other project's path, not its own.
var (
	sinkName     string
	sinkBasename string
	sinkTag      string
)

var sinkCmd = &cobra.Command{
	Use:   "sink",
	Short: "Compose the artifact references of the declared publish destinations",
	Long: "A SINK is a named place an artifact goes, declared under\n" +
		"org.projectfile.sinks. The name is a label: two sinks may address one\n" +
		"registry under two accounts, and nothing in the model can tell them apart.\n" +
		"\n" +
		"What a sink carries is a `ref` template — that template, not any rule in\n" +
		"pf-cli, decides the shape of the reference. It is expanded against the\n" +
		"sink’s own keys (${sink.owner}, ${sink.host}, any key you declare) and the\n" +
		"image coordinates (${image.basename}, ${image.flatname}, ${image.root},\n" +
		"${image.path}, ${image.flatpath}, ${image.namespace}, ${image.name},\n" +
		"${image.tag}). A matrix placeholder such as {B19_UBUNTU_SERIES} carries no\n" +
		"$ and survives verbatim, for the layer that substitutes it per cell.\n" +
		"\n" +
		"This verb exists so a shell never templates a reference itself. The build\n" +
		"plane, the README and the CI lowering all read the SAME composition, or a\n" +
		"project documents a path its build never pushed to.",
	Aliases: []string{"sinks"},
}

var sinkRefCmd = &cobra.Command{
	Use:   "ref [directory]",
	Short: "Compose one sink’s artifact reference",
	Long: "Compose the artifact reference of one declared sink and print it.\n" +
		"\n" +
		"Coordinates default to this project’s own image (the synthetic\n" +
		"image.basename, and image.tag from ci.tag / the :tag on ci.image /\n" +
		"latest). Pass --basename and --tag to compose a reference for ANOTHER\n" +
		"project’s image — which is what resolving a base image needs.\n" +
		"\n" +
		"A template that still carries an unresolved ${…} after expansion is\n" +
		"REFUSED (exit 1), never printed: a reference that silently lost a segment\n" +
		"is a push to the wrong repository. A reference naming several values\n" +
		"prints one line per value.\n" +
		"\n" +
		"Examples:\n" +
		"  pf-cli sink ref --sink ghcr\n" +
		"  pf-cli sink ref --sink kiota --basename b19/ubuntu/resolute --tag dev",
	Args: cobra.MaximumNArgs(1),
	RunE: runSinkRef,
}

var sinkListCmd = &cobra.Command{
	Use:   "list [directory]",
	Short: "List the declared sinks with their composed references",
	Long: "Print one line per declared sink, ranked the way the README ranks them\n" +
		"(priority descending, name as tiebreak):\n" +
		"\n" +
		"  <name>\\t<role>\\t<priority>\\t<composed ref>\n" +
		"\n" +
		"A sink whose reference does not compose is reported as a warning and left\n" +
		"out, so every line printed is a reference something can actually push to.",
	Args: cobra.MaximumNArgs(1),
	RunE: runSinkList,
}

var sinkRefsCmd = &cobra.Command{
	Use:   "refs [directory]",
	Short: "Compose one sink’s reference for many images, in one call",
	Long: "Read image coordinates from stdin, one per line, and print the\n" +
		"reference the named sink composes for each:\n" +
		"\n" +
		"  in    <basename>[\\t<tag>]\n" +
		"  out   <basename>\\t<tag>\\t<ref>\n" +
		"\n" +
		"The coordinates are echoed back, so a caller holding many images maps\n" +
		"each reference to the variable it came from without counting lines. A\n" +
		"reference naming several values prints one row per value, every row\n" +
		"carrying the same input pair. A line with no tag takes --tag; with\n" +
		"neither, the line is refused rather than tagged by guess.\n" +
		"\n" +
		"This is `ref` in bulk, and the build plane is why it exists: a base\n" +
		"image and every tool run-image resolve through ONE process, so reaching\n" +
		"a sink whose template cannot be expressed as a path prefix costs one\n" +
		"spawn rather than one per image.\n" +
		"\n" +
		"Example:\n" +
		"  printf 'b19/ubuntu/resolute\\tdev\\nd9t/go-tools\\tdev\\n' |\n" +
		"    pf-cli sink refs --sink ghcr",
	Args: cobra.MaximumNArgs(1),
	RunE: runSinkRefs,
}

// runSinkRef composes the one named sink.
func runSinkRef(_ *cobra.Command, args []string) error {
	if sinkName == "" {
		return errUsage("sink ref requires --sink <name>")
	}
	sinks, coords, err := readSinks(args)
	if err != nil {
		return err
	}
	s, found := sink.ByName(sinks, sinkName)
	if !found {
		return fmt.Errorf("no sink named %q in %s (declared: %s)",
			sinkName, sink.ExtensionNS, strings.Join(sinkNames(sinks), ", "))
	}
	refs, ok := s.ComposeFanOut(coords)
	if !ok {
		return fmt.Errorf("sink %q: template %q left an unresolved reference for image %q — refusing a partial ref",
			sinkName, s.Template(), coords.Basename)
	}
	for _, ref := range refs {
		fmt.Println(ref)
	}
	return nil
}

// runSinkList composes every declared sink.
func runSinkList(_ *cobra.Command, args []string) error {
	sinks, coords, err := readSinks(args)
	if err != nil {
		return err
	}
	for _, s := range sinks {
		refs, ok := s.ComposeFanOut(coords)
		if !ok {
			genlog.Warn("sink ref did not compose — omitted",
				"sink", s.Name, "template", s.Template(), "basename", coords.Basename)
			continue
		}
		for _, ref := range refs {
			fmt.Printf("%s\t%s\t%d\t%s\n", s.Name, s.Role(), s.Priority(), ref)
		}
	}
	return nil
}

// runSinkRefs composes the one named sink for every coordinate pair on stdin.
//
// The document is read ONCE and the coordinates come from the caller, which is
// the whole point: the make plane resolves a base image and a dozen tool
// run-images against a foreign path each, and paying a process for each of them
// on every parse is what this verb removes.
func runSinkRefs(cmd *cobra.Command, args []string) error {
	if sinkName == "" {
		return errUsage("sink refs requires --sink <name>")
	}
	sinks, path, err := readSinkTable(args)
	if err != nil {
		return err
	}
	s, found := sink.ByName(sinks, sinkName)
	if !found {
		return fmt.Errorf("no sink named %q in %s (declared: %s)",
			sinkName, sink.ExtensionNS, strings.Join(sinkNames(sinks), ", "))
	}

	out := cmd.OutOrStdout()
	scanner := bufio.NewScanner(cmd.InOrStdin())
	pairs := 0
	for scanner.Scan() {
		coords, ok, err := coordsFromLine(scanner.Text())
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		refs, composed := s.ComposeFanOut(coords)
		if !composed {
			return fmt.Errorf("sink %q: template %q left an unresolved reference for image %q — refusing a partial ref",
				sinkName, s.Template(), coords.Basename)
		}
		pairs++
		for _, ref := range refs {
			fmt.Fprintf(out, "%s\t%s\t%s\n", coords.Basename, coords.Tag, ref)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading coordinates for sink %q: %w", sinkName, err)
	}
	genlog.Decision("sink_refs", sinkName, path, strconv.Itoa(pairs)+" coordinates")
	return nil
}

// coordsFromLine splits one stdin row into coordinates.
//
// A blank line is SKIPPED — a caller assembling the list from make variables
// emits one naturally when a slot is empty. A tagless line is REFUSED unless
// --tag supplies one: guessing a tag here would compose a reference to a
// different image than the caller holds, which is the failure the whole verb
// exists to prevent. The default lives in core's ImageTag, not in a second
// copy of the word `latest`.
func coordsFromLine(line string) (sink.Coords, bool, error) {
	basename, tag, split := strings.Cut(strings.TrimSpace(line), "\t")
	basename = strings.TrimSpace(basename)
	if basename == "" {
		return sink.Coords{}, false, nil
	}
	if !split {
		tag = sinkTag
	}
	if tag = strings.TrimSpace(tag); tag == "" {
		return sink.Coords{}, false, fmt.Errorf(
			"image %q carries no tag and --tag names none — refusing to guess one", basename)
	}
	return sink.Coords{Basename: basename, Tag: tag}, true, nil
}

// readSinks resolves the document once and binds the coordinates the ref and
// list subcommands compose against.
//
// The basename is REQUIRED unless overridden: a project with no identity name
// and no ci.image publishes no image, and composing from an empty path would
// emit a reference with a hole where the repository belongs.
func readSinks(args []string) ([]sink.Sink, sink.Coords, error) {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}
	doc, path, err := projectfile.ReadWithOptions(dir, readOpts())
	if err != nil {
		return nil, sink.Coords{}, err
	}
	coords, err := sinkCoords(doc, path)
	if err != nil {
		return nil, sink.Coords{}, err
	}
	sinks, err := declaredSinks(doc, path)
	if err != nil {
		return nil, sink.Coords{}, err
	}
	genlog.Decision("sink_coords", coords.Basename+":"+coords.Tag, path, strconv.Itoa(len(sinks))+" sinks")
	return sinks, coords, nil
}

// readSinkTable resolves the document for a caller that brings its OWN
// coordinates. It deliberately skips sinkCoords: a project resolving a foreign
// image need not name an image of its own, and demanding one would make the
// build plane unreachable from any project that publishes nothing.
func readSinkTable(args []string) ([]sink.Sink, string, error) {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}
	doc, path, err := projectfile.ReadWithOptions(dir, readOpts())
	if err != nil {
		return nil, "", err
	}
	sinks, err := declaredSinks(doc, path)
	if err != nil {
		return nil, "", err
	}
	return sinks, path, nil
}

// declaredSinks is the one place an empty sink table becomes an error, so the
// three subcommands cannot disagree on what "declares no sinks" means.
func declaredSinks(doc *projectfile.Document, path string) ([]sink.Sink, error) {
	sinks, err := sink.Declared(doc)
	if err != nil {
		return nil, err
	}
	if len(sinks) == 0 {
		return nil, fmt.Errorf("%s declares no sinks in %s", path, sink.ExtensionNS)
	}
	return sinks, nil
}

// sinkCoords binds the image half of the template, preferring the flags so the
// same declared sink composes a reference for a foreign project's image.
func sinkCoords(doc *projectfile.Document, path string) (sink.Coords, error) {
	basename := sinkBasename
	if basename == "" {
		resolved, ok := projectfile.ImageBasename(doc)
		if !ok {
			return sink.Coords{}, fmt.Errorf(
				"%s resolves no image basename — declare ci.image or identity.name, or pass --basename", path)
		}
		basename = resolved
	}
	tag := sinkTag
	if tag == "" {
		tag, _ = projectfile.ImageTag(doc)
	}
	return sink.Coords{Basename: basename, Tag: tag}, nil
}

// sinkNames lists the declared names for the "no such sink" message. A typo in a
// registry name is the failure this verb exists to catch loudly: a bare word
// with no dot and no colon is a Docker Hub NAMESPACE, so an unrecognised one
// pulls from Docker Hub in silence rather than failing.
func sinkNames(sinks []sink.Sink) []string {
	out := make([]string, 0, len(sinks))
	for _, s := range sinks {
		out = append(out, s.Name)
	}
	return out
}

func init() {
	for _, c := range []*cobra.Command{sinkRefCmd, sinkRefsCmd} {
		c.Flags().StringVar(&sinkName, "sink", "",
			"name of the declared sink to compose (required)")
	}
	for _, c := range []*cobra.Command{sinkRefCmd, sinkListCmd} {
		c.Flags().StringVar(&sinkBasename, "basename", "",
			"image path to compose for, overriding this project’s own — e.g. b19/ubuntu/resolute")
	}
	for _, c := range []*cobra.Command{sinkRefCmd, sinkListCmd, sinkRefsCmd} {
		c.Flags().StringVar(&sinkTag, "tag", "",
			"image tag to compose for, overriding this project’s own")
	}
	sinkCmd.AddCommand(sinkRefCmd)
	sinkCmd.AddCommand(sinkListCmd)
	sinkCmd.AddCommand(sinkRefsCmd)
	rootCmd.AddCommand(sinkCmd)
}
