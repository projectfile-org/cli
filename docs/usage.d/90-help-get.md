<!--
SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
SPDX-License-Identifier: MIT
-->

# pf-cli get

```console
$ pf-cli get --help
Read values out of the projectfile by path.
Prints plain text, one value per line.
Exits 3 when a path is missing, 2 when the invocation is wrong.

Usage:
  pf-cli get <path>… [flags]

Examples:
  pf-cli get identity.name
  pf-cli get repositories[role=origin].url
  pf-cli get repositories[].url
  pf-cli get identity --format json
  pf-cli get org.projectfile.sinks.kiota.ref --scope org.projectfile.image

Common flags:
    -q, --quiet          mute info; warnings and errors still print
  -v, --verbose        show each step; also PF_CLI_VERBOSE=1
  --offline            refuse network; use cache and embedded data
  --path-file          explicit projectfile path (skips detection)
  --format             raw, json, yaml, toml, sh, flat

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
      --path-file string    explicit projectfile path (skips detection)
      --print-path          print the projectfile path and exit
      --scope stringArray   fill ${…} from this address (repeatable)

Global Flags:
  …

Documentation:
  https://github.com/projectfile-org/cli/blob/main/docs/USAGE.md#pf-cli-get

Report a bug:
  https://github.com/projectfile-org/cli/issues
```
