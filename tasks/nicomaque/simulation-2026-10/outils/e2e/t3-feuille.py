#!/usr/bin/env python3
"""t3-feuille.py — écrit feuilles/T3.json depuis ce que Léa a noté (t3-raw.json), lance l'oracle
dans les deux variantes N26 et compare, joueur par joueur, au classement de l'application.

    python3 t3-feuille.py $SIM_DIR/t3

Les noms viennent de l'écran (fiches de résultat, propositions) ; les identifiants sont ceux de
l'application (journal), pour ne pas dépendre d'une réécriture du slug. Les tirages (poules,
places du tableau) sont ceux relevés au moment du tirage.
"""
import json
import os
import subprocess
import sys

here = os.path.dirname(os.path.abspath(__file__))
root = os.path.dirname(os.path.dirname(here))  # simulation-2026-10
d = sys.argv[1]
raw = json.load(open(os.path.join(d, "t3-raw.json")))
A, B = "Open A", "Suisse B"


def load(n):
    p = os.path.join(d, n)
    return json.load(open(p)) if os.path.exists(p) else None


ids = {}  # nom → id appli
for ep in "AB":
    for e in load("history-%s.json" % ep) or []:
        for k in ("a", "b", "player"):
            nk = {"a": "aName", "b": "bName", "player": "playerName"}[k]
            if e.get(k) and e.get(nk):
                ids[e[nk]] = e[k]
names = {v: k for k, v in ids.items()}


def pid(n):
    return ids.get(n, n)


def phases_of(br):
    out = []
    for ph in br:
        c = ph.get("config") or {}
        p = {"kind": ph["kind"]}
        for k in ("lives", "target", "group_size", "qualifiers", "entry", "seeding"):
            if c.get(k) not in (None, "", 0):
                p[k] = c[k]
        out.append(p)
    return out


def groups_of(br):
    ph = br[0]
    secs = [s for s in ph["sections"] if s.get("kind") == "poule"]
    secs.sort(key=lambda s: s.get("group", 0))
    out = []
    for s in secs:
        g = s.get("players") or []
        for m in s["matches"]:
            for k in ("a", "b"):
                if m.get(k) and m[k] not in g:
                    g.append(m[k])
        out.append(g)
    return out


def slots_of(br, idx):
    ph = br[idx]
    main = [s for s in ph["sections"] if s["kind"] in ("main", "bracket") or s["name"] in ("main", "Tableau principal")] or ph["sections"]
    r0 = [m for m in main[0]["matches"] if m["round"] == 0]
    r0.sort(key=lambda m: int(m["key"].split(".")[-1]))
    slots = []
    for m in r0:
        slots += [None if m.get(k) in (None, "", "BYE") else m[k] for k in ("a", "b")]
    return slots


def repechage(sheet):
    """Le suivant non retiré de la poule du qualifié retiré (plus de victoires de poule)."""
    out = [e["player"] for e in sheet["events"] if e["type"] == "withdraw"]
    groups = next((e["groups"] for e in sheet["events"] if e.get("groups")), [])
    wins = {}
    for e in sheet["events"]:
        if e["type"] == "match" and e["phase"] == 0:
            wins[e["winner"]] = wins.get(e["winner"], 0) + 1
    for g in groups:
        if any(o in g for o in out):
            ranked = sorted((p for p in g if p not in out), key=lambda p: -wins.get(p, 0))
            return ranked[1] if len(ranked) > 1 else None  # ranked[0] = l'autre qualifié
    return None


