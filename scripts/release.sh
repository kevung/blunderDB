#!/usr/bin/env bash
#
# release.sh — Automate blunderDB version release
#
# Updates version numbers in all required files, commits, tags, and pushes.
#
# Usage:
#   ./scripts/release.sh <version> [options]
#
# Examples:
#   ./scripts/release.sh 1.0.0
#   ./scripts/release.sh 1.0.0 --changelog "Bug fixes and new features."
#   ./scripts/release.sh 1.0.0 --push
#   ./scripts/release.sh 1.0.0 --changelog "Bug fixes." --push
#   ./scripts/release.sh --check          # Show current versions; exit 1 if
#                                         # conf.py/metaStore.js/wails.json
#                                         # disagree (CI runs this)
#
# Files updated:
#   - doc/source/conf.py           (Sphinx release variable)
#   - frontend/src/stores/metaStore.js  (application version in UI)
#   - wails.json                   (info.productVersion → Windows .exe version
#                                   resource, macOS Info.plist CFBundleVersion)
#   - doc/source/historique.rst    (changelog entry, if --changelog given)
#   - packaging/flatpak/io.github.kevung.blunderDB.yml  (url/sha256 bumped to
#                                   the latest ALREADY-published release, so
#                                   the tracked manifest stays buildable —
#                                   see packaging/flatpak/README.md §3)
#
# After running, the CI workflow (.github/workflows/build.yml) is triggered
# by the pushed tag to build binaries, PDFs, and update GitHub Pages.
#
# `blunderdb version` (CLI) reports the app version from a build-time ldflag
# (internal/cli.appVersion, see internal/cli/version.go), not from a file this
# script edits: `make build` and the CI matrix both derive it from `git
# describe --tags`, i.e. the very tag this script creates. Nothing to update
# here for that.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

CONF_PY="$REPO_ROOT/doc/source/conf.py"
META_STORE="$REPO_ROOT/frontend/src/stores/metaStore.js"
HISTORIQUE_RST="$REPO_ROOT/doc/source/historique.rst"
WAILS_JSON="$REPO_ROOT/wails.json"
FLATPAK_MANIFEST="$REPO_ROOT/packaging/flatpak/io.github.kevung.blunderDB.yml"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

info()  { echo -e "${BLUE}[INFO]${NC} $*"; }
ok()    { echo -e "${GREEN}[OK]${NC}   $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERR]${NC}  $*" >&2; }

usage() {
    cat <<EOF
Usage: $(basename "$0") <version> [options]
       $(basename "$0") --check

Arguments:
  <version>       New version number (semver format: X.Y.Z)

Options:
  --changelog|-c <text>   Changelog bullet for doc/source/historique.rst (X.Y.0 only)
  --push|-p               Push commit and tag to origin after creating them
  --dry-run|-n            Show what would be changed without modifying files
  --check                 Show current version numbers and exit
  --help|-h               Show this help message

Examples:
  $(basename "$0") 1.0.0
  $(basename "$0") 1.0.0 --changelog "New feature X. Bug fix Y."
  $(basename "$0") 1.0.0 -c "New feature X." --push
  $(basename "$0") --check
EOF
}

# Extract current versions from source files
get_current_versions() {
    local conf_ver meta_ver wails_ver
    conf_ver=$(grep -oP "^release\s*=\s*'\\K[^']+" "$CONF_PY" 2>/dev/null || echo "NOT FOUND")
    meta_ver=$(grep -oP "applicationVersion:\s*'\\K[^']+" "$META_STORE" 2>/dev/null || echo "NOT FOUND")
    wails_ver=$(grep -oP '"productVersion":\s*"\K[^"]+' "$WAILS_JSON" 2>/dev/null || echo "NOT FOUND")
    echo "$conf_ver" "$meta_ver" "$wails_ver"
}

