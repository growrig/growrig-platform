#!/usr/bin/env bash
# Cut a GrowRig release. The git tag is the single source of truth for the
# version; this script keeps the add-on manifest and CHANGELOG in lockstep with
# it, then pushes the tag that triggers .github/workflows/release.yml.
#
#   make release VERSION=0.2.0
#
# What it does:
#   1. validates VERSION (X.Y.Z) and that the tag does not already exist
#   2. requires a clean tree on the default branch, up to date with origin
#   3. writes VERSION into ha-addon/growrig/config.yaml
#   4. moves "## Unreleased" CHANGELOG entries into "## X.Y.Z — <today>"
#   5. commits "release: vX.Y.Z", creates annotated tag vX.Y.Z, pushes both
#
# After CI publishes the images, bump the version in the public
# growrig-ha-addons repo's config.yaml so Home Assistant offers the update.
set -euo pipefail

VERSION="${VERSION:-${1:-}}"
MANIFEST="ha-addon/growrig/config.yaml"
CHANGELOG="CHANGELOG.md"
DEFAULT_BRANCH="main"

die() { echo "release: $*" >&2; exit 1; }

[ -n "$VERSION" ] || die "set VERSION, e.g. make release VERSION=0.2.0"
[[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || die "VERSION must be X.Y.Z (got '$VERSION')"

TAG="v$VERSION"
cd "$(git rev-parse --show-toplevel)"

[ -f "$MANIFEST" ] || die "missing $MANIFEST — run from the growrig-platform repo"
git rev-parse -q --verify "refs/tags/$TAG" >/dev/null && die "tag $TAG already exists"

branch="$(git rev-parse --abbrev-ref HEAD)"
[ "$branch" = "$DEFAULT_BRANCH" ] || die "on '$branch'; release from '$DEFAULT_BRANCH'"
[ -z "$(git status --porcelain)" ] || die "working tree is dirty; commit or stash first"

git fetch --quiet origin "$DEFAULT_BRANCH"
[ "$(git rev-parse HEAD)" = "$(git rev-parse "origin/$DEFAULT_BRANCH")" ] \
  || die "local $DEFAULT_BRANCH is not in sync with origin/$DEFAULT_BRANCH"

grep -q '^## Unreleased' "$CHANGELOG" || die "no '## Unreleased' section in $CHANGELOG"

# The embedded catalog submodule must be at the latest published catalog release.
scripts/check-catalog.sh

echo "release: preparing $TAG"

# 1. Manifest version (quoted string, HA requires it literal in config.yaml).
perl -0pi -e "s/^version: .*$/version: \"$VERSION\"/m" "$MANIFEST"

# 2. Promote Unreleased -> dated version section, leaving Unreleased empty.
today="$(date +%Y-%m-%d)"
awk -v ver="$VERSION" -v date="$today" '
  /^## Unreleased/ && !done {
    print
    print ""
    print "## " ver " — " date
    done = 1
    next
  }
  { print }
' "$CHANGELOG" > "$CHANGELOG.tmp" && mv "$CHANGELOG.tmp" "$CHANGELOG"

# 3. Commit, tag, push.
git add "$MANIFEST" "$CHANGELOG"
git commit -m "release: $TAG"
git tag -a "$TAG" -m "GrowRig $TAG"
git push origin "HEAD:$DEFAULT_BRANCH" --follow-tags

echo "release: pushed $TAG — CI will test, publish images, and cut the GitHub Release."
echo "release: next, bump version to $VERSION in growrig-ha-addons/config.yaml so HA offers the update."
