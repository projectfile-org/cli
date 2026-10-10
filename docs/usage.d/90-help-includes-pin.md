<!--
SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
SPDX-License-Identifier: MIT
-->

# pf-cli includes pin

```console
$ pf-cli includes pin --help
Rewrite each HTTP(S) include as {url, sha256}, fetching it fresh.
Readers then reject the include once its bytes change.
With no URL every remote include is pinned; a pinned one is re-pinned.

Usage:
  pf-cli includes pin [url…] [flags]

Examples:
  pf-cli includes pin
  pf-cli includes pin https://example.org/base.yaml --dry-run

Common flags:
    -q, --quiet          mute info; warnings and errors still print
  -v, --verbose        show each step; also PF_CLI_VERBOSE=1
  --offline            refuse network; use cache and embedded data
  -n, --dry-run        print the digests without saving
  --file               explicit projectfile path (skips detection)

Flags:
  -n, --dry-run       print the digests without saving
      --file string   explicit projectfile path (skips detection)
  -h, --help          help for pin

Global Flags:
  …

Documentation:
  https://github.com/projectfile-org/cli/blob/main/docs/USAGE.md#pf-cli-pin

Report a bug:
  https://github.com/projectfile-org/cli/issues
```