check_versions() {
    read -r conf_ver meta_ver wails_ver <<< "$(get_current_versions)"
    local latest_tag
    latest_tag=$(git -C "$REPO_ROOT" describe --tags --abbrev=0 2>/dev/null || echo "NO TAG")

    echo ""
    echo "Current version numbers in blunderDB:"
    echo "======================================="
    printf "  %-40s %s\n" "doc/source/conf.py (release)" "$conf_ver"
    printf "  %-40s %s\n" "frontend/src/stores/metaStore.js" "$meta_ver"
    printf "  %-40s %s\n" "wails.json (info.productVersion)" "$wails_ver"
    printf "  %-40s %s\n" "Latest git tag" "$latest_tag"
    echo ""

    local in_sync=true
    if [[ "$conf_ver" == "$meta_ver" && "$conf_ver" == "$wails_ver" ]]; then
        echo -e "  ${GREEN}✓ conf.py, metaStore.js and wails.json are in sync${NC}"
    else
        echo -e "  ${RED}✗ conf.py, metaStore.js and wails.json are OUT OF SYNC${NC}"
        in_sync=false
    fi

    # Informational only: between two releases the files legitimately carry
    # the version of the last tag, and a shallow CI checkout has no tag at all.
    if [[ "$conf_ver" == "$latest_tag" ]]; then
        echo -e "  ${GREEN}✓ conf.py matches latest git tag${NC}"
    else
        echo -e "  ${YELLOW}! conf.py ($conf_ver) differs from latest tag ($latest_tag)${NC}"
    fi
    echo ""

    $in_sync
}

validate_semver() {
    local ver="$1"
    if [[ ! "$ver" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        error "Invalid version format: '$ver'. Expected semver X.Y.Z (e.g., 1.0.0)"
        exit 1
    fi
}

# --- Parse arguments ---

VERSION=""
CHANGELOG=""
DO_PUSH=false
DRY_RUN=false
CHECK_ONLY=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --check)
            CHECK_ONLY=true
            shift
            ;;
        --changelog|-c)
            CHANGELOG="$2"
            shift 2
            ;;
        --push|-p)
            DO_PUSH=true
            shift
            ;;
        --dry-run|-n)
            DRY_RUN=true
            shift
            ;;
        --help|-h)
            usage
            exit 0
            ;;
        -*)
            error "Unknown option: $1"
            usage
            exit 1
            ;;
        *)
            if [[ -z "$VERSION" ]]; then
                VERSION="$1"
            else
                error "Unexpected argument: $1"
                usage
                exit 1
            fi
            shift
            ;;
    esac
done

# --- Check mode ---
if $CHECK_ONLY; then
    check_versions
    exit $?
fi

# --- Validate inputs ---
if [[ -z "$VERSION" ]]; then
    error "Version number is required."
    echo ""
    usage
    exit 1
fi

validate_semver "$VERSION"

# Check we're in a git repo
if ! git -C "$REPO_ROOT" rev-parse --git-dir &>/dev/null; then
    error "Not in a git repository"
    exit 1
fi

# Check for existing tag
if git -C "$REPO_ROOT" tag -l "$VERSION" | grep -q "^$VERSION$"; then
    error "Tag '$VERSION' already exists."
    exit 1
fi

# Check for uncommitted changes (warn only)
if ! git -C "$REPO_ROOT" diff --quiet 2>/dev/null || ! git -C "$REPO_ROOT" diff --cached --quiet 2>/dev/null; then
    warn "You have uncommitted changes in the working tree."
    if ! $DRY_RUN; then
        read -rp "Continue anyway? [y/N] " yn
        [[ "$yn" =~ ^[Yy]$ ]] || exit 1
    fi
fi

# --- Show current state ---
read -r conf_ver meta_ver wails_ver <<< "$(get_current_versions)"
echo ""
info "Current versions:"
echo "  conf.py:       $conf_ver"
echo "  metaStore.js:  $meta_ver"
echo "  wails.json:    $wails_ver"
info "New version:     $VERSION"
if [[ -n "$CHANGELOG" ]]; then
    info "Changelog:       $CHANGELOG"
fi
echo ""

if $DRY_RUN; then
    info "[DRY RUN] The following changes would be made:"
    echo ""
fi

