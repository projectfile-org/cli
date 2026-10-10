<!--
SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
SPDX-License-Identifier: MIT
-->

# pf-cli cache

```console
$ pf-cli cache --help
Keep downloaded includes on disk for offline reads.
Shared with pf-bridge and pf-ci.

Usage:
  pf-cli cache [command]

Other:
  purge       Delete cached includes (all, or one URL)
  refresh     Alias for cache warm --force
  status      Show what is cached and where
  warm        Download includes now so later reads work offline

Common flags:
  …

Flags:
  -h, --help   help for cache

Global Flags:
  …

Documentation:
  https://github.com/projectfile-org/cli/blob/main/docs/USAGE.md#pf-cli-cache

Report a bug:
  https://github.com/projectfile-org/cli/issues

Use "pf-cli cache [command] --help" for more information about a command.
```
