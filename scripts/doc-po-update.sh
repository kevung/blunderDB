#!/usr/bin/env bash
#
# doc-po-update.sh — regenerate the eight gettext catalogues from the French
# Sphinx sources, then repair what the tooling gets wrong.
#
# Use this instead of calling sphinx-build/sphinx-intl by hand: it encodes
# three things that are easy to get wrong and expensive to notice.
#
#  1. The .pot templates are written to a path RELATIVE to the repository
#     root. sphinx-build records each string's source location (the `#:`
#     comments) relative to the output directory, so an absolute output path
#     rewrites every reference in every catalogue — a diff of thousands of
#     lines hiding the handful that matter.
#
#  2. msgmerge flags an entry `python-format` whenever the French text holds
#     a percent sign followed by a space ("93,4 % (3735/4000)" reads as the
#     directive `% (`), and the English translation "93.4% (…)" is then an
#     invalid format string that msgfmt refuses to compile. It also APPENDS
#     that flag behind an existing `no-python-format`, and gettext honours
#     the last one — so the false positive comes back at every update and
#     must be normalised after, not once. No catalogue in this project holds
#     a real Python format string; if one ever does, this normalisation is
#     what has to change.
#
#  3. msgcat is never used to re-wrap a catalogue. Babel and msgcat disagree
#     on line breaks, so one pass rewrites every entry of the file.
#
# What it does NOT do: translate. New `msgstr ""` entries are yours to fill —
# the script prints how many are left per language, and scripts/doc-i18n-check.sh
# is the gate.
#
# Usage:
#   scripts/doc-po-update.sh [--langs "en de ..."]
#
# Needs sphinx-build (doc/requirements.txt; activate .venv or set SPHINX_BUILD),
# sphinx-intl and GNU gettext.

set -euo pipefail

LANGS="en,de,el,es,fi,it,ja,ru"
while [ $# -gt 0 ]; do
  case "$1" in
    --langs) shift; LANGS="$(echo "${1:-}" | tr ' ' ',')" ;;
    -h|--help) sed -n '2,40p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
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
    SPHINX_BUILD="$ROOT/.venv/bin/sphinx-build"
  else
    echo "sphinx-build not found: activate .venv or pip install -r doc/requirements.txt" >&2
    exit 1
  fi
fi
SPHINX_INTL="${SPHINX_INTL:-}"
if [ -z "$SPHINX_INTL" ]; then
  if command -v sphinx-intl >/dev/null 2>&1; then
    SPHINX_INTL=sphinx-intl
  elif [ -x .venv/bin/sphinx-intl ]; then
    SPHINX_INTL="$ROOT/.venv/bin/sphinx-intl"
  else
    echo "sphinx-intl not found: activate .venv or pip install -r doc/requirements.txt" >&2
    exit 1
  fi
fi

# (1) relative output path, on purpose — see the header.
( cd doc && "$SPHINX_BUILD" -q -b gettext source build/gettext )
( cd doc && "$SPHINX_INTL" update -p build/gettext -l "$LANGS" )

# (2) msgmerge's false positive, normalised after every update.
grep -rl 'python-format' doc/source/locale/ 2>/dev/null \
  | xargs -r sed -i -E 's/^#,.*python-format.*$/#, no-python-format/'

status=0
for po in doc/source/locale/*/LC_MESSAGES/*.po; do
  case "$po" in doc/source/locale/fr/*) continue ;; esac   # French is the source
  if ! msgfmt -c -o /dev/null "$po" 2>/tmp/doc-po-update.$$; then
    echo "msgfmt refuses $po:" >&2
    grep -v 'header field' /tmp/doc-po-update.$$ >&2 || true
    status=1
  fi
done
rm -f /tmp/doc-po-update.$$
[ "$status" -eq 0 ] || exit "$status"

echo
echo "Catalogues regenerated. Entries left to translate:"
for lang in $(echo "$LANGS" | tr ',' ' '); do
  n=0
  for po in doc/source/locale/"$lang"/LC_MESSAGES/*.po; do
    [ -e "$po" ] || continue
    # msgattrib prints nothing at all when a catalogue is complete; when it
    # prints anything, the first entry is the file header, not a string.
    c=$(msgattrib --untranslated --no-obsolete "$po" | grep -c '^msgid "' || true)
    [ "$c" -gt 0 ] && n=$((n + c - 1))
  done
  printf '  %-3s %s\n' "$lang" "$n"
done
echo
echo "Then translate the empty msgstr and run scripts/doc-i18n-check.sh."
