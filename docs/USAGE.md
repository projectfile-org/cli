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
  includes    Manage the includes list
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

## Exit codes

Every non-zero exit is one documented failure class, so a script branches on
the kind of failure rather than treating every non-zero as “broken”. A
command that succeeded exits `0`; any other value means the work did not
finish.

| Code | Meaning | Examples |
| --- | --- | --- |
| `1` | Runtime failure — the document could not be read, an include could not be resolved, or a write was refused | `pf-cli get --path-file /nope.yaml identity.name`, an unreachable include, a failed write |
| `2` | Usage error — the invocation itself is wrong, and nothing ran | `pf-cli --nope`, `pf-cli bogus`, `pf-cli set` with no path |
| `3` | The requested field is absent | `pf-cli get identity.bogus`, `pf-cli get identity.name --exists` on a missing field |
| `4` | Validation failed — the document was read and violates the v1 schema | `pf-cli validate` on a document missing required fields |

```console
$ pf-cli get identity.bogus; echo $?
identity.bogus: no such field. To continue, list what is there with pf-cli get identity --format yaml, or pass --default <value> to fall back.
3
$ pf-cli bogus; echo $?
Error: unknown command "bogus" for "pf-cli". Run pf-cli --help to see every command.
2
```

Codes `2`, `3` and `4` were all `1` before this map existed. A caller that
branches on “non-zero versus zero” is unaffected; one that branched on `1`
alone now sees `3` for a missing field and `4` for a schema failure.

## Tab completion

`pf-cli completion` prints the shell-completion script for the shell named as
its argument — `bash`, `zsh`, `fish` or `powershell`. Load it once per shell
session, or from your shell’s startup file:

```sh
source <(pf-cli completion bash)
source <(pf-cli completion zsh)
source <(pf-cli completion fish)
```

Every command, subcommand, flag and flag value is completed, including the
`--format` list on `get`. Print the script to inspect it, or install it under
your shell’s completion directory:

```console
$ pf-cli completion zsh --help
Generate the autocompletion script for the zsh shell.
...
```

## pf-cli add

```console
$ pf-cli add --help
Append items to the list at <path>.
Skips items that are already there.

Usage:
  pf-cli add <path> [value…] [flags]

Examples:
  pf-cli add keywords rust wasm
  pf-cli add repositories --field url=https://example.com/r
  pf-cli add keywords rust -n

Common flags:
    -q, --quiet          mute info; warnings and errors still print
  -v, --verbose        show each step; also PF_CLI_VERBOSE=1
  --offline            refuse network; use cache and embedded data
  -n, --dry-run        show the append without saving it
  --path-file          explicit projectfile path (skips detection)

Flags:
      --allow-duplicate     append even if already present
  -n, --dry-run             show the append without saving it
      --field stringArray   set one item field as k=v (repeatable)
  -h, --help                help for add
      --path-file string    explicit projectfile path (skips detection)
      --value-json string   the whole item as JSON

Global Flags:
  …

Documentation:
  https://github.com/projectfile-org/cli/blob/main/docs/USAGE.md#pf-cli-add

Report a bug:
  https://github.com/projectfile-org/cli/issues
```

## pf-cli cache

```console
$ pf-cli cache --help
Keep downloaded includes on disk for offline reads.
Shared with pf-bridge and pf-ci.

Usage:
  pf-cli cache [command]

Other:
  purge       Delete cached includes (all, or one URL)
  refresh     Alias for cache warm --force
  status      Show what is cached and where
  warm        Download includes now so later reads work offline

Common flags:
  …

Flags:
  -h, --help   help for cache

Global Flags:
  …

Documentation:
  https://github.com/projectfile-org/cli/blob/main/docs/USAGE.md#pf-cli-cache

Report a bug:
  https://github.com/projectfile-org/cli/issues

Use "pf-cli cache [command] --help" for more information about a command.
```

## pf-cli cache purge