def build(ep, n26):
    evs = [e for e in raw["events"] if e["ep"] == ep]
    draws = {k: v for k, v in raw["draws"].items() if k.startswith(ep + "#") and isinstance(v, list)}
    last = draws[max(draws, key=lambda k: int(k.split("#")[1]))]
    roster = [p for p in (load("history-%s.json" % ("A" if ep == A else "B")) or []) if p.get("kind") == "player_added"]
    sheet = {"name": ep, "n26": n26, "players": [{"id": p["player"], "name": p["playerName"]} for p in roster],
             "phases": phases_of(last), "events": []}
    phase = 0
    pending_rep = {}
    for i, e in enumerate(evs):
        t = e["type"]
        if t == "match":
            sheet["events"].append({"type": "match", "phase": phase, "a": pid(e["a"]), "b": pid(e["b"]), "winner": pid(e["winner"])})
        elif t == "repechage":
            # Noté au passage de phase : le repêché de la poule du retiré (« poule:A » → « Poule A »).
            pending_rep["Poule " + e["pool"].split(":")[-1]] = pid(e["player"])
        elif t == "withdraw":
            sheet["events"].append({"type": "withdraw", "player": pid(e["player"])})
        elif t == "bye":
            who = e["label"].split(":")[-1].strip()
            sheet["events"].append({"type": "bye", "player": pid(who)})
        elif t == "draw" and "arrage" in e.get("label", ""):
            continue  # barrage de poule : déduit par l'oracle
        elif t in ("next_phase", "draw"):
            if t == "next_phase":
                phase += 1
                ev = {"type": "next_phase"}
                if pending_rep and n26 == "repechage":
                    ev["repechage"] = dict(pending_rep)
                pending_rep.clear()
                sheet["events"].append(ev)
            # le tirage relevé juste après ce lancement
            key = min((k for k in draws if int(k.split("#")[1]) > raw["events"].index(e) and (t == "next_phase" or True)), key=lambda k: int(k.split("#")[1]), default=None)
            br = draws.get(key)
            if not br or phase >= len(br) or not br[phase].get("drawn"):
                continue
            if br[phase]["kind"] == "round_robin":
                if any(x.get("type") == "draw" and "groups" in x for x in sheet["events"]):
                    continue
                sheet["events"].append({"type": "draw", "groups": groups_of(br)})
            elif br[phase]["kind"] != "swiss_lives":
                if not any(x.get("type") == "draw" and "slots" in x for x in sheet["events"]):
                    slots = slots_of(br, phase)
                    rep = repechage(sheet) if n26 == "repechage" and None in slots else None
                    if rep and None in slots:
                        # Ce que Léa voulait : le repêché à la place laissée vide par le retiré.
                        slots[slots.index(None)] = rep
                        sheet["n26_note"] = "place du retiré donnée à %s (voulu par la directrice ; l'appli a laissé une exemption)" % names.get(rep, rep)
                    sheet["events"].append({"type": "draw", "slots": slots})
    return sheet


def oracle(sheet_path):
    r = subprocess.run([sys.executable, os.path.join(root, "outils", "oracle", "oracle.py"), sheet_path], capture_output=True, text=True)
    if r.returncode:
        return {"error": (r.stdout + r.stderr)[-1500:]}
    return json.loads(r.stdout)


def app_ranks(ep):
    st = load("standings-%s.json" % ("A" if ep == A else "B"))
    secs = (st or {}).get("sections") or []
    rows = next((x["rows"] for x in secs if not x.get("name")), secs[0]["rows"] if secs else [])
    out = {}
    for r in rows:
        p = r.get("id") or r.get("player")
        out[p] = r.get("rank")
    return out, st


repA = build(A, "repechage")
# Si l'appli a laissé une exemption là où Léa voulait le repêché (n26_note), le tableau voulu
# diffère dès le 1er tour : les matchs joués à l'écran ne se rattachent plus et la variante
# s'arrête au tirage. Quand l'appli a repêché elle-même, la feuille se rejoue jusqu'au bout.
if "n26_note" in repA:
    cut = next(i for i, e in enumerate(repA["events"]) if e["type"] == "draw" and "slots" in e)
    repA["events_tableau_non_rejouables"] = len([e for e in repA["events"][cut + 1:] if e["type"] == "match"])
    repA["events"] = repA["events"][:cut + 1]
motA = build(A, "moteur")
feuille = {"rencontre": "Rencontre du club", "epreuves": [repA, build(B, "repechage")], "variante_moteur": motA}
os.makedirs(os.path.join(root, "feuilles"), exist_ok=True)
fp = os.path.join(root, "feuilles", "T3.json")
json.dump(feuille, open(fp, "w"), ensure_ascii=False, indent=1)
moteur = {"rencontre": "Rencontre du club", "epreuves": [motA, feuille["epreuves"][1]]}
mp = os.path.join(d, "T3-moteur.json")
json.dump(moteur, open(mp, "w"), ensure_ascii=False, indent=1)

report = {}
# La variante « moteur » (place du retiré perdue) n'a de sens que si l'appli n'a pas repêché.
for variant, path in (("repechage", fp), ("moteur", mp))[: 2 if "n26_note" in repA else 1]:
    res = oracle(path)
    report[variant] = res
    json.dump(res, open(os.path.join(d, "oracle-%s.json" % variant), "w"), ensure_ascii=False, indent=1)

for ep_i, ep in enumerate((A, B)):
    app, st = app_ranks(ep)
    print("==", ep, "app:", len(app), "joueurs")
    for variant in report:
        res = report[variant]
        if "error" in res:
            print(variant, "ERREUR", res["error"][-600:])
            continue
        e = res["epreuves"][ep_i] if "epreuves" in res else res
        orc = {r["player"]: r["rank"] for r in e["final"]}
        diff = [(names.get(p, p), orc.get(p), app.get(p)) for p in sorted(set(orc) | set(app)) if orc.get(p) != app.get(p)]
        print(variant, "identique" if not diff else "ÉCARTS %d" % len(diff), diff[:16], "warnings:", e.get("warnings"))
        if ep == A:
            q = e["phases"][0].get("qualified")
            print("   qualifiés oracle :", [names.get(p, p) for p in q])
