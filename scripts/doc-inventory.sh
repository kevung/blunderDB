#!/usr/bin/env bash
#
# doc-inventory.sh — is every command the binary answers to actually written
# down, everywhere a reader looks for it?
#
# The question is not whether a command works: it is whether someone who has
# only the application can find out that it exists. `trash` shipped dispatched,
# tested and described in cli.rst while `blunderdb help` was the one place a
# user without the online docs would have looked; the in-app manual, then a
# hand-written digest, had gone eight sections further behind than that. This
# script is the check that was missing, and it is deliberately narrow: it only
# covers ground no test already holds.
#
# What already has a guard, and is NOT re-checked here (do not duplicate it —
# a second list is what drifts):
#
#   internal/cli/cli_doc_sync_test.go            handlers() <-> doc/source/cli.rst
#   frontend .../commandVocabulary.sync.test.js  processor <-> vocabulary
#   frontend .../helpVocabulary.sync.test.js     vocabulary <-> cmd_mode.rst
#   frontend .../keyboardShortcuts.sync.test.js  bindings <-> raccourcis.rst
#   go test ./cmd/help-gen                       the in-app help <-> the .rst
#
# What this script checks, none of which any test does:
#
#   1. every subcommand handlers() dispatches is named by `blunderdb help`
#      (printUsage), the only command list a user offline ever sees;
#   2. every one of them has a hand-written section in CLI_USAGE.md, not just
#      a captured --help in its generated flag reference;
#   3. every subcommand a command's own --help advertises is written in
#      cli.rst, so `trash discard` cannot exist only in the binary.
#
# Usage: scripts/doc-inventory.sh        (from anywhere; exit 1 on any gap)

set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

CLI=internal/cli/cli.go
RST=doc/source/cli.rst
USAGE=CLI_USAGE.md

fail=0
note() { printf '  %s\n' "$*"; }
gap() {
    fail=1
    printf '\n\033[31mMANQUE\033[0m %s\n' "$1"
    shift
    for line in "$@"; do note "$line"; done
}

# handlers() is the single source of truth for what counts as a CLI
# invocation (see its comment in cli.go); every list below is compared to it.
commands=$(
    awk '/^func \(cli \*CLI\) handlers\(\)/,/^}/' "$CLI" |
        grep -oE '"[a-z]+":' | tr -d '":'
)
[ -n "$commands" ] || {
    echo "doc-inventory: could not read handlers() from $CLI" >&2
    exit 2
}

# --- 1. `blunderdb help` names every dispatched command ----------------------

usage_names=$(
    awk '/^func \(cli \*CLI\) printUsage\(\)/,/^}/' "$CLI" |
        grep -oE 'fmt\.Println\("  [a-z]+' | awk '{print $NF}'
)

for c in $commands; do
    grep -qx "$c" <<<"$usage_names" || gap \
        "\`blunderdb help\` ne nomme pas la commande \`$c\`" \
        "printUsage() dans $CLI est la seule liste qu'un utilisateur hors ligne voit." \
        "Ajouter la ligne, dans le groupe qui convient."
done

# --- 2. CLI_USAGE.md has a written section, not only a captured --help -------

# `healthcheck` belongs to the headless chapter (it probes a daemon, and the
# CLI page exempts it too); `help` and `version` are the binary answering about
# itself, and printUsage above is their whole documentation.
skip="healthcheck help version"

for c in $commands; do
    case " $skip " in *" $c "*) continue ;; esac
    title="$(tr '[:lower:]' '[:upper:]' <<<"${c:0:1}")${c:1}"
    grep -qiE "^## $title Command" "$USAGE" || gap \
        "$USAGE n'a pas de section rédigée pour \`$c\`" \
        "La référence générée en fin de fichier capture son --help, ce qui n'est" \
        "pas la même chose : elle ne dit ni pourquoi ni quand employer la commande." \
        "Ajouter une section \"## $title Command\" avant la référence générée."
done

# --- 3. every advertised subcommand is written in cli.rst -------------------

# A command with sub-commands states them in one of two shapes, and both are a
# source of truth rather than a restatement: a `<cmd>Handlers()` table (anki,
# bearoff, collection — the same shape as handlers() one level up), or, where
# the dispatch is a switch, the "Subcommands:" block of its own --help (trash).
subcommands_of() {
    local file="$1" cmd="$2"
    awk -v c="$cmd" '$0 ~ "^func \\(cli \\*CLI\\) " c "Handlers\\(\\)" {inside=1; next}
         inside && /^}/ {exit}
         inside' "$file" | grep -oE '"[a-z-]+":' | tr -d '":' || true
    awk '/fmt\.Println\("Subcommands:"\)/{inside=1; next}
         inside && /fmt\.Println\(\)/{exit}
         inside' "$file" |
        grep -oE 'fmt\.Println\("  [a-z][a-z-]*' | awk '{print $NF}' || true
}

for file in internal/cli/cli_*.go; do
    cmd=$(basename "$file" .go)
    cmd=${cmd#cli_}
    grep -qx "$cmd" <<<"$commands" || continue

    subs=$(subcommands_of "$file" "$cmd" | sort -u)
    [ -n "$subs" ] || continue

    # The command's own section of cli.rst, down to the next one.
    section=$(awk -v c="$cmd" '
        $0 ~ "^" c " — " {inside=1}
        inside && /^[a-z]+ — / && $0 !~ "^" c " — " {exit}
        inside {print}
    ' "$RST")
    [ -n "$section" ] || continue

    for sub in $subs; do
        # Three ways the page names a sub-command, all of them legitimate: in a
        # **Sous-commandes:** list (``restore --id N``), as a paragraph heading
        # (**generate.**, which is how bearoff documents its four), or spelled
        # out in an example (`blunderdb bearoff verify`). The name must end
        # where the page says it does — matching the prefix alone let
        # ``discardXX`` pass for `discard`.
        grep -qE "(\`\`$sub(\`\`| )|\*\*$sub[.*]|blunderdb $cmd $sub([^a-z-]|\$))" <<<"$section" || gap \
            "$RST ne décrit pas \`$cmd $sub\`" \
            "La commande la dispatche : elle existe pour l'utilisateur." \
            "L'ajouter à la liste **Sous-commandes:** de la section \`$cmd\`, et traduire les huit .po."
    done
done

if [ "$fail" -eq 0 ]; then
    printf '\033[32mdoc-inventory: rien ne manque\033[0m — %s commandes, leurs sous-commandes et leurs pages.\n' \
        "$(wc -w <<<"$commands")"
else
    printf '\nUne commande qu%s\n' "'aucune page ne nomme est une commande que personne ne trouve."
fi
exit "$fail"
