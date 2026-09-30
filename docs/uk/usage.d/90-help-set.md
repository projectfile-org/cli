<!--
SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
SPDX-License-Identifier: MIT
-->

# pf-cli set

```console
$ pf-cli set --help
Replace the value at <path>.
Missing sections are created.

Usage:
  pf-cli set <path> [value] [flags]

Flags:
      --create-only         fail if the path already has a value
      --csv string          value as a comma-separated string list
  -n, --dry-run             show the write without saving it
  -h, --help                help for set
  -f, --path-file string    explicit projectfile path (skips detection)
      --value-json string   value as JSON (needed for objects and arrays)

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
  pf-cli set license.spdx MIT
  pf-cli set keywords --csv rust,wasm
  pf-cli set contacts --value-json '{"email":"a@example.com"}'
```
