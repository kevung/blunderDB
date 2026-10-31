#!/usr/bin/env python3
"""Banc V1.7 : F1 des événements détectés contre les Repères pris comme fenêtres.

Usage: mesure.py <events.json> <marks.csv> [options]

  events.json  la sortie de `replay(records)` de frontend/src/vision/descriptors.js,
               sérialisée : un tableau (ou {"events": [...]}) d'objets
               {type, mediaTime (secondes), state}.
  marks.csv    la sortie de marks_export.py (n, roll_tick_ms, tick_ms, dés, prev_tick_ms).

Conventions (chacune tenue par un test de test_mesure.py, et par une mutation) :
  - un Repère est une FENÊTRE (bancs/02-couverture-sql.md § 4) : celle du jet va du
    tick_ms de l'action précédente de la partie (prev_tick_ms) à roll_tick_ms ; celle de
    l'action va de roll_tick_ms à tick_ms (de prev_tick_ms si le coup n'a pas de jet) ;
  - un événement du type voulu est un vrai positif s'il tombe dans la fenêtre élargie de
    `--tol` ms de chaque côté, bornes incluses ;
  - une fenêtre sans borne de début, ou à rebours, n'est pas évaluable : elle sort du
    rappel (comptée à part, `sans_fenetre`) et les événements qu'elle aurait portés sont
    des faux positifs, donc la précision est alors un minorant ;
  - l'appariement est un à un, glouton par distance croissante à la fenêtre (0 dedans) ;
    un événement en trop est un faux positif, une fenêtre sans événement un manque ;
  - `--paire TYPE=COLONNE` (répétable) choisit ce qui se compare à quoi ; par défaut
    dice.rest=roll_tick_ms ; le F1 est calculé par paire puis sur leur réunion ;
  - `--etats` filtre les événements gardés (par défaut tous).

Gain de navigation, modèle explicite : les Repères de toutes les paires forment une seule
suite lue dans l'ordre du temps. Sans marqueur, atteindre un Repère coûte la traversée
depuis le précédent. Avec marqueurs, une fenêtre appariée coûte un saut (`--saut` s) plus la
distance de l'événement à la fenêtre ; une fenêtre sans événement coûte la traversée ; chaque
faux positif coûte un saut de plus (le marqueur est visité pour rien).
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


def _cell(r, col):
    v = r.get(col, "")
    return None if v in ("", None) else int(v)


def charger_fenetres(path, colonne):
    """Les fenêtres (debut, fin) en ms de `colonne`, dans l'ordre du match.

    debut est None quand la fenêtre n'est pas évaluable (voir l'en-tête).
    """
    with open(path, newline="", encoding="utf-8") as fh:
        rows = list(csv.DictReader(fh))
    out = []
    for r in rows:
        fin = _cell(r, colonne)
        if fin is None:
            continue
        roll, prev = _cell(r, "roll_tick_ms"), _cell(r, "prev_tick_ms")
        debut = roll if colonne == "tick_ms" and roll is not None else prev
        if debut is not None and debut > fin:
            debut = None
        out.append((debut, fin))
    return out


def apparier(detectes, fenetres, tol_ms=500):
    """Appariement un à un, glouton par distance croissante à la fenêtre.

    fenetres = [(debut, fin)]. Rend (paires, ndet, nrep) où paires =
    [(i_detecte, j_fenetre, distance_ms)], distance = 0 dans [debut, fin], sinon l'écart à la
    borne la plus proche ; seules les distances <= tol_ms sont candidates.
    """
    cand = []
    for i, t in enumerate(detectes):
        for j, (debut, fin) in enumerate(fenetres):
            dist = max(0, debut - t, t - fin)
            if dist <= tol_ms:
                cand.append((dist, fin, t, i, j))
    cand.sort()
    pris_i, pris_j, paires = set(), set(), []
    for dist, _, _, i, j in cand:
        if i in pris_i or j in pris_j:
            continue
        pris_i.add(i)
        pris_j.add(j)
        paires.append((i, j, dist))
    return paires, len(detectes), len(fenetres)


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


def gain_navigation(cibles, faux_positifs, saut_ms=2000):
    """(sans_ms, avec_ms) pour la lecture dans l'ordre du temps.

    cibles = [(fin_ms, distance_ms ou None)] : le Repère et, s'il est apparié, la distance de
    l'événement à sa fenêtre ; faux_positifs = nombre d'événements non appariés.
    """
    sans = avec = 0
    prev = 0
    for fin, dist in sorted(cibles, key=lambda c: c[0]):
        travers = max(0, fin - prev)
        sans += travers
        avec += travers if dist is None else saut_ms + dist
        prev = max(prev, fin)
    return sans, avec + faux_positifs * saut_ms


def mesurer(events_path, marks_path, paires_cfg, tol_ms=500, etats=None, saut_ms=2000):
    evts = charger_evenements(events_path, etats)
    par_paire, tp_tot, det_tot, rep_tot, dists = {}, 0, 0, 0, []
    cibles, fp_tot, sans_fenetre = [], 0, 0
    for typ, col in paires_cfg:
        det = sorted(t for ty, t in evts if ty == typ)
        toutes = charger_fenetres(marks_path, col)
        evaluables = [w for w in toutes if w[0] is not None]
        sans_fenetre += len(toutes) - len(evaluables)
        paires, nd, nr = apparier(det, evaluables, tol_ms)
        par_paire[f"{typ}={col}"] = scores(len(paires), nd, nr)
        tp_tot += len(paires)
        det_tot += nd
        rep_tot += nr
        dists += [d for _, _, d in paires]
        fp_tot += nd - len(paires)
        dist_de = {j: d for _, j, d in paires}
        cibles += [(w[1], dist_de.get(j)) for j, w in enumerate(evaluables)]
        cibles += [(w[1], None) for w in toutes if w[0] is None]
    sans, avec = gain_navigation(cibles, fp_tot, saut_ms)
    return {
        "tol_ms": tol_ms,
        "paires": par_paire,
        "total": scores(tp_tot, det_tot, rep_tot),
        "sans_fenetre": sans_fenetre,
        "distance_mediane_ms": mediane(dists),
        "navigation": {
            "saut_ms": saut_ms,
            "sans_ms": sans,
            "avec_ms": avec,
            "gain": None if sans == 0 else 1 - avec / sans,
        },
    }


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
    med = res["distance_mediane_ms"]
    lines.append("")
    lines.append(f"tolérance ±{res['tol_ms']} ms ; fenêtres non évaluables : {res['sans_fenetre']} ; "
                 "distance médiane à la fenêtre (appariés) : " + ("n/a" if med is None else f"{med:.0f} ms"))
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
    ap.add_argument("--paire", action="append", help="TYPE=COLONNE, répétable (dice.rest=roll_tick_ms)")
    ap.add_argument("--etats", help="états gardés, séparés par des virgules (tous)")
    ap.add_argument("--saut", type=int, default=2000, help="coût d'un saut au marqueur, ms (2000)")
    ap.add_argument("--json", action="store_true")
    a = ap.parse_args(argv[1:])
    cfg = [tuple(p.split("=", 1)) for p in a.paire] if a.paire else [DEFAUT_PAIRE]
    etats = set(a.etats.split(",")) if a.etats else None
    res = mesurer(a.events, a.marks, cfg, a.tol, etats, a.saut)
    print(json.dumps(res, indent=1, ensure_ascii=False) if a.json else rapport(res))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
