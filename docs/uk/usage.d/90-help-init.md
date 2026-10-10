<!--
SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
SPDX-License-Identifier: MIT
-->

# pf-cli init

```console
$ pf-cli init --help
Create a new projectfile.yaml (or .toml/.json) in [directory].
With pf-bridge installed this delegates to pf-bridge-init: full
detection from sources (package.json, CITATION.cff, …) plus scanner
gap-fill (git history, stack detection). Without it, a basic local
scaffold prompts for namespace, name and title only — no scanners.

Usage:
  pf-cli init [directory] [flags]

Aliases:
  init, scaffold

Examples:
  pf-cli init
  pf-cli init /tmp/demo --namespace org.example --name demo --non-interactive

Common flags:
    -q, --quiet          mute info; warnings and errors still print
  -v, --verbose        show each step; also PF_CLI_VERBOSE=1
  --offline            refuse network; use cache and embedded data
  --format             output format: yaml, toml, json (default: prompt)

Flags:
      --format string      output format: yaml, toml, json (default: prompt)
  -h, --help               help for init
      --license string     license SPDX expression (e.g. MIT)
      --name string        identity.name (project slug)
      --namespace string   identity.namespace (reverse-DNS, e.g. org.example)
      --no-scan            skip init-time scanners (accepted for pf-bridge-init parity; the basic scaffold never scans)
      --non-interactive    fail if required fields are missing instead of prompting

Global Flags:
  …

Documentation:
  https://github.com/projectfile-org/cli/blob/main/docs/USAGE.md#pf-cli-init

Report a bug:
  https://github.com/projectfile-org/cli/issues
```
