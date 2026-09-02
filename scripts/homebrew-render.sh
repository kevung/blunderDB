#!/usr/bin/env bash
#
# homebrew-render.sh — render the Homebrew cask (packaging/homebrew/blunderdb.rb.in)
# for a released version. It writes the file a tap expects at
# Casks/blunderdb.rb; it does NOT push to any tap (see
# packaging/homebrew/README.md for the human step).
#
# Usage:
#   scripts/homebrew-render.sh <version> [--assets DIR] [--out DIR]
#
# <version>     released tag, e.g. 0.34.1 — blunderDB-macos-<version>.zip and
#               its .sha256 must be published on the GitHub release.
# --assets DIR  read blunderDB-macos-<version>.zip.sha256 from DIR (searched
#               recursively) instead of downloading it — what CI does with the
#               build artifact, so the job never races the release upload.
# --out DIR     output directory (default: ./dist/homebrew). The cask is
#               written to DIR/Casks/blunderdb.rb.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TEMPLATE="$REPO_ROOT/packaging/homebrew/blunderdb.rb.in"

VERSION="${1:-}"
[[ -n "$VERSION" ]] || { echo "usage: $(basename "$0") <version> [--assets DIR] [--out DIR]" >&2; exit 1; }
shift

ASSETS_DIR=""
OUT_DIR="$REPO_ROOT/dist/homebrew"
while [[ $# -gt 0 ]]; do
  case "$1" in
    --assets) ASSETS_DIR="$2"; shift 2 ;;
    --out)    OUT_DIR="$2"; shift 2 ;;
    *) echo "unknown option: $1" >&2; exit 1 ;;
  esac
done

ASSET="blunderDB-macos-${VERSION}.zip"
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

# `<hex>  <name>` (shasum -a 256 on the macOS runner) — first field only.
# Homebrew spells hashes in lower case.
SHA="$(cut -d' ' -f1 "$workdir/$SUMFILE" | tr '[:upper:]' '[:lower:]')"
[[ "$SHA" =~ ^[0-9a-f]{64}$ ]] || { echo "malformed checksum in $SUMFILE: $SHA" >&2; exit 1; }
echo "sha256: $SHA"

dest="$OUT_DIR/Casks"
mkdir -p "$dest"
out="$dest/blunderdb.rb"
sed -e "s/@VERSION@/${VERSION}/g" -e "s/@SHA256@/${SHA}/g" "$TEMPLATE" > "$out"

if grep -n '@[A-Z_]*@' "$out"; then
  echo "unsubstituted placeholder(s) above" >&2
  exit 1
fi
echo "Wrote $out"
