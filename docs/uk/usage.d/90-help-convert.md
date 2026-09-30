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

Flags:
      --delete-source   remove the source file after converting
  -f, --force           overwrite the output file if it already exists
  -h, --help            help for convert

Global Flags:
      --colors string        colour output: auto|always|never; also PF_CLI_NO_COLOR=1 (default "auto")
      --fail-on string       abort includes at error|warning (default "error")
      --ignore-user-config   skip $XDG_CONFIG_HOME/projectfile/cli.* loading
      --offline              refuse network; use cache and embedded data
  -q, --quiet                mute info; warnings and errors still print
      --sorted               write YAML keys in sorted order
      --timeout duration     per-attempt network timeout (default 10s)
  -v, --verbose              show each step; also PF_CLI_VERBOSE=1

Examples:
  pf-cli convert yaml toml
  pf-cli convert yaml json --delete-source
  pf-cli convert toml yaml --force
```
