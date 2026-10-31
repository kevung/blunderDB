#!/usr/bin/env python3
"""Test par mutation de mesure.py : chaque convention, altérée, doit faire échouer test_mesure.py.

Usage: mutations.py   (depuis n'importe où ; n'écrit que dans un dossier temporaire)
Code de sortie 1 si une mutation survit.
"""
import os
import shutil
import subprocess
import sys
import tempfile

HERE = os.path.dirname(os.path.abspath(__file__))
MUTANTS = [
    ("tolérance exclusive", "if abs(ecart) <= tol_ms:", "if abs(ecart) < tol_ms:"),
    ("tolérance élargie", "if abs(ecart) <= tol_ms:", "if abs(ecart) <= tol_ms + 1:"),
    ("pas d'unicité côté événement", "if i in pris_i or j in pris_j:", "if j in pris_j:"),
    ("pas d'unicité côté Repère", "if i in pris_i or j in pris_j:", "if i in pris_i:"),
    ("ordre d'arrivée au lieu du plus proche", "cand.sort()", "cand.sort(key=lambda c: (c[2], c[1]))"),
    ("avance ignorée", "ecart = t - (r - avance_ms)", "ecart = t - r"),
    ("avance de signe inverse", "ecart = t - (r - avance_ms)", "ecart = t - (r + avance_ms)"),
    ("écart non signé", "paires.append((i, j, ecart))", "paires.append((i, j, abs(ecart)))"),
    ("filtre de type absent", "ty == typ", "True"),
    ("filtre d'état absent", 'e.get("state") not in etats', "False"),
    ("arrondi tronqué", "int(round(seconds * 1000))", "int(seconds * 1000)"),
    ("cellule vide comptée", 'if r.get(colonne, "") != ""', "if True"),
    ("F1 sans manque ni excès", "2 * tp / (ndet + nrep)", "2 * tp / (ndet + nrep) if tp < 99 else 0"),
    ("F1 vide nul", "1.0 if ndet + nrep == 0", "0.0 if ndet + nrep == 0"),
    ("navigation : non apparié gratuit", "else travers", "else 0"),
    ("navigation : résiduel ignoré", "saut_ms + abs(apparie[j])", "saut_ms"),
    ("navigation : saut ignoré", "saut_ms + abs(apparie[j])", "abs(apparie[j])"),
]


def main():
    src = open(os.path.join(HERE, "mesure.py"), encoding="utf-8").read()
    survivors = 0
    for nom, old, new in MUTANTS:
        if src.count(old) != 1:
            print(f"SANS PRISE  {nom}: motif introuvable ou ambigu")
            survivors += 1
            continue
        with tempfile.TemporaryDirectory() as d:
            shutil.copytree(os.path.join(HERE, "fixtures"), os.path.join(d, "fixtures"))
            shutil.copy(os.path.join(HERE, "test_mesure.py"), d)
            with open(os.path.join(d, "mesure.py"), "w", encoding="utf-8") as fh:
                fh.write(src.replace(old, new))
            r = subprocess.run([sys.executable, "-I", "-m", "unittest", "test_mesure"], cwd=d, capture_output=True)
        tue = r.returncode != 0
        print(("tué       " if tue else "SURVIT    ") + nom)
        survivors += not tue
    return 1 if survivors else 0


if __name__ == "__main__":
    sys.exit(main())