# --- Update doc/source/conf.py ---
do_update_conf_py() {
    info "Updating $CONF_PY ..."
    if $DRY_RUN; then
        echo "  release = '$conf_ver'  →  release = '$VERSION'"
        return
    fi
    sed -i "s/^release = '.*'/release = '$VERSION'/" "$CONF_PY"
    ok "Updated conf.py: release = '$VERSION'"
}

# --- Update frontend/src/stores/metaStore.js ---
do_update_meta_store() {
    info "Updating $META_STORE ..."
    if $DRY_RUN; then
        echo "  applicationVersion: '$meta_ver'  →  applicationVersion: '$VERSION'"
        return
    fi
    sed -i "s/applicationVersion: '.*'/applicationVersion: '$VERSION'/" "$META_STORE"
    ok "Updated metaStore.js: applicationVersion = '$VERSION'"
}

# --- Update wails.json ---
# Feeds the Windows .exe version resource and the macOS Info.plist
# CFBundleVersion. Left unset, every non-macOS binary ships as 0.0.0.
do_update_wails_json() {
    info "Updating $WAILS_JSON ..."
    if $DRY_RUN; then
        echo "  \"productVersion\": \"$wails_ver\"  →  \"productVersion\": \"$VERSION\""
        return
    fi
    sed -i "s/\"productVersion\": \"[^\"]*\"/\"productVersion\": \"$VERSION\"/" "$WAILS_JSON"
    ok "Updated wails.json: productVersion = \"$VERSION\""
}

# --- Update doc/source/historique.rst changelog ---
# The version history is a page of its own (historique.rst): one section per
# X.Y.0 release, newest first, each a bulleted list — one bullet per change so
# a later correction invalidates one translated string, not the whole entry.
# Patch releases (X.Y.Z, Z>0) get no section (CLAUDE.md, "major-only").
do_update_changelog() {
    if [[ -z "$CHANGELOG" ]]; then
        warn "No changelog text provided (use --changelog). Skipping historique.rst update."
        return
    fi
    if [[ "$VERSION" != *.0 ]]; then
        warn "Patch release $VERSION: the version history lists X.Y.0 only, skipping historique.rst."
        return
    fi

    info "Updating $HISTORIQUE_RST changelog ..."

    local today heading underline
    today=$(date +%Y-%m-%d)
    heading="$VERSION ($today)"
    underline=$(printf '%*s' "${#heading}" '' | tr ' ' '-')

    if $DRY_RUN; then
        echo "  New section: $heading"
        echo "  - $CHANGELOG"
        return
    fi

    # Insert before the first existing version heading ("X.Y.Z (YYYY-MM-DD)").
    local first_version_line
    first_version_line=$(grep -nE '^[0-9]+\.[0-9]+\.[0-9]+ \(' "$HISTORIQUE_RST" | head -1 | cut -d: -f1)
    if [[ -z "$first_version_line" ]]; then
        error "Could not find a version heading in historique.rst to insert the new section before."
        return
    fi

    local tmp
    tmp=$(mktemp)
    {
        head -n $((first_version_line - 1)) "$HISTORIQUE_RST"
        printf '%s\n%s\n\n- %s\n\n' "$heading" "$underline" "$CHANGELOG"
        tail -n +"$first_version_line" "$HISTORIQUE_RST"
    } > "$tmp"
    mv "$tmp" "$HISTORIQUE_RST"
    ok "Added changelog section for $heading"
}

