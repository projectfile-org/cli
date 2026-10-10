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

Flags:
  -n, --dry-run            print the digests without saving
  -h, --help               help for pin
  -f, --path-file string   explicit projectfile path (skips detection)

Global Flags:
  …

Examples:
  pf-cli includes pin
  pf-cli includes pin https://example.org/base.yaml --dry-run
```
