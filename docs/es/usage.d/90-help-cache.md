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

Commands:
  purge       Delete cached includes (all, or one URL)
  refresh     Alias for cache warm --force
  status      Show what is cached and where
  warm        Download includes now so later reads work offline

Flags:
  -h, --help   help for cache

Global Flags:
      --colors string        colour output: auto|always|never; also PF_CLI_NO_COLOR=1 (default "auto")
      --fail-on string       abort includes at error|warning (default "error")
      --ignore-user-config   skip $XDG_CONFIG_HOME/projectfile/cli.* loading
      --offline              refuse network; use cache and embedded data
  -q, --quiet                mute info; warnings and errors still print
      --sorted               write YAML keys in sorted order
      --timeout duration     per-attempt network timeout (default 10s)
  -v, --verbose              show each step; also PF_CLI_VERBOSE=1

Use "pf-cli cache [command] --help" for more information about a command.
```
