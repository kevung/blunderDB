#!/usr/bin/env python3
"""Banc V1.7 : F1 des événements détectés contre les Repères, gain de navigation.

Usage: mesure.py <events.json> <marks.csv> [options]

  events.json  la sortie de `replay(records)` de frontend/src/vision/descriptors.js,
               sérialisée : un tableau (ou {"events": [...]}) d'objets
               {type, mediaTime (secondes), state}.
  marks.csv    la sortie de marks_export.py (n, roll_tick_ms, tick_ms, dés).

Conventions (chacune tenue par un test de test_mesure.py) :
  - un Repère est un instant ; il est apparié à un événement du type voulu dont
    l'instant est à au plus `--tol` ms de (Repère − `--avance`) ; la borne est incluse ;
  - l'appariement est un à un, glouton par écart croissant (égalité : instant du
    Repère, puis de l'événement) ; un événement en trop est un faux positif, un
    Repère sans événement un manque ;
  - `--paire TYPE=COLONNE` (répétable) choisit ce qui se compare à quoi ; par défaut
    dice.rest=roll_tick_ms ; le F1 est calculé par paire puis sur leur réunion ;
  - `--etats` filtre les événements gardés (par défaut tous).

Gain de navigation, modèle explicite : lire les Repères dans l'ordre, du premier au
dernier. Sans marqueur, atteindre le Repère k coûte la traversée r_k − r_(k−1) ; avec
marqueurs, un saut (`--saut` s) plus l'écart résiduel à l'événement apparié, ou la
traversée complète si aucun événement n'est apparié.
"""
import argparse
import csv
import json
import sys

DEFAUT_PAIRE = ("dice.rest", "roll_tick_ms")


def ms(seconds):
    return int(round(seconds * 1000))


def charger_evenements(path, etats=None):
    with open(path, encoding="utf-8") as fh:
        data = json.load(fh)
    if isinstance(data, dict):
        data = data["events"]
    out = []
    for e in data:
        if etats is not None and e.get("state") not in etats:
            continue
        out.append((e["type"], ms(e["mediaTime"])))
    return out


def charger_reperes(path, colonne):
    """Les instants (ms) de `colonne`, dans l'ordre du match ; les vides sont omis."""
    with open(path, newline="", encoding="utf-8") as fh:
        rows = list(csv.DictReader(fh))
    return [int(r[colonne]) for r in rows if r.get(colonne, "") != ""]


def apparier(detectes, reperes, tol_ms=500, avance_ms=0):
    """Appariement un à un, glouton par écart croissant.

    Rend (paires, ndet, nrep) où paires = [(i_detecte, j_repere, ecart_signe_ms)],
    ecart_signe = détecté − (Repère − avance).
    """
    cand = []
    for i, t in enumerate(detectes):
        for j, r in enumerate(reperes):
            ecart = t - (r - avance_ms)
            if abs(ecart) <= tol_ms:
                cand.append((abs(ecart), r, t, i, j, ecart))
    cand.sort()
    pris_i, pris_j, paires = set(), set(), []
    for _, _, _, i, j, ecart in cand:
        if i in pris_i or j in pris_j:
            continue
        pris_i.add(i)
        pris_j.add(j)
        paires.append((i, j, ecart))
    return paires, len(detectes), len(reperes)


def scores(tp, ndet, nrep):
    fp, fn = ndet - tp, nrep - tp
    prec = tp / ndet if ndet else None
    rap = tp / nrep if nrep else None
    f1 = 1.0 if ndet + nrep == 0 else 2 * tp / (ndet + nrep)
    return {"tp": tp, "fp": fp, "fn": fn, "precision": prec, "rappel": rap, "f1": f1}


def mediane(xs):
    if not xs:
        return None
    s = sorted(xs)
    m = len(s) >> 1
    return s[m] if len(s) % 2 else (s[m - 1] + s[m]) / 2


