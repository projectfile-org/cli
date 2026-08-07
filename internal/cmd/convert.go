// SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"kiota.ch/projectfile/core/v2/pkg/genlog"
	"kiota.ch/projectfile/core/v2/pkg/pflock"
	"kiota.ch/projectfile/core/v2/pkg/projectfile"
	"projectfile.org/projectfile/cli/internal/validate"
)

const (
	fmtTOML = "toml"
	fmtYAML = "yaml"
	fmtYML  = "yml"
	fmtJSON = "json"
)

var (
	convertForce        bool
	convertDeleteSource bool
)

// formatExt maps every accepted format token to its on-disk extension. "yml"
// is kept separate from "yaml" so `convert yaml yml` honours the user's
// chosen filename spelling on output; encoder-equivalence is enforced in
// runConvert (both extensions land in the same YAML encoder, so converting
// between them would be a no-op rename and we refuse it).
var formatExt = map[string]string{
	fmtTOML: ".toml",
	fmtYAML: ".yaml",
	fmtYML:  ".yml",
	fmtJSON: ".json",
}

// encoderFor folds the two YAML extension spellings together so the
// same-encoding check treats `yaml` and `yml` as equivalent.
func encoderFor(token string) string {
	if token == fmtYML {
		return fmtYAML
	}
	return token
}

var convertCmd = &cobra.Command{
	Use:   "convert <from-format> <to-format> [directory]",
	Short: "Convert a projectfile between encodings (toml/yaml/json)",
	Long: "Read projectfile.<from-format> from [directory] (default: current\n" +
		"directory), validate it against the v1 schema, write\n" +
		"projectfile.<to-format>, and re-validate the result. Supported\n" +
		"formats: toml, yaml (yml), json. The source file is kept by\n" +
		"default; pass --delete-source to remove it after a successful\n" +
		"write. Refuses to overwrite an existing output file unless --force.",
	Aliases: []string{"conv"},
	Args:    cobra.RangeArgs(2, 3),
	RunE:    runConvert,
}

func runConvert(_ *cobra.Command, args []string) error {
	fromTok := strings.ToLower(args[0])
	toTok := strings.ToLower(args[1])
	dir := "."
	if len(args) == 3 {
		dir = args[2]
	}

	srcExt, ok := formatExt[fromTok]
	if !ok {
		return fmt.Errorf("unknown source format %q (supported: %s)", fromTok, supportedFormats())
	}
	dstExt, ok := formatExt[toTok]
	if !ok {
		return fmt.Errorf("unknown target format %q (supported: %s)", toTok, supportedFormats())
	}

	// Same-encoder shortcut: yaml↔yml would just rename the file via two
	// (de)serialisations. Refuse so users don't conflate convert with mv.
	if encoderFor(fromTok) == encoderFor(toTok) {
		return fmt.Errorf("source and target formats use the same encoder (%s); nothing to convert", encoderFor(fromTok))
	}

	srcPath := filepath.Join(dir, projectfile.BaseName+srcExt)
	dstPath := filepath.Join(dir, projectfile.BaseName+dstExt)

	// We do NOT fall back to DetectPath here: the user explicitly named the
	// source format, and silently picking a sibling encoding would be
	// astonishing (and could pick the wrong file if multiple coexist).
	if _, err := os.Stat(srcPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("source not found at %s", srcPath)
		}
		return fmt.Errorf("stat source %s: %w", srcPath, err)
	}

	if !convertForce {
		if _, err := os.Stat(dstPath); err == nil {
			return fmt.Errorf("output file %s already exists (use --force to overwrite)", dstPath)
		}
	}

	genlog.Plain(fmt.Sprintf("convert: reading %s", srcPath))

	// Read the BASE document (no include resolution).  Converting must
	// preserve the `includes` list as-is so the new encoding can still
	// resolve them — materialising include data into the output would
	// defeat the purpose of includes.
	raw, err := projectfile.ReadRawBaseFromPath(srcPath)
	if err != nil {
		return err
	}
	if err := validate.Validate(raw); err != nil {
		return fmt.Errorf("input %s failed schema validation: %w", srcPath, err)
	}
	genlog.Plain(fmt.Sprintf("convert: input valid (%s)", srcPath))

	doc, err := projectfile.ReadBaseFromPath(srcPath)
	if err != nil {
		return fmt.Errorf("parse %s: %w", srcPath, err)
	}

	return pflock.WithLock(dstPath, func() error {
		if err := projectfile.WriteClean(doc, dstPath); err != nil {
			return fmt.Errorf("write %s: %w", dstPath, err)
		}
		genlog.Plain(fmt.Sprintf("convert: wrote %s", dstPath))

		outRaw, err := projectfile.ReadRawBaseFromPath(dstPath)
		if err != nil {
			return fmt.Errorf("re-read %s: %w", dstPath, err)
		}
		if err := validate.Validate(outRaw); err != nil {
			return fmt.Errorf("output %s failed schema validation after write: %w", dstPath, err)
		}
		genlog.Plain(fmt.Sprintf("convert: output valid (%s)", dstPath))

		if convertDeleteSource {
			if err := os.Remove(srcPath); err != nil {
				return fmt.Errorf("remove source %s: %w", srcPath, err)
			}
			genlog.Plain(fmt.Sprintf("convert: removed %s", srcPath))
		}

		return nil
	})
}

func supportedFormats() string {
	out := make([]string, 0, len(formatExt))
	for k := range formatExt {
		out = append(out, k)
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}

func init() {
	convertCmd.Flags().BoolVarP(&convertForce, "force", "f", false,
		"overwrite the output file if it already exists")
	convertCmd.Flags().BoolVar(&convertDeleteSource, "delete-source", false,
		"delete the source projectfile after a successful conversion")
	rootCmd.AddCommand(convertCmd)
}
