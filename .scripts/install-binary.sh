#!/bin/sh

# SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
#
# SPDX-License-Identifier: MIT

set -eu

# install-binary.sh — the m6e-only local install: build the host-native pf-cli
# (reusing build-binaries.sh, which defaults GOOS/GOARCH to the host and drops the
# unsuffixed dist/pf-cli) and copy it into ~/.local/bin so the make-plane bootstraps
# with a fresh CLI. Self-contained on purpose: a build-binaries tool homes on ONE CI
# node (binaries-built), so routing the install through the DAG would multi-home it
# AND gate the install on source-is-ready — this quick, ungated build mirrors the old
# install-local instead. The forge lowerings prune this tool (they release artifacts).

dst="${HOME}/.local/bin"

log() { printf '[install-binary] %s\n' "$*" >&2; }

# Build host-native (no version arg → build-binaries derives one from git).
log "building host-native pf-cli"
.scripts/build-binaries.sh

mkdir -p "${dst}"
install -m 0755 dist/pf-cli "${dst}/pf-cli"
log "installed dist/pf-cli -> ${dst}/pf-cli"
