#!/usr/bin/env bash
#
# aur-publish.sh — render the AUR PKGBUILD from the template for a given version
# and (optionally) push it to the AUR. Useful for publishing manually instead of
# relying on the CI workflow (.github/workflows/aur.yml).
#
# Usage:
#   scripts/aur-publish.sh <version> [--push]
#
# <version>  released tag, e.g. 0.27.1 — the matching
#            blunderDB-linux-webkit2gtk-4.1-<version>.tar.gz release asset must
#            already be published on GitHub.
# --push     clone the AUR repo, copy PKGBUILD + .SRCINFO, commit and push.
#            Requires an AUR account with your SSH key registered and `makepkg`.
#
# Without --push it only writes ./PKGBUILD so you can inspect / makepkg locally.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TEMPLATE="$REPO_ROOT/packaging/aur/PKGBUILD.in"
PKGNAME="blunderdb-bin"

VERSION="${1:-}"
DO_PUSH=false
[[ "${2:-}" == "--push" ]] && DO_PUSH=true

if [[ -z "$VERSION" ]]; then
  echo "usage: $(basename "$0") <version> [--push]" >&2
  exit 1
fi

ASSET="blunderDB-linux-webkit2gtk-4.1-${VERSION}.tar.gz"
URL="https://github.com/kevung/blunderDB/releases/download/${VERSION}/${ASSET}"

workdir="$(mktemp -d)"
trap 'rm -rf "$workdir"' EXIT

echo "Downloading $URL ..."
curl -fsSL "$URL" -o "$workdir/$ASSET"
SHA="$(sha256sum "$workdir/$ASSET" | cut -d' ' -f1)"
echo "sha256: $SHA"

sed -e "s/@PKGVER@/${VERSION}/g" -e "s/@SHA256@/${SHA}/g" "$TEMPLATE" > "$workdir/PKGBUILD"
cp "$workdir/PKGBUILD" "$REPO_ROOT/PKGBUILD"
echo "Wrote $REPO_ROOT/PKGBUILD"

if ! $DO_PUSH; then
  echo "Dry run — inspect ./PKGBUILD, then run with --push to publish."
  exit 0
fi

command -v makepkg >/dev/null || { echo "makepkg required for --push (run on Arch)"; exit 1; }

aurdir="$workdir/aur"
git clone "ssh://aur@aur.archlinux.org/${PKGNAME}.git" "$aurdir"
cp "$workdir/PKGBUILD" "$aurdir/PKGBUILD"
( cd "$aurdir" && makepkg --printsrcinfo > .SRCINFO && git add PKGBUILD .SRCINFO \
    && git commit -m "Update to ${VERSION}" && git push )
echo "Pushed ${PKGNAME} ${VERSION} to the AUR."
