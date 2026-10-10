<!--
SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
SPDX-License-Identifier: MIT
-->

# pf-cli add

```console
$ pf-cli add --help
Append items to the list at <path>.
Skips items that are already there.

Usage:
  pf-cli add <path> [value…] [flags]

Examples:
  pf-cli add keywords rust wasm
  pf-cli add repositories --field url=https://example.com/r
  pf-cli add keywords rust -n

Common flags:
    -q, --quiet          mute info; warnings and errors still print
  -v, --verbose        show each step; also PF_CLI_VERBOSE=1
  --offline            refuse network; use cache and embedded data
  -n, --dry-run        show the append without saving it
  --file               explicit projectfile path (skips detection)

Flags:
      --allow-duplicate     append even if already present
  -n, --dry-run             show the append without saving it
      --field stringArray   set one item field as k=v (repeatable)
      --file string         explicit projectfile path (skips detection)
  -h, --help                help for add
      --value-json string   the whole item as JSON

Global Flags:
  …

Documentation:
  https://github.com/projectfile-org/cli/blob/main/docs/USAGE.md#pf-cli-add

Report a bug:
  https://github.com/projectfile-org/cli/issues
```
