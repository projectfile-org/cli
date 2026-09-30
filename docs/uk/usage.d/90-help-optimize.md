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
  …

Examples:
  pf-cli optimize
  pf-cli optimize --dry-run
```
