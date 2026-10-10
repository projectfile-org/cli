<!--
SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>

SPDX-License-Identifier: MIT
-->

# projectfile/cli — agent guide

## Purpose

The **`pf-cli` binary** — the document-backend CLI. A thin Cobra command
layer over the `projectfile/core` **library**, which it consumes through core’s
`pkg/*` façades only (never core `internal/`). One of three peer library
consumers alongside `pf-bridge` (`projectfile/bridge`) and `pf-ci`
(`projectfile/ci`).

> **Bridge Revolution Phase 8 (2026-07-07):** `main.go` +
> `internal/{cmd,validate,usersetup}` were extracted from `projectfile/core`
> into this module so core became a pure library. See
> [../bridge-revolution.md](../bridge-revolution.md) and
> [../core/AGENTS.md](../core/AGENTS.md).

## Commands

`get`/`set`/`add`/`del` (dotted-path query + mutate), `convert` (encoding),
`validate` (v1 JSON Schema), `optimize` (strip include-redundant fields AND redundant includes),
`cache` (warm/purge HTTP **includes** only — SPDX moved to pf-bridge),
`setup` (per-user config), `init` (scaffold a new document — delegates to
`pf-bridge-init` when installed, else a basic local scaffold). Projections
onto external files/forges/the repository (`bridge`/`forge`/`scan`) live in
`pf-bridge`, not here.

### `get --scope` — reading a value the document COMPOSES

`--scope <address>` (repeatable) makes a subtree answer a `${…}` reference
before the document root, and turns expansion ON for the resolved value. It is
the whole mechanism behind composed references, and pf-cli knows no vocabulary
of any domain:

```yaml
org.projectfile.image: {org: b19, name: ${identity.name}, series: resolute, tag: latest}
org.projectfile.sinks:
  kiota: {ref: "kiota.ch/${org}/${name}-${series}:${tag}", priority: 10}
  ghcr:  {ref: "ghcr.io/buho/${org}-${name}-${series}:${tag}", priority: 90}
```

```sh
pf-cli get org.projectfile.sinks.kiota.ref --scope org.projectfile.image
# kiota.ch/b19/ubuntu-resolute:latest

pf-cli get 'org.projectfile.sinks{}.values' --scope org.projectfile.image
# every destination composed, ONE spawn, ranked by priority descending
```

Three properties the make plane depends on:

- **Without `--scope` nothing changes.** `${org}` is not a document address, so
    an unscoped `get` returns the template verbatim and every caller that
    predates the flag keeps its behaviour. It is also why a half-composed
    reference cannot be emitted by accident.
- **Expansion descends into entries.** A map projection hands back whole entry
    maps, and `expandScoped` walks into them — which is what makes "compose every
    destination" one spawn instead of one per entry.
- **`{AXIS}` survives.** A matrix placeholder carries no `$`, so it passes
    through composition and is substituted per cell by the layer that owns the
    matrix. Parts may hold one (`series: "{B19_UBUNTU_SERIES}"`).

A scope is an ADDRESS the caller picks, so the same template composes a foreign
subject by naming that subject’s parts — which is how a base image resolves
without pf-cli knowing what a base image is.

## Layout

```text
projectfile/cli/
├── main.go               entry point → internal/cmd.Execute (Cobra root)
├── go.mod                module projectfile.org/projectfile/cli
│                         require kiota.ch/projectfile/core + replace => ../core
└── internal/
    ├── cmd/              Cobra commands + scope (scoped resolve + `${…}` expansion for get)
    ├── validate/         v1 JSON Schema check (embedded/v1.json) — cli-only
    └── usersetup/        interactive first-run wizard — cli-only
```

`validate` and `usersetup` are cli-only (no other consumer); everything else the
commands need — model, read/write, fieldpath, genlog, spdx, pflock, userconfig,
selector — is imported from `kiota.ch/projectfile/core/pkg/*`.

## The core seam

- Consume core **only** through `core/pkg/*`. A mutable core toggle crosses as a setter/getter, never a value alias (`genlog.SetQuiet`, `projectfile.SetYAML‌ OutputSorted`/`YAMLOutputSortedEnabled`) — a `var X = internal.X` reexport copies, so a write here would not reach core.
- `go.mod` requires `kiota.ch/projectfile/core` as a pinned module with no
    `replace`. Release ordering: tag core → `go get core@<tag>` here. A
    temporary `replace => ../core` is used during development to test core
    changes before a tag is cut.
- SPDX texts are NOT cached or warmed here — pf-cli’s cache is HTTP includes
    only. SPDX boilerplate (corpus + warming) lives in pf-bridge, which is the
    sole reader of `spdx.Text`.

## `git -C` does not isolate a repository — `GIT_DIR` beats it

