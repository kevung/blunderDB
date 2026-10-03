#!/usr/bin/env bash
# sample-corpus.sh — draw a reproducible random sample of match files from a
# corpus too large to import whole, for measuring import throughput,
# duplicates and name spellings on real data.
#
#   scripts/sample-corpus.sh <folder|archive.zip> <n> <output-dir> [seed]
#
# The source is either a folder (searched recursively) or a .zip archive read
# in place, so a 20 GB corpus never has to be extracted to be sampled. Only
# .xg files are drawn. The same source, n and seed always give the same
# sample: the candidate list is sorted, then shuffled by `shuf` fed with a
# keystream derived from the seed (the GNU coreutils recipe for repeatable
# shuffles), so the draw does not depend on filesystem or archive order.
#
# The sample is copied flat into <output-dir>, each file renamed
# NNNNN-<basename> so that two files with the same name in different folders
# do not collide; <output-dir>/ORIGINES.tsv maps every copy to its original
# path. Keep <output-dir> outside the repository: a corpus carries real
# player names, and nothing of it belongs in git.
set -euo pipefail

usage() { echo "usage: $0 <folder|archive.zip> <n> <output-dir> [seed]" >&2; exit 2; }
[ $# -ge 3 ] || usage
src="$1"; n="$2"; out="$3"; seed="${4:-518}"
case "$n" in ''|*[!0-9]*) usage ;; esac

seeded_stream() {
  openssl enc -aes-256-ctr -pass pass:"$1" -nosalt -pbkdf2 </dev/zero 2>/dev/null
}

mkdir -p "$out"
list="$out/.candidats"
if [ -d "$src" ]; then
  (cd "$src" && find . -type f -iname '*.xg' | sed 's|^\./||') | LC_ALL=C sort >"$list"
elif [ -f "$src" ] && unzip -Z1 "$src" >/dev/null 2>&1; then
  unzip -Z1 "$src" | grep -i '\.xg$' | LC_ALL=C sort >"$list"
else
  echo "sample-corpus: $src is neither a folder nor a zip archive" >&2
  exit 1
fi

total=$(wc -l <"$list")
if [ "$total" -lt "$n" ]; then
  echo "sample-corpus: only $total .xg files in $src, fewer than $n" >&2
  exit 1
fi
shuf --random-source=<(seeded_stream "$seed") -n "$n" "$list" >"$out/.tirage"

printf 'copie\torigine\n' >"$out/ORIGINES.tsv"
if [ -d "$src" ]; then
  i=0
  while IFS= read -r rel; do
    i=$((i + 1))
    flat=$(printf '%05d-%s' "$i" "$(basename "$rel")")
    cp -- "$src/$rel" "$out/$flat"
    printf '%s\t%s\n' "$flat" "$rel" >>"$out/ORIGINES.tsv"
  done <"$out/.tirage"
else
  # unzip reads its arguments as wildcards: escape the wildcard characters a
  # file name may carry, then extract in batches into a staging folder that
  # keeps the archive's paths, and only then flatten.
  stage="$out/.extraction"
  mkdir -p "$stage"
  sed 's/[][*?\\]/\\&/g' "$out/.tirage" | tr '\n' '\0' |
    xargs -0 -n 200 unzip -qq -o "$src" -d "$stage"
  i=0
  while IFS= read -r rel; do
    i=$((i + 1))
    flat=$(printf '%05d-%s' "$i" "$(basename "$rel")")
    mv -- "$stage/$rel" "$out/$flat"
    printf '%s\t%s\n' "$flat" "$rel" >>"$out/ORIGINES.tsv"
  done <"$out/.tirage"
  rm -rf "$stage"
fi
rm -f "$list" "$out/.tirage"
echo "sample-corpus: $n of $total files from $src (seed $seed) -> $out"
