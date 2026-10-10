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
      --path-file string    explicit projectfile path (skips detection)
      --value-json string   value as JSON (needed for objects and arrays)

Global Flags:
  …

Examples:
  pf-cli set license.spdx MIT
  pf-cli set keywords --csv rust,wasm
  pf-cli set contacts --value-json '{"email":"a@example.com"}'
```
