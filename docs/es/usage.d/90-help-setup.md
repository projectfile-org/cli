<!--
SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
SPDX-License-Identifier: MIT
-->

# pf-cli setup

```console
$ pf-cli setup --help
Answer questions once; answers prefill future prompts.
Enter accepts, Esc skips a section.

Usage:
  pf-cli setup [flags]

Flags:
  -f, --format string   output format: yaml, toml, json (default: prompt)
  -h, --help            help for setup

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
  pf-cli setup
  pf-cli setup --format yaml
```
