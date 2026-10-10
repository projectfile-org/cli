<!--
SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
SPDX-License-Identifier: MIT
-->

# pf-cli set

```console
$ pf-cli set --help
Replace the value at <path>.
A bare value is stored as text, exactly as typed.
Use --value-json or --csv for a typed value.
Missing sections are created.

Usage:
  pf-cli set <path> [value] [flags]

Examples:
  pf-cli set license.spdx MIT
  pf-cli set identity.version 1.10
  pf-cli set keywords --csv rust,wasm
  pf-cli set contacts --value-json '{"email":"a@example.com"}'

Common flags:
    -q, --quiet          mute info; warnings and errors still print
  -v, --verbose        show each step; also PF_CLI_VERBOSE=1
  --offline            refuse network; use cache and embedded data
  -n, --dry-run        show the write without saving it
  --file               explicit projectfile path (skips detection)

Flags:
      --create-only         fail if the path already has a value
      --csv string          value as a comma-separated string list
  -n, --dry-run             show the write without saving it
      --file string         explicit projectfile path (skips detection)
  -h, --help                help for set
      --value-json string   value as JSON (a bare value is text)

Global Flags:
  …

Documentation:
  https://github.com/projectfile-org/cli/blob/main/docs/USAGE.md#pf-cli-set

Report a bug:
  https://github.com/projectfile-org/cli/issues
```