```console
$ pf-cli cache purge --help
Clear the cache, or one URL. The next read downloads it again.

Usage:
  pf-cli cache purge [url] [flags]

Examples:
  pf-cli cache purge
  pf-cli cache purge https://example.com/fleet.yaml

Common flags:
  …

Flags:
  -h, --help   help for purge

Global Flags:
  …

Documentation:
  https://github.com/projectfile-org/cli/blob/main/docs/USAGE.md#pf-cli-purge

Report a bug:
  https://github.com/projectfile-org/cli/issues
```

## pf-cli cache refresh

```console
$ pf-cli cache refresh --help
Alias for cache warm --force

Usage:
  pf-cli cache refresh [directory] [flags]

Examples:
  pf-cli cache refresh

Common flags:
  …

Flags:
  -h, --help   help for refresh

Global Flags:
  …

Documentation:
  https://github.com/projectfile-org/cli/blob/main/docs/USAGE.md#pf-cli-refresh

Report a bug:
  https://github.com/projectfile-org/cli/issues
```

## pf-cli cache status

```console
$ pf-cli cache status --help
Show what is cached and where

Usage:
  pf-cli cache status [flags]

Examples:
  pf-cli cache status

Common flags:
  …

Flags:
  -h, --help   help for status

Global Flags:
  …

Documentation:
  https://github.com/projectfile-org/cli/blob/main/docs/USAGE.md#pf-cli-status

Report a bug:
  https://github.com/projectfile-org/cli/issues
```

## pf-cli cache warm

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

## pf-cli completion

```console
$ pf-cli completion --help
Generate the autocompletion script for pf-cli for the specified shell.
See each sub-command's help for details on how to use the generated script.

Usage:
  pf-cli completion [command]

Other:
  bash        Generate the autocompletion script for bash
  fish        Generate the autocompletion script for fish
  powershell  Generate the autocompletion script for powershell
  zsh         Generate the autocompletion script for zsh

Common flags:
  …

Flags:
  -h, --help   help for completion

Global Flags:
  …

Documentation:
  https://github.com/projectfile-org/cli/blob/main/docs/USAGE.md#pf-cli-completion

Report a bug:
  https://github.com/projectfile-org/cli/issues

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

Examples:
  pf-cli convert yaml toml
  pf-cli convert yaml json --delete-source
  pf-cli convert toml yaml --force

Common flags:
    -q, --quiet          mute info; warnings and errors still print
  -v, --verbose        show each step; also PF_CLI_VERBOSE=1
  --offline            refuse network; use cache and embedded data
  -f, --force          overwrite the output file if it already exists

Flags:
      --delete-source   remove the source file after converting
  -f, --force           overwrite the output file if it already exists
  -h, --help            help for convert

Global Flags:
  …

Documentation:
  https://github.com/projectfile-org/cli/blob/main/docs/USAGE.md#pf-cli-convert

Report a bug:
  https://github.com/projectfile-org/cli/issues
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

Examples:
  pf-cli del keywords[0]
  pf-cli del keywords[99] --strict

Common flags:
    -q, --quiet          mute info; warnings and errors still print
  -v, --verbose        show each step; also PF_CLI_VERBOSE=1
  --offline            refuse network; use cache and embedded data
  -n, --dry-run        show the deletion without saving it
  --path-file          explicit projectfile path (skips detection)

Flags:
  -n, --dry-run            show the deletion without saving it
  -h, --help               help for del
      --path-file string   explicit projectfile path (skips detection)
      --strict             exit 1 when the path is already absent

Global Flags:
  …

Documentation:
  https://github.com/projectfile-org/cli/blob/main/docs/USAGE.md#pf-cli-del

Report a bug:
  https://github.com/projectfile-org/cli/issues
```

## pf-cli get

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

## pf-cli includes

