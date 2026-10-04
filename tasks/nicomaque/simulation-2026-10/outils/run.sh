#!/usr/bin/env bash
# Lance une spec de la simulation : ./run.sh e2e/smoke.spec.js
# Relie node_modules (ignoré par git) depuis le dépôt principal si le worktree n'en a pas.
set -euo pipefail
here=$(cd "$(dirname "$0")" && pwd)
repo=$(cd "$here/../../../.." && pwd)
main=$(git -C "$repo" worktree list --porcelain | awk '/^worktree /{print $2; exit}')
[ -e "$repo/frontend/node_modules" ] || ln -s "$main/frontend/node_modules" "$repo/frontend/node_modules"
[ -e "$here/node_modules" ] || ln -s "$repo/frontend/node_modules" "$here/node_modules"
export BLUNDERDB_E2E_PORT=${BLUNDERDB_E2E_PORT:-5183}
export SIM_DIR=${SIM_DIR:-/tmp/blunderdb-sim}
mkdir -p "$SIM_DIR"
cd "$here"
exec npx playwright test -c playwright.config.js "$@"
