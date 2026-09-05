#!/bin/sh

# SPDX-FileCopyrightText: 2026 Damián Búho <damian.buho@proton.me>
#
# SPDX-License-Identifier: MIT

set -eu

# =============================================================================
# gh-release.sh — attach one matrix binary to the GitHub release
# =============================================================================
#
# Creates the release with generated notes and the assets attached; if the
# release already exists (re-run, race, manual pre-create), uploads them
# with --clobber so the latest build wins.
#
# The .torrent and .magnet ride along when this project seeds, and the .asc when
# it signs, found by suffixing the binary path exactly as the Forgejo release
# action does — so neither end has to learn how torrent.sh spells its flat seed
# name. GitHub has no shared release action, so this is the per-project copy of
# that rule.
# =============================================================================

version="${1:?usage: gh-release.sh <version>}"
goos="${GOOS:?GOOS must be set}"
goarch="${GOARCH:?GOARCH must be set}"
asset="dist/pf-cli-${goos}-${goarch}"

log() { printf '[gh-release] %s\n' "$*" >&2; }

# The binary, plus whatever sidecars the torrent and signing steps left beside it.
# A project that opted into neither has none, so an absent one is simply skipped.
set -- "${asset}"
for ext in torrent magnet asc; do
    if [ -f "${asset}.${ext}" ]; then
        set -- "$@" "${asset}.${ext}"
        log "attaching sidecar ${asset}.${ext}"
    else
        log "no ${asset}.${ext} to attach, skipping"
    fi
done

# Create wins on first run; upload-with-clobber wins on re-runs.
if gh release create "${version}" --generate-notes "$@"; then
    log "created release ${version} with $# asset(s), first is ${asset}"
else
    log "release ${version} exists, uploading $# asset(s) for ${asset} (--clobber)"
    gh release upload "${version}" "$@" --clobber
fi