**Any script that runs Git against a nested clone MUST clear the inherited Git
environment first.** `git -C DIR` changes directory but does NOT override
`GIT_DIR`, and Git exports `GIT_DIR` to every hook. So under lefthook’s
pre-commit, every `git -C _specification …` in `.scripts/fetch-schema.sh`
targeted **cli itself**: `remote set-url origin` repointed cli at
`https://kiota.ch/projectfile/specification.git`, and `reset --hard origin/main`
moved the checkout onto the specification’s history. A push in that state sends
cli’s code to the wrong repository.

What makes it hard to catch: `rev-parse --show-toplevel` still answers `DIR`,
so a toplevel assertion passes while `remote get-url origin` already answers
about the enclosing repository. Assert on `rev-parse --absolute-git-dir`
instead — that is the value the environment overrides.

```sh
unset GIT_DIR GIT_WORK_TREE GIT_INDEX_FILE GIT_PREFIX GIT_COMMON_DIR
```

Reproduce (and prove a fix) without a commit:
`GIT_DIR=$(git rev-parse --absolute-git-dir) sh .scripts/fetch-schema.sh`.

The signature of the damage: `git log` shows specification commits, `git remote
get-url origin` names the spec repository, and every file the local commits
touched reads as modified because HEAD moved behind them. Repair with the URL
cli’s own projectfile declares, then re-point the branch — `--mixed`, never
`--hard`, or the working tree goes with it:

```sh
git remote set-url origin ssh://git@kiota.ch/projectfile/cli.git
git fetch origin --prune
git reset --mixed <the commit the branch should sit on>
```

## Build

The host binary is built by the `build-binaries` manifest tool (matrix-per-cell on
the forge; host-native off-matrix on the make-plane) — there is no hand-written
`build-local`. Local install is the m6e-only `install-binary` tool, which
self-builds via `.scripts/build-binaries.sh` then copies `dist/pf-cli` into
`~/.local/bin` (quick + ungated, as the old `install-local` was).

```sh
make build-binaries   # compile host-native binary into dist/pf-cli (+ suffixed cell)
make install-binary   # build-binaries + copy dist/pf-cli into ~/.local/bin
make help             # every target
```

`-X projectfile.org/projectfile/cli/internal/cmd.version` carries the release
version (a clean tag on the forge, `git describe`/`dev` on the make-plane).
The root help Description is `identity.description.en` from `projectfile.yaml`,
embedded via `main.go` (`//go:embed`, so the `Dockerfile` builder `COPY` must
keep carrying that file). Help headers and command names carry lipgloss
color in the setup-wizard palette, auto-off when piped or under NO_COLOR;
every help line stays within 80 visible columns.
Iterate with `go build ./... && go test ./...`.

### Help rendering

The root `--help` order (Description, Examples, Commands, Common flags,
Flags, Global flags, doc links) and the concise screen a bare invocation
prints both live in `helpTemplate` in `internal/cmd/root.go`. Commands are
grouped read / write / validate / maintain through `groupCmd(cmd, groupID,
page)` — a new command MUST go through that helper or it lands ungrouped and
loses its docs page. The docs and issue URLs are resolved at RENDER time from
`projectfileYAML` (`docLinks`), because `SetProjectfileYAML` runs after package
init, so an eagerly stamped annotation reads empty.

### The YAML emitter cannot fold

`go.yaml.in/yaml/v3` sets `best_width = -1` (unbounded) and exposes no width
knob, so every folded `>-` block scalar comes back from a write as ONE physical
line — the marker survives, the author breaks do not, and a long description
then fails every consumer yamllint (`line-length: 120`). `internal/cmd/write.go`
closes that gap: `writeProjectfile` (the only writer every mutating command uses)
re-folds the block at isolated spaces after the write. Keep the pass TEXT-level
and YAML-only: a value with embedded newlines is re-emitted by the emitter with a
empty line per break, which changes the value. A literal `|` block, a single
unbreakable word, a paragraph break and a `>` inside a plain scalar are left alone.

## Contracts (do NOT rename)

The binary is `pf-cli` (the `pf-*` triad with `pf-ci` + `pf-bridge`); its image
is `projectfile/cli` (from `identity.name: cli`). The on-disk/config/env
contracts share the `cli`/`pf-cli` spelling: the `# pf-cli-managed:` sentinel,
`PF_CLI_VERBOSE`, the `projectfile/cli.toml` user-config path, and the
shared `pf/` XDG cache slot (its includes cache under
`$XDG_CACHE_HOME/pf/`, shared with pf-bridge and pf-ci). Renaming any of
these would orphan existing files/config.

### Exit codes

`Execute` maps every failure onto one class through the table in
`internal/cmd/root.go`: 1 runtime/IO, 2 bad invocation (the cobra unknown-command
and unknown shorthand are re-classed into this one), 3 the requested field is
absent, 4 the document fails the v1 schema. A new failure mode gets its own
number in that table and a row in `docs/usage.d/20-exit-codes.md` — folding it
silently into 1 is what makes a script branch useless. The table is pinned by
`internal/cmd/exit_codes_test.go`.