```console
$ pf-cli includes --help
Manage the includes list

Usage:
  pf-cli includes [command]

Other:
  pin         Pin remote includes to the SHA-256 of their current bytes

Common flags:
  …

Flags:
  -h, --help   help for includes

Global Flags:
  …

Documentation:
  https://github.com/projectfile-org/cli/blob/main/docs/USAGE.md#pf-cli-includes

Report a bug:
  https://github.com/projectfile-org/cli/issues

Use "pf-cli includes [command] --help" for more information about a command.
```

## pf-cli includes pin

```console
$ pf-cli includes pin --help
Rewrite each HTTP(S) include as {url, sha256}, fetching it fresh.
Readers then reject the include once its bytes change.
With no URL every remote include is pinned; a pinned one is re-pinned.

Usage:
  pf-cli includes pin [url…] [flags]

Examples:
  pf-cli includes pin
  pf-cli includes pin https://example.org/base.yaml --dry-run

Common flags:
    -q, --quiet          mute info; warnings and errors still print
  -v, --verbose        show each step; also PF_CLI_VERBOSE=1
  --offline            refuse network; use cache and embedded data
  -n, --dry-run        print the digests without saving
  --path-file          explicit projectfile path (skips detection)

Flags:
  -n, --dry-run            print the digests without saving
  -h, --help               help for pin
      --path-file string   explicit projectfile path (skips detection)

Global Flags:
  …

Documentation:
  https://github.com/projectfile-org/cli/blob/main/docs/USAGE.md#pf-cli-pin

Report a bug:
  https://github.com/projectfile-org/cli/issues
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

## pf-cli optimize

```console
$ pf-cli optimize --help
Remove values your includes already provide.
Keeps includes, $schema and spec_version.

Usage:
  pf-cli optimize [directory] [flags]

Aliases:
  optimize, opt

Examples:
  pf-cli optimize
  pf-cli optimize --dry-run

Common flags:
    -q, --quiet          mute info; warnings and errors still print
  -v, --verbose        show each step; also PF_CLI_VERBOSE=1
  --offline            refuse network; use cache and embedded data
  -n, --dry-run        list removals without saving
  --path-file          explicit projectfile path (skips detection)

Flags:
  -n, --dry-run            list removals without saving
  -h, --help               help for optimize
      --path-file string   explicit projectfile path (skips detection)

Global Flags:
  …

Documentation:
  https://github.com/projectfile-org/cli/blob/main/docs/USAGE.md#pf-cli-optimize

Report a bug:
  https://github.com/projectfile-org/cli/issues
```

## pf-cli set

```console
$ pf-cli set --help
Replace the value at <path>.
A bare value is stored as text, exactly as typed.
Use --value-json or --csv for a typed value.
Missing sections are created.

Usage:
  pf-cli set <path> [value] [flags]

Examples:
  pf-cli set license.spdx MIT
  pf-cli set identity.version 1.10
  pf-cli set keywords --csv rust,wasm
  pf-cli set contacts --value-json '{"email":"a@example.com"}'

Common flags:
    -q, --quiet          mute info; warnings and errors still print
  -v, --verbose        show each step; also PF_CLI_VERBOSE=1
  --offline            refuse network; use cache and embedded data
  -n, --dry-run        show the write without saving it
  --path-file          explicit projectfile path (skips detection)

Flags:
      --create-only         fail if the path already has a value
      --csv string          value as a comma-separated string list
  -n, --dry-run             show the write without saving it
  -h, --help                help for set
      --path-file string    explicit projectfile path (skips detection)
      --value-json string   value as JSON (a bare value is text)

Global Flags:
  …

Documentation:
  https://github.com/projectfile-org/cli/blob/main/docs/USAGE.md#pf-cli-set

Report a bug:
  https://github.com/projectfile-org/cli/issues
```

## pf-cli setup

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

## pf-cli validate

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

Common flags:
  …

Flags:
  -h, --help              help for validate
      --strict-includes   fail on repeated includes (default: warn)

Global Flags:
  …

Documentation:
  https://github.com/projectfile-org/cli/blob/main/docs/USAGE.md#pf-cli-validate

Report a bug:
  https://github.com/projectfile-org/cli/issues
```