# --- Update packaging/flatpak/io.github.kevung.blunderDB.yml ---
# Points the committed manifest at the latest ALREADY-published release, not
# the one being cut here (its assets don't exist yet) — so the file checked
# into the repo always builds as-is with `flatpak-builder`. CI's `flatpak`
# job never reads this file's url/sha256 (it renders its own copy against
# the tarball it just built); this is purely for a local/offline build of
# the tracked manifest. See packaging/flatpak/README.md §3.
do_update_flatpak_manifest() {
    local prev_tag asset url
    prev_tag=$(git -C "$REPO_ROOT" describe --tags --abbrev=0 2>/dev/null || echo "")
    if [[ -z "$prev_tag" ]]; then
        warn "No previous git tag found; skipping Flatpak manifest update."
        return
    fi

    asset="blunderDB-linux-webkit2gtk-4.1-${prev_tag}.tar.gz"
    url="https://github.com/kevung/blunderDB/releases/download/${prev_tag}/${asset}"

    info "Updating $FLATPAK_MANIFEST (pointing at published release $prev_tag) ..."
    if $DRY_RUN; then
        echo "  url    → $url"
        echo "  sha256 → (downloaded from ${url}.sha256)"
        return
    fi

    local sumfile sha
    sumfile="$(mktemp)"
    if ! curl -fsSL "${url}.sha256" -o "$sumfile" 2>/dev/null; then
        warn "Could not download ${asset}.sha256 (offline, or $prev_tag has no Linux asset). Leaving $FLATPAK_MANIFEST untouched."
        rm -f "$sumfile"
        return
    fi
    sha="$(cut -d' ' -f1 "$sumfile" | tr '[:upper:]' '[:lower:]')"
    rm -f "$sumfile"
    if [[ ! "$sha" =~ ^[0-9a-f]{64}$ ]]; then
        warn "Malformed checksum for $asset; leaving $FLATPAK_MANIFEST untouched."
        return
    fi

    sed -i \
        -e "s#^\( *\)url: https://github.com/kevung/blunderDB/releases/download/.*#\1url: ${url}#" \
        -e "s/^\( *\)sha256: [0-9a-fA-F]\{1,\}$/\1sha256: ${sha}/" \
        "$FLATPAK_MANIFEST"
    ok "Updated Flatpak manifest: release $prev_tag, sha256 $sha"
}

# --- Perform updates ---
do_update_conf_py
do_update_meta_store
do_update_wails_json
do_update_changelog
do_update_flatpak_manifest

if $DRY_RUN; then
    echo ""
    info "[DRY RUN] No files were modified."
    exit 0
fi

# --- Show diff ---
echo ""
info "Changes made:"
git -C "$REPO_ROOT" diff --stat
echo ""
git -C "$REPO_ROOT" diff

# --- Git commit and tag ---
echo ""
read -rp "Commit and tag as '$VERSION'? [Y/n] " yn
if [[ "$yn" =~ ^[Nn]$ ]]; then
    warn "Changes were made to files but NOT committed. You can:"
    echo "  git add -A && git commit -m 'Release $VERSION'"
    echo "  git tag $VERSION"
    exit 0
fi

git -C "$REPO_ROOT" add "$CONF_PY" "$META_STORE" "$WAILS_JSON" "$HISTORIQUE_RST" "$FLATPAK_MANIFEST"
git -C "$REPO_ROOT" commit -m "Release $VERSION"

# Signed tag when a GPG (or SSH, see gpg.format) signing key is configured
# (E.7, #223) — this signs the git TAG object itself, unrelated to the
# per-asset minisign/attestation signing the release workflow does (see
# packaging/minisign/README.md). Falls back to a lightweight tag with a
# warning rather than aborting the release: most contributors' machines
# have no signing key configured, and CI never runs this script.
tag_err="$(mktemp)"
if git -C "$REPO_ROOT" tag -s "$VERSION" -m "Release $VERSION" 2>"$tag_err"; then
    ok "Created commit and signed tag '$VERSION'"
else
    cat "$tag_err" >&2
    warn "No signing key configured (user.signingkey / gpg.format) — created an unsigned tag."
    git -C "$REPO_ROOT" tag "$VERSION"
    ok "Created commit and unsigned tag '$VERSION'"
fi
rm -f "$tag_err"

# --- Push ---
if $DO_PUSH; then
    info "Pushing to origin ..."
    git -C "$REPO_ROOT" push origin main
    git -C "$REPO_ROOT" push origin "$VERSION"
    ok "Pushed commit and tag to origin"
else
    echo ""
    info "To push the release (triggers CI build + doc deployment):"
    echo "  git push origin main && git push origin $VERSION"
fi

echo ""
ok "Release $VERSION complete!"
