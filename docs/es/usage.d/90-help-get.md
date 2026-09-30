<!--
SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
SPDX-License-Identifier: MIT
-->

# pf-cli get

```console
$ pf-cli get --help
Read values out of the projectfile by path.
Prints plain text, one value per line.
Exits 1 when a path is missing.

Usage:
  pf-cli get <path>… [flags]

Flags:
      --batch               read several paths in one run
      --default string      value to print when the path is missing
      --exists              exit 0 if present, 1 if absent; prints nothing
      --expand              fill ${…} from the document; leave the rest as written
      --expand-env          fill ${VAR} from the environment first
      --format string       raw, json, yaml, toml, sh, flat (default "raw")
  -h, --help                help for get
      --lang string         choose a language for translated fields
      --or-default          use the built-in default if missing
      --path stringArray    label a path as KEY=ADDR (repeatable)
  -f, --path-file string    explicit projectfile path (skips detection)
      --print-path          print the projectfile path and exit
      --scope stringArray   fill ${…} from this address (repeatable)

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
  pf-cli get identity.name
  pf-cli get repositories[role=origin].url
  pf-cli get repositories[].url
  pf-cli get identity --format json
  pf-cli get org.projectfile.sinks.kiota.ref --scope org.projectfile.image
```
