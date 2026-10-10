<!--
SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
SPDX-License-Identifier: MIT
-->

# pf-cli

```console
$ pf-cli --help
pf-cli is the command-line frontend for the projectfile: it reads, writes and
validates fields, resolves remote includes, and generates derived artifacts. It
embeds the v1 JSON Schema for offline validation and is the canonical tool for
editing projectfiles locally and in CI.

File and forge sync (bridge, forge, scan): use pf-bridge.

Usage:
  pf-cli [command]

Examples:
  pf-cli init --namespace org.example --name demo
  pf-cli get identity.name
  pf-cli set license.spdx MIT
  pf-cli add keywords rust wasm
  pf-cli del keywords[0]
  pf-cli validate
  pf-cli convert yaml toml
  pf-cli optimize
  pf-cli cache status

Read:
  get         Read one or more projectfile fields

Write:
  add         Append items to a projectfile list
  del         Remove a field, list item, or map entry from projectfile
  set         Write a value into a projectfile field

Validate:
  validate    Validate a projectfile against the v1 JSON Schema

Maintain:
  cache       Manage pf-cli’s local cache for offline use
  convert     Convert a projectfile between encodings (toml/yaml/json)
  includes    Manage the includes list
  init        Scaffold a new projectfile document
  optimize    Remove local fields that duplicate include values
  setup       Edit your per-user configuration interactively

Other:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command

Common flags:
    -q, --quiet          mute info; warnings and errors still print
  -v, --verbose        show each step; also PF_CLI_VERBOSE=1
  --offline            refuse network; use cache and embedded data

Flags:
      --colors string        colour output: auto|always|never; also PF_CLI_NO_COLOR=1 (default "auto")
      --fail-on string       abort includes at error|warning (default "error")
  -h, --help                 help for pf-cli
      --ignore-user-config   skip $XDG_CONFIG_HOME/projectfile/cli.* loading
      --offline              refuse network; use cache and embedded data
  -q, --quiet                mute info; warnings and errors still print
      --sorted               write YAML keys in sorted order
      --timeout duration     per-attempt network timeout (default 10s)
  -v, --verbose              show each step; also PF_CLI_VERBOSE=1
  -V, --version              print the version

Documentation:
  https://github.com/projectfile-org/cli/tree/main/docs

Report a bug:
  https://github.com/projectfile-org/cli/issues

Use "pf-cli [command] --help" for more information about a command.
```
