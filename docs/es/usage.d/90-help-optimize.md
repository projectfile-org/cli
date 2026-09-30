<!--
SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
SPDX-License-Identifier: MIT
-->

# pf-cli optimize

```console
$ pf-cli optimize --help
Remove values your includes already provide.
Keeps includes, $schema and spec_version.

Usage:
  pf-cli optimize [directory] [flags]

Aliases:
  optimize, opt

Flags:
  -n, --dry-run            list removals without saving
  -h, --help               help for optimize
  -f, --path-file string   explicit projectfile path (skips detection)

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
  pf-cli optimize
  pf-cli optimize --dry-run
```
