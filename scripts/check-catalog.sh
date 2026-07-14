#!/usr/bin/env bash
# Verify the embedded catalog/ submodule is pinned to the LATEST published
# catalog release tag. A platform release must ship a released, current catalog
# version — never an unreleased WIP commit or a stale older tag.
#
# Run by scripts/release.sh (before tagging) and by .github/workflows/release.yml
# (as a gate). If it fails, release the catalog first, then:
#   git -C catalog fetch --tags && git -C catalog checkout <latest> && git add catalog
set -euo pipefail

SUB="catalog"
cd "$(git rev-parse --show-toplevel)"

git submodule update --init "$SUB" >/dev/null 2>&1 || true
[ -e "$SUB/.git" ] || { echo "check-catalog: submodule '$SUB' is not initialized" >&2; exit 1; }

git -C "$SUB" fetch --quiet --tags origin

latest="$(git -C "$SUB" tag --sort=-v:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | head -n1 || true)"
if [ -z "$latest" ]; then
  echo "check-catalog: the catalog has no release tags yet — cut one with 'make release' in growrig-catalog." >&2
  exit 1
fi

pinned="$(git -C "$SUB" rev-parse HEAD)"
want="$(git -C "$SUB" rev-list -n1 "$latest")"

if [ "$pinned" != "$want" ]; then
  {
    echo "check-catalog: submodule '$SUB' is not at the latest catalog release ($latest)."
    echo "  pinned:  $pinned"
    echo "  $latest: $want"
    echo "  fix: git -C $SUB fetch --tags && git -C $SUB checkout $latest && git add $SUB"
    echo "  (if $latest is missing content you need, cut a new catalog release first)"
  } >&2
  exit 1
fi

echo "check-catalog: '$SUB' pinned to latest catalog release $latest ($pinned)"
