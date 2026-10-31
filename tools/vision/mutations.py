#!/usr/bin/env python3
"""Test par mutation de mesure.py (test_mesure.py) et de readable_hist.py / prepare_annot.py (test_annot.py) :
chaque convention, altérée, doit faire échouer sa suite.

Usage: mutations.py   (depuis n'importe où ; n'écrit que dans un dossier temporaire)
Code de sortie 1 si une mutation survit.
"""
import os
import shutil
import subprocess
import sys
import tempfile

HERE = os.path.dirname(os.path.abspath(__file__))
# (source, suite, nom, motif, remplacement)
MUTANTS_ANNOT = [
    ("readable_hist.py", "signe inversé (roll - lisible)", "parse_time(t) - roll[n]", "roll[n] - parse_time(t)"),
    ("readable_hist.py", "unité : ms lus comme s", "parse_time(t) - roll[n]", "parse_time(t) * 1000 - roll[n]"),
    ("readable_hist.py", "affichage en ms au lieu de s", "s['median'] / 1000:+.2f", "s['median']:+.2f"),
    ("readable_hist.py", "quantile sans interpolation", "(sorted_vals[hi] - sorted_vals[lo]) * (k - lo)", "0"),
    ("readable_hist.py", "rang des quantiles : n au lieu de n-1", "(len(sorted_vals) - 1) * p / 100", "len(sorted_vals) * p / 100"),
    ("readable_hist.py", "quantiles sans tri", "s = sorted(deltas)", "s = list(deltas)"),
    ("readable_hist.py", "p10 prend le p90", '"p10": percentile(s, 10)', '"p10": percentile(s, 90)'),
    ("readable_hist.py", "borne basse incluse comme dehors", "d < LOW_MS", "d <= LOW_MS"),
    ("readable_hist.py", "borne haute incluse comme dehors", "d > HIGH_MS", "d >= HIGH_MS"),
    ("readable_hist.py", "borne basse ignorée", "d < LOW_MS or ", ""),
    ("readable_hist.py", "borne haute ignorée", " or d > HIGH_MS", ""),
    ("readable_hist.py", "fenêtre basse = -2 s", "LOW_MS, HIGH_MS = -1000, 2000", "LOW_MS, HIGH_MS = -2000, 2000"),
    ("readable_hist.py", "fenêtre haute = +3 s", "LOW_MS, HIGH_MS = -1000, 2000", "LOW_MS, HIGH_MS = -1000, 3000"),
    ("readable_hist.py", "classes arrondies au lieu de plancher", "math.floor(d / BIN_MS)] = counts.get(math.floor(d / BIN_MS), 0)", "round(d / BIN_MS)] = counts.get(round(d / BIN_MS), 0)"),
    ("readable_hist.py", "cellule vide comptée", "if not t or", "if t == '' and False or"),
    ("prepare_annot.py", "sélection : premiers coups au lieu d'étalée", "round(i * (len(rows) - 1) / (n - 1))", "i"),
    ("prepare_annot.py", "sélection : dernier coup omis", "(len(rows) - 1) / (n - 1)", "len(rows) / n"),
    ("prepare_annot.py", "horloge : minutes en secondes", "ms // 60000", "ms // 1000"),
    ("prepare_annot.py", "horloge : ms tronquées", "ms % 1000:03d", "ms % 100:03d"),
    ("prepare_annot.py", "analyse : minute à 100 s", "whole * 60 + int(p)", "whole * 100 + int(p)"),
    ("prepare_annot.py", "analyse : secondes prises pour ms", "int(round(secs * 1000))", "int(round(secs))"),
    ("prepare_annot.py", "extrait : début non borné à 0", "max(0, roll_ms - back_ms)", "roll_ms - back_ms"),
    ("prepare_annot.py", "extrait : retour en arrière ignoré", "roll_ms - back_ms)", "roll_ms)"),
    ("prepare_annot.py", "extrait : durée ignorée", "-t {span_ms / 1000:.3f}", "-t 1"),
]

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


def run_annot():
    survivors = 0
    for fichier, nom, old, new in MUTANTS_ANNOT:
        src = open(os.path.join(HERE, fichier), encoding="utf-8").read()
        if src.count(old) != 1:
            print(f"SANS PRISE  {nom}: motif introuvable ou ambigu")
            survivors += 1
            continue
        with tempfile.TemporaryDirectory() as d:
            shutil.copytree(os.path.join(HERE, "fixtures"), os.path.join(d, "fixtures"))
            for f in ("readable_hist.py", "prepare_annot.py", "test_annot.py"):
                shutil.copy(os.path.join(HERE, f), d)
            with open(os.path.join(d, fichier), "w", encoding="utf-8") as fh:
                fh.write(src.replace(old, new))
            r = subprocess.run([sys.executable, "-I", "-m", "unittest", "test_annot"], cwd=d, capture_output=True)
        tue = r.returncode != 0
        print(("tué       " if tue else "SURVIT    ") + nom)
        survivors += not tue
    return survivors


def main():
    survivors = run_annot()
    src = open(os.path.join(HERE, "mesure.py"), encoding="utf-8").read()
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
