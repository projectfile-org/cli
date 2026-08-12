#!/bin/sh

# SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
#
# SPDX-License-Identifier: MIT

set -eu

# =============================================================================
# fetch-schema.sh — install the v1 JSON Schema for embedding
# =============================================================================
#
# What we are trying to do: give `pf-cli validate` the canonical projectfile v1
# JSON Schema at build time, so the embedded copy cannot drift from the
# specification. The previous arrangement kept a hand-copied v1.json under
# version control with no gate on it; it silently fell 49 lines behind.
#
# Mirrors website/.scripts/fetch-specification.sh: one network clone that works
# the same in local dev and in CI, so an upstream schema change reaches this
# build without a re-commit here.
#
# The dir is underscore-prefixed (not dotted): actions/upload-artifact excludes
# hidden paths by default, so a .specification artifact would upload empty and
# the cross-job build-context hand-off would starve.
#
# Idempotent: clones when _specification/.git is absent; fast-forwards when
# present. Refuses to clobber a non-git _specification/ (likely user mistake).
#
# Override the defaults via env:
#   PROJECTFILE_SPECIFICATION_REPO  git URL (default: kiota origin, anon HTTPS)
#   PROJECTFILE_SPECIFICATION_REF   branch/tag to pin (default: main)
# =============================================================================
# HTTPS (not SSH) so a shared forgejo-runner with no ssh client and no keys can
# clone it: the repo is public on kiota.ch. A maintainer with SSH access can
# still override PROJECTFILE_SPECIFICATION_REPO.

REPO="${PROJECTFILE_SPECIFICATION_REPO:-https://kiota.ch/projectfile/specification.git}"
REF="${PROJECTFILE_SPECIFICATION_REF:-main}"
DEST="_specification"
SOURCE="${DEST}/spec/schema/v1.json"
TARGET="internal/validate/embedded/v1.json"

log() { printf '[fetch-schema] %s\n' "$*" >&2; }

# `git -C DIR` does NOT isolate the repository: GIT_DIR in the environment wins
# over directory discovery. Git exports GIT_DIR to every hook, so under
# lefthook's pre-commit every `git -C ${DEST}` below silently targeted THIS
# repository — repointing its origin at the specification repository and
# resetting the checkout onto a foreign history, which a later push would have
# sent to the wrong place.
#
# The trap is that `rev-parse --show-toplevel` still reports ${DEST}, so a
# toplevel assertion looks like it passes while `remote get-url origin` already
# answers about the enclosing repository. Only clearing the inherited
# environment fixes it.
unset GIT_DIR GIT_WORK_TREE GIT_INDEX_FILE GIT_PREFIX GIT_COMMON_DIR

# Belt and braces: assert the repository git resolves for ${DEST} really is
# ${DEST}'s own. Compares the GIT DIRECTORY, not the toplevel — that is the
# value the environment overrides, so it is the one that detects the fault.
assert_dest_is_repo_root() {
    _got=$(git -C "${DEST}" rev-parse --absolute-git-dir 2>/dev/null || true)
    _want=$(cd "${DEST}" && pwd -P)/.git
    if [ "${_got}" != "${_want}" ]; then
        log "ERROR: git resolves ${DEST} to '${_got:-none}', want '${_want}'"
        log "refusing to run git commands that would act on another repository"
        return 1
    fi
    return 0
}

if [ -d "${DEST}/.git" ]; then
    log "existing clone at ${DEST}, syncing to ref=${REF} repo=${REPO}"
    assert_dest_is_repo_root || exit 1
    git -C "${DEST}" remote set-url origin "${REPO}"
    git -C "${DEST}" fetch --quiet --force origin "${REF}"
    git -C "${DEST}" reset --quiet --hard "origin/${REF}"
elif [ -d "${DEST}" ]; then
    log "ERROR: ${DEST} exists but is not a git clone; refusing to overwrite"
    exit 1
else
    log "cloning repo=${REPO} ref=${REF} into ${DEST}"
    git clone --quiet --no-tags --branch "${REF}" "${REPO}" "${DEST}"
fi

if [ ! -f "${SOURCE}" ]; then
    log "ERROR: ${SOURCE} missing in the specification clone at ref=${REF}"
    exit 1
fi

# The REUSE sidecar travels with the file: the embed target is untracked here,
# so nothing else would carry its licence.
mkdir --parents "$(dirname "${TARGET}")"
cp "${SOURCE}" "${TARGET}"
cp "${SOURCE}.license" "${TARGET}.license"

log "ok schema=${TARGET} bytes=$(wc --bytes <"${TARGET}") spec=$(git -C "${DEST}" rev-parse --short HEAD) ref=${REF}"
