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
(`projectfile/ci-resolver`).

> **Bridge Revolution Phase 8 (2026-07-07):** `main.go` +
> `internal/{cmd,validate,usersetup}` were extracted from `projectfile/core`
> into this module so core became a pure library. See
> [../bridge-revolution.md](../bridge-revolution.md) and
> [../core/AGENTS.md](../core/AGENTS.md).

## Commands

`get`/`set`/`add`/`del` (dotted-path query + mutate), `convert` (encoding),
`validate` (v1 JSON Schema), `optimize` (strip include-redundant fields),
`cache` (warm/purge HTTP **includes** only — SPDX moved to pf-bridge),
`setup` (per-user config). Projections onto
external files/forges/the repository (`bridge`/`forge`/`scan`/`init`) live in
`pf-bridge`, not here.

## Layout

```text
projectfile/cli/
├── main.go               entry point → internal/cmd.Execute (Cobra root)
├── go.mod                module projectfile.org/projectfile/cli
│                         require kiota.ch/projectfile/core + replace => ../core
└── internal/
    ├── cmd/              Cobra commands + derived (synthetic get addresses)
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
Iterate with `go build ./... && go test ./...`.

## Contracts (do NOT rename)

The binary is `pf-cli` (the `pf-*` triad with `pf-ci` + `pf-bridge`); its image
is `projectfile/cli` (from `identity.name: cli`). The on-disk/config/env
contracts share the `cli`/`pf-cli` spelling: the `# pf-cli-managed:` sentinel,
`PF_CLI_VERBOSE`, the `projectfile/cli.toml` user-config path, and the
`projectfile/cli/` XDG cache slot (its includes cache under
`$XDG_CACHE_HOME/projectfile/cli/`). Renaming any of these would orphan
existing files/config.
