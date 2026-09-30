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

Flags:
      --allow-duplicate     append even if already present
  -n, --dry-run             show the append without saving it
      --field stringArray   set one item field as k=v (repeatable)
  -h, --help                help for add
  -f, --path-file string    explicit projectfile path (skips detection)
      --value-json string   the whole item as JSON

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
  pf-cli add keywords rust wasm
  pf-cli add repositories --field url=https://example.com/r
  pf-cli add keywords rust -n
```
