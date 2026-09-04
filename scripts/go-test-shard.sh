#!/usr/bin/env bash
# go-test-shard.sh — exécute une tranche des tests d'un paquet Go.
#
# Usage : scripts/go-test-shard.sh <paquet> <index> <total> [flags go test…]
#   scripts/go-test-shard.sh ./pkg/blunderdb/database/ 0 2 -race -count=1
#
# Pourquoi (E.3, #219) : `pkg/blunderdb/database` porte à lui seul plus de
# vingt minutes de tests instrumentés par le détecteur de courses, davantage
# que le budget d'un job. Il était coupé en deux par la PREMIÈRE LETTRE du nom
# de test (`-run '^Test[A-I]'` / `'^Test[^A-I]'`), ce qui a deux défauts : les
# deux moitiés n'ont aucune raison d'être équilibrées — elles pesaient 279 s
# et 419 s — et renommer un test déplace silencieusement de la charge d'un
# job à l'autre.
#
# Ici la tranche est décidée par l'INDEX du test dans la liste triée que
# `go test -list` donne : chaque tranche reçoit un test sur `total`, donc la
# répartition ne dépend plus de l'orthographe des noms, et ajouter un test le
# range dans la tranche suivante sans rien déséquilibrer.
#
# `go test -list` ne compile que le paquet de test ; il n'exécute rien, et
# n'imprime que les noms (plus la ligne « ok … » finale, écartée par le grep).
set -euo pipefail

pkg=${1:?paquet manquant}
index=${2:?index de tranche manquant}
total=${3:?nombre de tranches manquant}
shift 3

if [ "$index" -ge "$total" ] || [ "$index" -lt 0 ]; then
    echo "index de tranche $index hors de [0, $total)" >&2
    exit 2
fi

# `-list` filtre aussi les Fuzz* et Example*, qui doivent tomber dans une
# tranche comme les autres plutôt que d'être oubliés par un motif trop étroit.
names=$(go test -list '.*' "$pkg" | grep -E '^(Test|Fuzz|Example)' | sort)
if [ -z "$names" ]; then
    echo "aucun test trouvé dans $pkg" >&2
    exit 1
fi

slice=$(printf '%s\n' "$names" | awk -v n="$total" -v i="$index" '(NR - 1) % n == i')
if [ -z "$slice" ]; then
    echo "tranche $index/$total vide pour $pkg" >&2
    exit 1
fi

count=$(printf '%s\n' "$slice" | wc -l)
echo "tranche $index/$total : $count test(s) sur $(printf '%s\n' "$names" | wc -l)"

pattern=$(printf '%s\n' "$slice" | paste -sd'|')
exec go test -run "^(${pattern})\$" "$@" "$pkg"
