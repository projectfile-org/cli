<!--
SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
SPDX-License-Identifier: MIT
-->

# pf-cli cache warm

```console
$ pf-cli cache warm --help
Fetch every include for [directory] into the cache.

Usage:
  pf-cli cache warm [directory] [flags]

Examples:
  pf-cli cache warm
  pf-cli cache warm /tmp/demo --force

Common flags:
    -q, --quiet          mute info; warnings and errors still print
  -v, --verbose        show each step; also PF_CLI_VERBOSE=1
  --offline            refuse network; use cache and embedded data
  -f, --force          download everything again, even when fresh

Flags:
  -f, --force   download everything again, even when fresh
  -h, --help    help for warm

Global Flags:
  …

Documentation:
  https://github.com/projectfile-org/cli/blob/main/docs/USAGE.md#pf-cli-warm

Report a bug:
  https://github.com/projectfile-org/cli/issues
```
