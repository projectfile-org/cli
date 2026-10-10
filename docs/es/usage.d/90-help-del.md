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

Examples:
  pf-cli del keywords[0]
  pf-cli del keywords[99] --strict

Common flags:
    -q, --quiet          mute info; warnings and errors still print
  -v, --verbose        show each step; also PF_CLI_VERBOSE=1
  --offline            refuse network; use cache and embedded data
  -n, --dry-run        show the deletion without saving it
  --file               explicit projectfile path (skips detection)

Flags:
  -n, --dry-run       show the deletion without saving it
      --file string   explicit projectfile path (skips detection)
  -h, --help          help for del
      --strict        exit 1 when the path is already absent

Global Flags:
  …

Documentation:
  https://github.com/projectfile-org/cli/blob/main/docs/USAGE.md#pf-cli-del

Report a bug:
  https://github.com/projectfile-org/cli/issues
```
