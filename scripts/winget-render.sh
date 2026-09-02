#!/usr/bin/env bash
#
# winget-render.sh — render the winget manifests (packaging/winget/*.in) for a
# released version. Produces the directory tree a pull request on
# microsoft/winget-pkgs expects; it does NOT submit anything (see
# packaging/winget/README.md for the human step).
#
# Usage:
#   scripts/winget-render.sh <version> [--assets DIR] [--out DIR] [--date YYYY-MM-DD]
#
# <version>     released tag, e.g. 0.34.1 — blunderDB-windows-<version>.exe and
#               its .sha256 must be published on the GitHub release.
# --assets DIR  read blunderDB-windows-<version>.exe.sha256 from DIR (searched
#               recursively) instead of downloading it — what CI does with the
#               build artifact, so the job never races the release upload.
# --out DIR     output root (default: ./dist/winget). Manifests land in
#               DIR/manifests/k/KevinUnger/blunderDB/<version>/.
# --date        ReleaseDate to stamp (default: today, UTC).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TEMPLATE_DIR="$REPO_ROOT/packaging/winget"
PACKAGE_ID="KevinUnger.blunderDB"

VERSION="${1:-}"
[[ -n "$VERSION" ]] || { echo "usage: $(basename "$0") <version> [--assets DIR] [--out DIR] [--date YYYY-MM-DD]" >&2; exit 1; }
shift

ASSETS_DIR=""
OUT_DIR="$REPO_ROOT/dist/winget"
RELEASE_DATE="$(date -u +%F)"
while [[ $# -gt 0 ]]; do
  case "$1" in
    --assets) ASSETS_DIR="$2"; shift 2 ;;
    --out)    OUT_DIR="$2"; shift 2 ;;
    --date)   RELEASE_DATE="$2"; shift 2 ;;
    *) echo "unknown option: $1" >&2; exit 1 ;;
  esac
done

ASSET="blunderDB-windows-${VERSION}.exe"
SUMFILE="${ASSET}.sha256"
URL="https://github.com/kevung/blunderDB/releases/download/${VERSION}/${SUMFILE}"

workdir="$(mktemp -d)"
trap 'rm -rf "$workdir"' EXIT

if [[ -n "$ASSETS_DIR" ]]; then
  src="$(find "$ASSETS_DIR" -name "$SUMFILE" -type f | head -1)"
  [[ -n "$src" ]] || { echo "$SUMFILE not found under $ASSETS_DIR" >&2; exit 1; }
  cp "$src" "$workdir/$SUMFILE"
else
  echo "Downloading $URL ..."
  curl -fsSL "$URL" -o "$workdir/$SUMFILE"
fi

# The checksum file is `<hex> *<name>` (sha256sum on Git Bash marks binary
# mode with `*`) — keep the first field only. winget-pkgs spells hashes in
# upper case; the schema accepts either, the convention avoids review noise.
SHA="$(cut -d' ' -f1 "$workdir/$SUMFILE" | tr '[:lower:]' '[:upper:]')"
[[ "$SHA" =~ ^[0-9A-F]{64}$ ]] || { echo "malformed checksum in $SUMFILE: $SHA" >&2; exit 1; }
echo "sha256: $SHA"

dest="$OUT_DIR/manifests/k/KevinUnger/blunderDB/$VERSION"
mkdir -p "$dest"
for tpl in "$TEMPLATE_DIR"/*.yaml.in; do
  out="$dest/$(basename "${tpl%.in}")"
  sed -e "s/@VERSION@/${VERSION}/g" \
      -e "s/@SHA256@/${SHA}/g" \
      -e "s/@RELEASE_DATE@/${RELEASE_DATE}/g" "$tpl" > "$out"
  echo "Wrote $out"
done

# A leftover placeholder means a template gained a token this script does not
# know about — fail loudly rather than ship it.
if grep -rn '@[A-Z_]*@' "$dest"; then
  echo "unsubstituted placeholder(s) above" >&2
  exit 1
fi
echo "Manifests for ${PACKAGE_ID} ${VERSION} are in $dest"
