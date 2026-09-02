#!/usr/bin/env bash
#
# doc-i18n-check.sh — are the eight documentation translations complete against
# the French source?
#
# French is the source language of doc/source; the other languages are gettext
# catalogues under doc/source/locale/<lang>/LC_MESSAGES/*.po. This script
# regenerates the .pot templates from the current sources, merges every
# catalogue against them in a temporary directory (the committed .po files are
# never touched) and counts the entries left untranslated or fuzzy — i.e. the
# strings the online docs would fall back to French for.
#
# Usage:
#   scripts/doc-i18n-check.sh [--strict] [--langs "en de ..."]
#
#   --strict   exit 1 when anything is missing (CI on a release tag). Without
#              it the gaps are reported — as ::warning:: annotations under
#              GitHub Actions — and the exit code stays 0 (pull requests and
#              merges may legitimately carry untranslated strings: CLAUDE.md
#              says the translations are refreshed at release time).
#   --langs    space-separated subset of languages (default: all eight).
#
# Needs sphinx-build (doc/requirements.txt) and GNU gettext (msgmerge, msgfmt,
# msginit). sphinx-build is usually not on the system PATH: either activate the
# repo virtualenv first (`source .venv/bin/activate`, see doc/build.py) or let
# the script pick it up itself — it falls back to .venv/bin/sphinx-build at the
# repository root, and SPHINX_BUILD=/path/to/sphinx-build overrides both.
#
# The .pot templates are written to doc/build/gettext, and the paths are given
# RELATIVE to the repository root on purpose: sphinx-build records the source
# location of every string (the `#:` comments) relative to the output
# directory, so an absolute output path would produce absolute `#:` references
# — harmless here (nothing is written back), but the same command is what
# regenerates the committed catalogues, and there it must stay relative.

set -euo pipefail

STRICT=0
LANGS="en de el es fi it ja ru"
while [ $# -gt 0 ]; do
  case "$1" in
    --strict) STRICT=1 ;;
    --langs) shift; LANGS="${1:-}" ;;
    -h|--help) sed -n '2,36p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "unknown option: $1" >&2; exit 2 ;;
  esac
  shift
done

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

SPHINX_BUILD="${SPHINX_BUILD:-}"
if [ -z "$SPHINX_BUILD" ]; then
  if command -v sphinx-build >/dev/null 2>&1; then
    SPHINX_BUILD=sphinx-build
  elif [ -x .venv/bin/sphinx-build ]; then
    SPHINX_BUILD=.venv/bin/sphinx-build
  else
    echo "sphinx-build not found: activate .venv or pip install -r doc/requirements.txt" >&2
    exit 2
  fi
fi
for tool in msgmerge msgfmt msginit; do
  command -v "$tool" >/dev/null 2>&1 || { echo "$tool not found: install GNU gettext" >&2; exit 2; }
done

POT_DIR=doc/build/gettext
rm -rf "$POT_DIR"
"$SPHINX_BUILD" -q -b gettext doc/source "$POT_DIR"

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

# msgfmt --statistics prints "N translated messages, N fuzzy translations,
# N untranslated messages." on stderr, omitting the zero categories; the
# header entry is not counted. Returns "<fuzzy> <untranslated>".
count_gaps() {
  local stats
  stats="$(msgfmt -o /dev/null --statistics "$1" 2>&1 >/dev/null || true)"
  local fuzzy untranslated
  fuzzy="$(sed -n 's/.*[^0-9]\([0-9]\+\) fuzzy.*/\1/p; s/^\([0-9]\+\) fuzzy.*/\1/p' <<<"$stats" | head -1)"
  untranslated="$(sed -n 's/.*[^0-9]\([0-9]\+\) untranslated.*/\1/p; s/^\([0-9]\+\) untranslated.*/\1/p' <<<"$stats" | head -1)"
  echo "${fuzzy:-0} ${untranslated:-0}"
}

annotate() { # level file message
  if [ -n "${GITHUB_ACTIONS:-}" ]; then
    echo "::$1 file=$2::$3"
  else
    echo "  $2: $3"
  fi
}

level=warning
[ "$STRICT" = 1 ] && level=error

total_fuzzy=0
total_untranslated=0
for lang in $LANGS; do
  lang_fuzzy=0
  lang_untranslated=0
  for pot in "$POT_DIR"/*.pot; do
    name="$(basename "$pot" .pot)"
    po="doc/source/locale/$lang/LC_MESSAGES/$name.po"
    merged="$TMP/$lang-$name.po"
    if [ -f "$po" ]; then
      msgmerge --no-fuzzy-matching --quiet -o "$merged" "$po" "$pot"
    else
      # No catalogue at all: every string of the document is untranslated.
      msginit --no-translator --locale="$lang.UTF-8" -i "$pot" -o "$merged" >/dev/null 2>&1
    fi
    read -r fuzzy untranslated <<<"$(count_gaps "$merged")"
    if [ "$fuzzy" -gt 0 ] || [ "$untranslated" -gt 0 ]; then
      annotate "$level" "$po" "$lang/$name: $untranslated untranslated, $fuzzy fuzzy"
    fi
    lang_fuzzy=$((lang_fuzzy + fuzzy))
    lang_untranslated=$((lang_untranslated + untranslated))
  done
  printf '%-3s %4d untranslated, %4d fuzzy\n' "$lang" "$lang_untranslated" "$lang_fuzzy"
  total_fuzzy=$((total_fuzzy + lang_fuzzy))
  total_untranslated=$((total_untranslated + lang_untranslated))
done

total=$((total_fuzzy + total_untranslated))
if [ "$total" -eq 0 ]; then
  echo "doc i18n: all translations complete."
  exit 0
fi
echo "doc i18n: $total_untranslated untranslated + $total_fuzzy fuzzy strings across $LANGS."
if [ "$STRICT" = 1 ]; then
  echo "Refresh the catalogues (sphinx-build -b gettext doc/source doc/build/gettext; sphinx-intl update -p doc/build/gettext -d doc/source/locale) and translate them before tagging." >&2
  exit 1
fi
[ -n "${GITHUB_ACTIONS:-}" ] && echo "::warning::doc i18n: $total_untranslated untranslated + $total_fuzzy fuzzy strings (blocking on release tags)"
exit 0
