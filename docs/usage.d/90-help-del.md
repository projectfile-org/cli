<!--
SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
SPDX-License-Identifier: MIT
-->

# pf-cli del

```console
$ pf-cli del --help
Delete the value at <path>.
Missing paths are not an error.

Usage:
  pf-cli del <path> [flags]

Aliases:
  del, delete

Flags:
  -n, --dry-run            show the deletion without saving it
  -h, --help               help for del
  -f, --path-file string   explicit projectfile path (skips detection)
      --strict             exit 1 when the path is already absent

Global Flags:
  …

Examples:
  pf-cli del keywords[0]
  pf-cli del keywords[99] --strict
```
