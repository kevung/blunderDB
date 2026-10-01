#!/usr/bin/env bash
# Résout les conflits de documentation d'une fusion de branche de fonctionnalité.
#
# Pourquoi ce script existe : deux branches qui ajoutent chacune des raccourcis
# produisent le même conflit à chaque fois — l'union du .rst, puis huit
# catalogues et neuf bundles d'aide qui sont des ARTEFACTS et n'ont pas à être
# résolus ligne à ligne. On les régénère, et on repose les traductions des deux
# côtés de la fusion par msgid EXACT, drapeau « fuzzy » compris.
#
# Ce que le script NE fait pas : résoudre le code. Les conflits de composants,
# de services ou de tests se lisent et se tranchent à la main, et se vérifient
# par les deux suites avant le commit.
#
# Usage : scripts/merge-resolve-docs.sh   (pendant une fusion en conflit)
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

# 1. Le .rst : union des deux ajouts, dans l'ordre HEAD puis branche.
python3 - <<'PY'
import re, pathlib, subprocess
files = subprocess.run(['git','diff','--diff-filter=U','--name-only'],
                       capture_output=True, text=True).stdout.split()
for f in files:
    if not f.endswith('.rst'): continue
    p = pathlib.Path(f); s = p.read_text(encoding='utf-8')
    s2 = re.sub(r'<<<<<<< HEAD\n(.*?)=======\n(.*?)>>>>>>> [^\n]+\n',
                lambda m: m.group(1) + m.group(2), s, flags=re.S)
    assert '<<<<<<<' not in s2, f
    p.write_text(s2, encoding='utf-8')
    print('rst union :', f)
PY

# 2. Les artefacts : on prend un côté, ils seront régénérés juste après.
for f in $(git diff --diff-filter=U --name-only | grep -E '\.po$|i18n/help/.*\.js$' || true); do
    git checkout --ours "$f" 2>/dev/null || true
done
git add -A doc/source frontend/src/i18n/help 2>/dev/null || true

# 3. Régénération des catalogues depuis les sources fusionnées.
source .venv/bin/activate
scripts/doc-po-update.sh >/dev/null

# 4. Les traductions des deux côtés, reposées par msgid exact (vides ET fuzzy).
python3 scripts/merge-carry-translations.py /tmp/merge-fill.json
scripts/po-fill.py /tmp/merge-fill.json >/dev/null
scripts/doc-i18n-check.sh | tail -1

# 5. L'aide embarquée, régénérée depuis la doc fusionnée.
GOTMPDIR="${GOTMPDIR:-$HOME/.cache/gotmp}" make help >/dev/null
echo "doc résolue ; il reste le code, à trancher à la main et à vérifier par les tests."
