#!/usr/bin/env bash
# Preflight check for the tailor-cv skill's Render step (issue #55).
#
# Compares the host's `typst` version and font availability against
# typst-version.txt (repo root) — the single source of truth ADR-0012's
# container also pins to. Never blocks the render: it prints a warning
# on mismatch or a missing font and always exits 0. The one exception is
# `typst` being entirely absent from the host, in which case this script
# does nothing and leaves the pre-existing "typst not found" failure to
# the `typst compile` invocation that follows, unchanged.
#
# Usage: scripts/preflight-typst.sh [path/to/typst-version.txt]
# Run from the repo root, same as the `typst compile` command it precedes.

set -u

VERSION_FILE="${1:-typst-version.txt}"

if [ ! -f "$VERSION_FILE" ]; then
  # Not run from the repo root (or path override doesn't exist) — try to
  # find it relative to the repo root instead of giving up outright.
  if command -v git >/dev/null 2>&1; then
    root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
    if [ -n "$root" ] && [ -f "$root/typst-version.txt" ]; then
      VERSION_FILE="$root/typst-version.txt"
    fi
  fi
fi

if [ ! -f "$VERSION_FILE" ]; then
  echo "preflight-typst: warning: can't find $VERSION_FILE to check the pinned typst version/font against — skipping preflight check." >&2
  exit 0
fi

# Absent `typst` entirely: leave the existing "typst not found" failure
# to the `typst compile` invocation itself, unchanged.
if ! command -v typst >/dev/null 2>&1; then
  exit 0
fi

# shellcheck disable=SC1090
. "$VERSION_FILE"

detected_version="$(typst --version 2>/dev/null | awk '{print $2}')"

if [ -n "${TYPST_VERSION:-}" ] && [ -n "$detected_version" ] && [ "$detected_version" != "$TYPST_VERSION" ]; then
  echo "preflight-typst: warning: host typst version ($detected_version) does not match the pinned version ($TYPST_VERSION) the container render path uses (ADR-0012). Pagination/spacing may differ from the reference render. Install typst v$TYPST_VERSION to match, or proceed if you judge the drift safe." >&2
fi

if [ -n "${TYPST_FONT_FAMILY:-}" ]; then
  if command -v fc-list >/dev/null 2>&1; then
    if ! fc-list | grep -qi "$TYPST_FONT_FAMILY"; then
      echo "preflight-typst: warning: '$TYPST_FONT_FAMILY' not found in the host's fonts (checked via fc-list). The container render path has it installed (ADR-0012); a substitute font may shift pagination/spacing. Install fonts-liberation (or your OS's equivalent) to match." >&2
    fi
  fi
  # No fc-list on this host: skip the font check silently, per issue #55's
  # scope (no bespoke per-OS font enumeration).
fi

exit 0
