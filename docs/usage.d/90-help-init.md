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

Flags:
  -f, --format string      output format: yaml, toml, json (default: prompt)
  -h, --help               help for init
      --license string     license SPDX expression (e.g. MIT)
      --name string        identity.name (project slug)
      --namespace string   identity.namespace (reverse-DNS, e.g. org.example)
      --no-scan            skip init-time scanners (accepted for pf-bridge-init parity; the basic scaffold never scans)
      --non-interactive    fail if required fields are missing instead of prompting

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
  pf-cli init
  pf-cli init /tmp/demo --namespace org.example --name demo --non-interactive
```
