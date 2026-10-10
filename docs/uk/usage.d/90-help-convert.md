<!--
SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
SPDX-License-Identifier: MIT
-->

# pf-cli convert

```console
$ pf-cli convert --help
Rewrite the projectfile from one format to another.
Checks the result before writing; keeps the source file.

Usage:
  pf-cli convert <from-format> <to-format> [directory] [flags]

Aliases:
  convert, conv

Examples:
  pf-cli convert yaml toml
  pf-cli convert yaml json --delete-source
  pf-cli convert toml yaml --force

Common flags:
    -q, --quiet          mute info; warnings and errors still print
  -v, --verbose        show each step; also PF_CLI_VERBOSE=1
  --offline            refuse network; use cache and embedded data
  -f, --force          overwrite the output file if it already exists

Flags:
      --delete-source   remove the source file after converting
  -f, --force           overwrite the output file if it already exists
  -h, --help            help for convert

Global Flags:
  …

Documentation:
  https://github.com/projectfile-org/cli/blob/main/docs/USAGE.md#pf-cli-convert

Report a bug:
  https://github.com/projectfile-org/cli/issues
```
