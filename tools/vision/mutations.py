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
    ("tolérance exclusive", "if dist <= tol_ms:", "if dist < tol_ms:"),
    ("tolérance élargie", "if dist <= tol_ms:", "if dist <= tol_ms + 1:"),
    ("borne de début ignorée (fenêtre = point de fin)", "max(0, debut - t, t - fin)", "abs(t - fin)"),
    ("borne de fin ignorée (fenêtre = point de début)", "max(0, debut - t, t - fin)", "abs(debut - t)"),
    ("borne de début non élargie", "max(0, debut - t, t - fin)", "max(0, t - fin)"),
    ("borne de fin non élargie", "max(0, debut - t, t - fin)", "max(0, debut - t)"),
    ("début de la fenêtre du jet = jet moins 3 s", "debut = roll if colonne == \"tick_ms\" and roll is not None else prev",
     "debut = roll if colonne == \"tick_ms\" and roll is not None else (fin - 3000)"),
    ("début de la fenêtre de l'action = action précédente", "debut = roll if colonne == \"tick_ms\" and roll is not None else prev", "debut = prev"),
    ("fenêtre à rebours acceptée", "if debut is not None and debut > fin:", "if False:"),
    ("pas d'unicité côté événement", "if i in pris_i or j in pris_j:", "if j in pris_j:"),
    ("pas d'unicité côté fenêtre", "if i in pris_i or j in pris_j:", "if i in pris_i:"),
    ("ordre d'arrivée au lieu du plus proche", "cand.sort()", "cand.sort(key=lambda c: (c[1], c[2]))"),
    ("filtre de type absent", "ty == typ", "True"),
    ("filtre d'état absent", 'e.get("state") not in etats', "False"),
    ("arrondi tronqué", "int(round(seconds * 1000))", "int(seconds * 1000)"),
    ("cellule vide comptée", 'None if v in ("", None) else int(v)', "int(v or 0)"),
    ("fenêtre non évaluable comptée dans le rappel", "evaluables = [w for w in toutes if w[0] is not None]", "evaluables = [(w[0] or 0, w[1]) for w in toutes]"),
    ("F1 faux", "2 * tp / (ndet + nrep)", "2 * tp / (ndet + nrep) if tp < 99 else 0"),
    ("F1 vide nul", "1.0 if ndet + nrep == 0", "0.0 if ndet + nrep == 0"),
    ("navigation : non apparié gratuit", "avec += travers if dist is None", "avec += 0 if dist is None"),
    ("navigation : distance ignorée", "saut_ms + dist", "saut_ms"),
    ("navigation : saut ignoré", "saut_ms + dist", "dist"),
    ("navigation : faux positif gratuit", "avec + faux_positifs * saut_ms", "avec"),
    ("navigation : traversée repartant de 0 à chaque paire", "cibles += [(w[1], dist_de.get(j)) for j, w in enumerate(evaluables)]",
     "cibles += [(w[1] + 10**9 * len(par_paire), dist_de.get(j)) for j, w in enumerate(evaluables)]"),
    ("navigation : ordre du temps ignoré", "sorted(cibles, key=lambda c: c[0])", "cibles"),
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
