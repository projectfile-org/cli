<!--
SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
SPDX-License-Identifier: MIT
-->

# pf-cli validate

```console
$ pf-cli validate --help
Check that the projectfile is well-formed.
Warns about repeated includes.

Usage:
  pf-cli validate [directory] [flags]

Aliases:
  validate, lint

Examples:
  pf-cli validate
  pf-cli validate /tmp/demo
  pf-cli validate --strict-includes
  pf-cli validate --fragment .projectfile/release.yaml

Common flags:
  …

Flags:
      --fragment          treat the argument as one included file and check it against the fragment schema
  -h, --help              help for validate
      --strict-includes   fail on repeated includes (default: warn)

Global Flags:
  …

Documentation:
  https://github.com/projectfile-org/cli/blob/main/docs/USAGE.md#pf-cli-validate

Report a bug:
  https://github.com/projectfile-org/cli/issues
```
