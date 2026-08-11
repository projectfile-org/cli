// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
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

// readSinks resolves the document once and binds the coordinates both
// subcommands compose against.
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
	sinks, err := sink.Declared(doc)
	if err != nil {
		return nil, sink.Coords{}, err
	}
	if len(sinks) == 0 {
		return nil, sink.Coords{}, fmt.Errorf("%s declares no sinks in %s", path, sink.ExtensionNS)
	}
	genlog.Decision("sink_coords", coords.Basename+":"+coords.Tag, path, strconv.Itoa(len(sinks))+" sinks")
	return sinks, coords, nil
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
	sinkRefCmd.Flags().StringVar(&sinkName, "sink", "",
		"name of the declared sink to compose (required)")
	for _, c := range []*cobra.Command{sinkRefCmd, sinkListCmd} {
		c.Flags().StringVar(&sinkBasename, "basename", "",
			"image path to compose for, overriding this project’s own — e.g. b19/ubuntu/resolute")
		c.Flags().StringVar(&sinkTag, "tag", "",
			"image tag to compose for, overriding this project’s own")
	}
	sinkCmd.AddCommand(sinkRefCmd)
	sinkCmd.AddCommand(sinkListCmd)
	rootCmd.AddCommand(sinkCmd)
}
