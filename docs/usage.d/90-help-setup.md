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

Examples:
  pf-cli setup
  pf-cli setup --format yaml

Common flags:
    -q, --quiet          mute info; warnings and errors still print
  -v, --verbose        show each step; also PF_CLI_VERBOSE=1
  --offline            refuse network; use cache and embedded data
  --format             output format: yaml, toml, json (default: prompt)

Flags:
      --format string   output format: yaml, toml, json (default: prompt)
  -h, --help            help for setup

Global Flags:
  …

Documentation:
  https://github.com/projectfile-org/cli/blob/main/docs/USAGE.md#pf-cli-setup

Report a bug:
  https://github.com/projectfile-org/cli/issues
```