def gain_navigation(reperes, paires, saut_ms=2000):
    """(sans_ms, avec_ms) pour la lecture séquentielle des Repères."""
    apparie = {j: ecart for _, j, ecart in paires}
    sans = avec = 0
    prev = 0
    for j, r in enumerate(reperes):
        travers = max(0, r - prev)
        sans += travers
        avec += saut_ms + abs(apparie[j]) if j in apparie else travers
        prev = r
    return sans, avec


def mesurer(events_path, marks_path, paires_cfg, tol_ms=500, avance_ms=0, etats=None, saut_ms=2000):
    evts = charger_evenements(events_path, etats)
    par_paire, tp_tot, det_tot, rep_tot, ecarts = {}, 0, 0, 0, []
    navs = []
    for typ, col in paires_cfg:
        det = sorted(t for ty, t in evts if ty == typ)
        rep = charger_reperes(marks_path, col)
        paires, nd, nr = apparier(det, rep, tol_ms, avance_ms)
        par_paire[f"{typ}={col}"] = scores(len(paires), nd, nr)
        tp_tot += len(paires)
        det_tot += nd
        rep_tot += nr
        ecarts += [e for _, _, e in paires]
        navs.append((rep, paires))
    out = {
        "tol_ms": tol_ms,
        "avance_ms": avance_ms,
        "paires": par_paire,
        "total": scores(tp_tot, det_tot, rep_tot),
        "ecart_median_ms": mediane(ecarts),
    }
    sans = avec = 0
    for rep, paires in navs:
        s, a = gain_navigation(rep, paires, saut_ms)
        sans += s
        avec += a
    out["navigation"] = {
        "saut_ms": saut_ms,
        "sans_ms": sans,
        "avec_ms": avec,
        "gain": None if sans == 0 else 1 - avec / sans,
    }
    return out


def fmt(x, pct=False):
    if x is None:
        return "n/a"
    return f"{x * 100:.1f} %" if pct else f"{x:.3f}"


def rapport(res):
    lines = ["| paire | TP | FP | FN | précision | rappel | F1 |", "|---|---|---|---|---|---|---|"]
    rows = list(res["paires"].items()) + [("total", res["total"])]
    for nom, s in rows:
        lines.append(
            f"| {nom} | {s['tp']} | {s['fp']} | {s['fn']} | {fmt(s['precision'])} | {fmt(s['rappel'])} | {fmt(s['f1'])} |"
        )
    med = res["ecart_median_ms"]
    lines.append("")
    lines.append(f"tolérance ±{res['tol_ms']} ms, avance {res['avance_ms']} ms ; écart médian (détecté − Repère) : "
                 + ("n/a" if med is None else f"{med:.0f} ms"))
    n = res["navigation"]
    lines.append(
        f"navigation (saut {n['saut_ms']} ms) : sans marqueur {n['sans_ms'] / 1000:.1f} s, "
        f"avec {n['avec_ms'] / 1000:.1f} s, gain {fmt(n['gain'], True)}"
    )
    return "\n".join(lines)


def main(argv):
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("events")
    ap.add_argument("marks")
    ap.add_argument("--tol", type=int, default=500, help="tolérance en ms (500)")
    ap.add_argument("--avance", type=int, default=0, help="avance de l'événement sur le Repère, ms (0)")
    ap.add_argument("--paire", action="append", help="TYPE=COLONNE, répétable (dice.rest=roll_tick_ms)")
    ap.add_argument("--etats", help="états gardés, séparés par des virgules (tous)")
    ap.add_argument("--saut", type=int, default=2000, help="coût d'un saut au marqueur, ms (2000)")
    ap.add_argument("--json", action="store_true")
    a = ap.parse_args(argv[1:])
    cfg = [tuple(p.split("=", 1)) for p in a.paire] if a.paire else [DEFAUT_PAIRE]
    etats = set(a.etats.split(",")) if a.etats else None
    res = mesurer(a.events, a.marks, cfg, a.tol, a.avance, etats, a.saut)
    print(json.dumps(res, indent=1, ensure_ascii=False) if a.json else rapport(res))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
