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

Examples:
  pf-cli optimize
  pf-cli optimize --dry-run

Common flags:
    -q, --quiet          mute info; warnings and errors still print
  -v, --verbose        show each step; also PF_CLI_VERBOSE=1
  --offline            refuse network; use cache and embedded data
  -n, --dry-run        list removals without saving
  --file               explicit projectfile path (skips detection)

Flags:
  -n, --dry-run       list removals without saving
      --file string   explicit projectfile path (skips detection)
  -h, --help          help for optimize

Global Flags:
  …

Documentation:
  https://github.com/projectfile-org/cli/blob/main/docs/USAGE.md#pf-cli-optimize

Report a bug:
  https://github.com/projectfile-org/cli/issues
```
