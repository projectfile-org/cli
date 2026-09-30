<!--
SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
SPDX-License-Identifier: MIT
-->

[Español](es/USAGE.md) · [Українська](uk/USAGE.md)

# Usage

## pf-cli

```console
$ pf-cli --help
pf-cli is the command-line frontend for the projectfile: it reads, writes and
validates fields, resolves remote includes, and generates derived artifacts. It
embeds the v1 JSON Schema for offline validation and is the canonical tool for
editing projectfiles locally and in CI.

File and forge sync (bridge, forge, scan): use pf-bridge.

Usage:
  pf-cli [command]

Commands:
  add         Append items to a projectfile list
  cache       Manage pf-cli’s local cache for offline use
  completion  Generate the autocompletion script for the specified shell
  convert     Convert a projectfile between encodings (toml/yaml/json)
  del         Remove a field, list item, or map entry from projectfile
  get         Read one or more projectfile fields
  help        Help about any command
  init        Scaffold a new projectfile document
  optimize    Remove local fields that duplicate include values
  set         Write a value into a projectfile field
  setup       Edit your per-user configuration interactively
  validate    Validate a projectfile against the v1 JSON Schema

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

Use "pf-cli [command] --help" for more information about a command.
```

## Create, edit and validate a projectfile

```console
$ pf-cli init --namespace org.example --name demo --license MIT --format yaml --non-interactive --no-scan
✓ created projectfile.yaml (3 fields from defaults)
$ pf-cli set identity.summary 'Demo project'
✓ set identity.summary = Demo project
$ pf-cli add keywords cli yaml
✓ add keywords: 2 added, 0 skipped
$ pf-cli get --batch identity.name identity.summary
identity.name=demo
identity.summary=Demo project
$ pf-cli get identity --format json
{
  "name": "demo",
  "namespace": "org.example",
  "summary": "Demo project"
}
$ pf-cli validate
/tmp/demo/projectfile.yaml: valid
```

## pf-cli add

```console
$ pf-cli add --help
Append items to the list at <path>.
Skips items that are already there.

Usage:
  pf-cli add <path> [value…] [flags]

Flags:
      --allow-duplicate     append even if already present
  -n, --dry-run             show the append without saving it
      --field stringArray   set one item field as k=v (repeatable)
  -h, --help                help for add
  -f, --path-file string    explicit projectfile path (skips detection)
      --value-json string   the whole item as JSON

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
  pf-cli add keywords rust wasm
  pf-cli add repositories --field url=https://example.com/r
  pf-cli add keywords rust -n
```

## pf-cli cache

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

## pf-cli cache purge

```console
$ pf-cli cache purge --help
Clear the cache, or one URL. The next read downloads it again.

Usage:
  pf-cli cache purge [url] [flags]

Flags:
  -h, --help   help for purge

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
  pf-cli cache purge
  pf-cli cache purge https://example.com/fleet.yaml
```

## pf-cli cache refresh

```console
$ pf-cli cache refresh --help
Alias for cache warm --force

Usage:
  pf-cli cache refresh [directory] [flags]

Flags:
  -h, --help   help for refresh

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
  pf-cli cache refresh
```

## pf-cli cache status

```console
$ pf-cli cache status --help
Show what is cached and where

Usage:
  pf-cli cache status [flags]

Flags:
  -h, --help   help for status

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
  pf-cli cache status
```

## pf-cli cache warm

```console
$ pf-cli cache warm --help
Fetch every include for [directory] into the cache.

Usage:
  pf-cli cache warm [directory] [flags]

Flags:
      --force   download everything again, even when fresh
  -h, --help    help for warm

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
  pf-cli cache warm
  pf-cli cache warm /tmp/demo --force
```

## pf-cli completion

```console
$ pf-cli completion --help
Generate the autocompletion script for pf-cli for the specified shell.
See each sub-command's help for details on how to use the generated script.

Usage:
  pf-cli completion [command]

Commands:
  bash        Generate the autocompletion script for bash
  fish        Generate the autocompletion script for fish
  powershell  Generate the autocompletion script for powershell
  zsh         Generate the autocompletion script for zsh

Flags:
  -h, --help   help for completion

Global Flags:
      --colors string        colour output: auto|always|never; also PF_CLI_NO_COLOR=1 (default "auto")
      --fail-on string       abort includes at error|warning (default "error")
      --ignore-user-config   skip $XDG_CONFIG_HOME/projectfile/cli.* loading
      --offline              refuse network; use cache and embedded data
  -q, --quiet                mute info; warnings and errors still print
      --sorted               write YAML keys in sorted order
      --timeout duration     per-attempt network timeout (default 10s)
  -v, --verbose              show each step; also PF_CLI_VERBOSE=1

Use "pf-cli completion [command] --help" for more information about a command.
```

## pf-cli convert

```console
$ pf-cli convert --help
Rewrite the projectfile from one format to another.
Checks the result before writing; keeps the source file.

Usage:
  pf-cli convert <from-format> <to-format> [directory] [flags]

Aliases:
  convert, conv

Flags:
      --delete-source   remove the source file after converting
  -f, --force           overwrite the output file if it already exists
  -h, --help            help for convert

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
  pf-cli convert yaml toml
  pf-cli convert yaml json --delete-source
  pf-cli convert toml yaml --force
```

## pf-cli del

```console
$ pf-cli del --help
Delete the value at <path>.
Missing paths are not an error.

Usage:
  pf-cli del <path> [flags]

Aliases:
  del, delete

Flags:
  -n, --dry-run            show the deletion without saving it
  -h, --help               help for del
  -f, --path-file string   explicit projectfile path (skips detection)
      --strict             exit 1 when the path is already absent

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
  pf-cli del keywords[0]
  pf-cli del keywords[99] --strict
```

## pf-cli get

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

## pf-cli init

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

## pf-cli optimize

```console
$ pf-cli optimize --help
Remove values your includes already provide.
Keeps includes, $schema and spec_version.

Usage:
  pf-cli optimize [directory] [flags]

Aliases:
  optimize, opt

Flags:
  -n, --dry-run            list removals without saving
  -h, --help               help for optimize
  -f, --path-file string   explicit projectfile path (skips detection)

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
  pf-cli optimize
  pf-cli optimize --dry-run
```

## pf-cli set

```console
$ pf-cli set --help
Replace the value at <path>.
Missing sections are created.

Usage:
  pf-cli set <path> [value] [flags]

Flags:
      --create-only         fail if the path already has a value
      --csv string          value as a comma-separated string list
  -n, --dry-run             show the write without saving it
  -h, --help                help for set
  -f, --path-file string    explicit projectfile path (skips detection)
      --value-json string   value as JSON (needed for objects and arrays)

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
  pf-cli set license.spdx MIT
  pf-cli set keywords --csv rust,wasm
  pf-cli set contacts --value-json '{"email":"a@example.com"}'
```

## pf-cli setup

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

## pf-cli validate

```console
$ pf-cli validate --help
Check that the projectfile is well-formed.
Warns about repeated includes.

Usage:
  pf-cli validate [directory] [flags]

Aliases:
  validate, lint

Flags:
  -h, --help              help for validate
      --strict-includes   fail on repeated includes (default: warn)

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
  pf-cli validate
  pf-cli validate /tmp/demo
  pf-cli validate --strict-includes
```
