#!/usr/bin/env bash
# Guards production comments against two decays: history (issue numbers,
# dates, tasks/ references belong in git and ADRs) and citations of tests that
# do not exist. Run from anywhere; exits 1 on any finding.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

files=$(git ls-files '*.go' '*.js' '*.svelte' |
  grep -Ev -e '_test\.go$' -e '\.(test|spec)\.js$' -e '(^|/)node_modules/' \
    -e '(^|/)testdata/' -e '(^|/)(e2e|tests?)/' -e '^frontend/(wailsjs|dist)/' \
    -e '\.pb\.go$' -e '_gen\.go$' -e '(^|/)(bindata|embed)_' || true)

# Prints "file:line:comment" for each comment, minus URLs, //go: and nolint.
extract() {
  # shellcheck disable=SC2016
  xargs awk '
    /^\/\/ Code generated .* DO NOT EDIT\.$/ { skip[FILENAME] = 1 }
    skip[FILENAME] { next }
    {
      c = ""
      if ($0 ~ /^[ \t]*(\/\/|\/\*|\*|<!--)/) c = $0
      else if (match($0, /[ \t]\/\/[ \t]/)) c = substr($0, RSTART)
      else if (match($0, /\/\*.*\*\/|<!--.*-->/)) c = substr($0, RSTART)
      if (c == "") next
      if (c ~ /^[ \t]*\/\/go:/ || c ~ /nolint/) next
      gsub(/https?:\/\/[^ \t)>]*/, "", c)
      print FILENAME ":" FNR ":" c
    }'
}

comments=$(printf '%s\n' "$files" | extract)
fail=0

# 2006-01-02 is Go's reference layout, not a date; &#39; is an HTML entity.
hist=$(printf '%s\n' "$comments" | sed 's/2006-01-02//g' |
  grep -E -e '(^|[^&A-Za-z0-9])#[0-9]{2,4}\b' -e '20[0-9]{2}-[0-9]{2}' -e 'tasks/' || true)
if [ -n "$hist" ]; then
  echo "check-comments: history in comments (issue number, date or tasks/ reference):"
  printf '%s\n' "$hist"
  fail=1
fi

known=$(git ls-files '*_test.go' | xargs grep -hoE '^func (Test|Fuzz)[A-Za-z0-9_]+' |
  awk '{print $2}' | sort -u)
cited=$(printf '%s\n' "$comments" | grep -oE 'Test[A-Z][A-Za-z0-9_]*' | sort -u || true)
missing=""
for t in $cited; do
  grep -qxF "$t" <<<"$known" || missing="$missing $t"
done
if [ -n "$missing" ]; then
  echo "check-comments: comments cite tests that do not exist:"
  for t in $missing; do
    printf '%s\n' "$comments" | grep -E "\b$t\b" | head -3
  done
  fail=1
fi
exit $fail
