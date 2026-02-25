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
#   ./scripts/release.sh --check          # Show current versions
#
# Files updated:
#   - doc/source/conf.py           (Sphinx release variable)
#   - frontend/src/stores/metaStore.js  (application version in UI)
#   - doc/source/index.rst         (changelog entry, if --changelog given)
#
# After running, the CI workflow (.github/workflows/build.yml) is triggered
# by the pushed tag to build binaries, PDFs, and update GitHub Pages.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

CONF_PY="$REPO_ROOT/doc/source/conf.py"
META_STORE="$REPO_ROOT/frontend/src/stores/metaStore.js"
INDEX_RST="$REPO_ROOT/doc/source/index.rst"

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
  --changelog|-c <text>   Changelog description for doc/source/index.rst
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
    local conf_ver meta_ver
    conf_ver=$(grep -oP "^release\s*=\s*'\\K[^']+" "$CONF_PY" 2>/dev/null || echo "NOT FOUND")
    meta_ver=$(grep -oP "applicationVersion:\s*'\\K[^']+" "$META_STORE" 2>/dev/null || echo "NOT FOUND")
    echo "$conf_ver" "$meta_ver"
}

check_versions() {
    read -r conf_ver meta_ver <<< "$(get_current_versions)"
    local latest_tag
    latest_tag=$(git -C "$REPO_ROOT" describe --tags --abbrev=0 2>/dev/null || echo "NO TAG")

    echo ""
    echo "Current version numbers in blunderDB:"
    echo "======================================="
    printf "  %-40s %s\n" "doc/source/conf.py (release)" "$conf_ver"
    printf "  %-40s %s\n" "frontend/src/stores/metaStore.js" "$meta_ver"
    printf "  %-40s %s\n" "Latest git tag" "$latest_tag"
    echo ""

    if [[ "$conf_ver" == "$meta_ver" ]]; then
        echo -e "  ${GREEN}✓ conf.py and metaStore.js are in sync${NC}"
    else
        echo -e "  ${RED}✗ conf.py and metaStore.js are OUT OF SYNC${NC}"
    fi

    if [[ "$conf_ver" == "$latest_tag" ]]; then
        echo -e "  ${GREEN}✓ conf.py matches latest git tag${NC}"
    else
        echo -e "  ${YELLOW}! conf.py ($conf_ver) differs from latest tag ($latest_tag)${NC}"
    fi
    echo ""
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
    exit 0
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
read -r conf_ver meta_ver <<< "$(get_current_versions)"
echo ""
info "Current versions:"
echo "  conf.py:       $conf_ver"
echo "  metaStore.js:  $meta_ver"
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

# --- Update doc/source/index.rst changelog ---
do_update_changelog() {
    if [[ -z "$CHANGELOG" ]]; then
        warn "No changelog text provided (use --changelog). Skipping index.rst update."
        return
    fi

    info "Updating $INDEX_RST changelog ..."

    local today
    today=$(date +%d/%m/%Y)

    # Build the new changelog row
    local new_entry="   $VERSION, $today, \"$CHANGELOG\""

    if $DRY_RUN; then
        echo "  New row: $new_entry"
        return
    fi

    # Find the last changelog entry line (last line starting with whitespace + digit before "Sommaire")
    # and append the new entry after it
    # Strategy: find "Sommaire" section and insert before it
    # The changelog entries end with a blank line before "Sommaire"
    local last_entry_line
    last_entry_line=$(grep -n "^Sommaire" "$INDEX_RST" | head -1 | cut -d: -f1)

    if [[ -z "$last_entry_line" ]]; then
        error "Could not find 'Sommaire' section in index.rst to insert changelog entry."
        return
    fi

    # Insert the new entry 2 lines before "Sommaire" (before the blank line)
    # We need to find the actual last content line of the changelog
    local insert_line=$((last_entry_line - 1))

    # Use sed to insert the new entry before the blank line preceding Sommaire
    sed -i "${insert_line}i\\${new_entry}" "$INDEX_RST"
    ok "Added changelog entry for $VERSION ($today)"
}

# --- Perform updates ---
do_update_conf_py
do_update_meta_store
do_update_changelog

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

git -C "$REPO_ROOT" add "$CONF_PY" "$META_STORE" "$INDEX_RST"
git -C "$REPO_ROOT" commit -m "Release $VERSION"
git -C "$REPO_ROOT" tag "$VERSION"
ok "Created commit and tag '$VERSION'"

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
