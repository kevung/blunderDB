#!/usr/bin/env python3
"""Relevé du coût en tokens des sessions Claude Code sur ce dépôt (ADR-0055).

    scripts/cout-tokens.py [--depuis AAAA-MM-JJ] [--jusqu-a AAAA-MM-JJ]

Rend un verdict d'une vingtaine de lignes, à consigner dans tasks/cout-journal.md.
Le coût « pondéré » applique les tarifs relatifs (entrée 1, écriture de cache 1,25,
lecture de cache 0,1, sortie 5). La relecture par catégorie est une estimation : chaque
sortie d'outil pèse sa taille × le nombre de tours qui la relisent ensuite.
"""
import argparse, collections, glob, json, os, re, statistics, subprocess

PLAFOND = 150
CATEGORIES = [
    ("tests", r"go test|vitest|npm (run )?test|playwright|golangci|go vet|npm run (lint|format)"),
    ("doc/.po", r"\.po\b|doc/source|sphinx|doc-po|doc-i18n|po-fill"),
    ("gh", r"\bgh\b"),
    ("git", r"\bgit (diff|log|show)"),
    ("code", r"\b(grep|rg|cat|sed|head|tail|find)\b"),
]
USAGE = ("input_tokens", "cache_read_input_tokens", "cache_creation_input_tokens", "output_tokens")
POIDS = dict(zip(USAGE, (1, 0.1, 1.25, 5)))


def categorie(nom, entree):
    if nom == "Read":
        return "doc/.po" if re.search(r"\.po$|doc/source", entree.get("file_path", "")) else "code"
    if nom != "Bash":
        return "autre"
    for lab, rx in CATEGORIES:
        if re.search(rx, entree.get("command", "")):
            return lab
    return "autre"


def lire(f, depuis, jusqua):
    ids, tours, ctx0, ctx_somme, sleeps, appels, sans_modele = set(), 0, None, 0, 0, 0, 0
    u, pend, sorties, debut = collections.Counter(), {}, [], None
    for ligne in open(f, errors="ignore"):
        try:
            j = json.loads(ligne)
        except ValueError:
            continue
        if debut is None and j.get("timestamp"):
            debut = j["timestamp"][:10]
            if (depuis and debut < depuis) or (jusqua and debut > jusqua):
                return None
        m = j.get("message") or {}
        if j.get("type") == "assistant":
            for c in m.get("content") or []:
                if not (isinstance(c, dict) and c.get("type") == "tool_use"):
                    continue
                appels += 1
                e = c.get("input", {})
                if c["name"] == "Bash" and re.search(r"\bsleep\s+\d", e.get("command", "")):
                    sleeps += 1
                if c["name"] in ("Agent", "Task") and not e.get("model") and not e.get("subagent_type") == "fork":
                    sans_modele += 1
                pend[c["id"]] = categorie(c["name"], e)
            if m.get("id") in ids:
                continue
            ids.add(m.get("id"))
            tours += 1
            us = m.get("usage") or {}
            for k in USAGE:
                u[k] += us.get(k) or 0
            ctx = sum(us.get(k) or 0 for k in USAGE[:3])
            ctx0 = ctx if ctx0 is None else ctx0
            ctx_somme += ctx
        elif j.get("type") == "user" and isinstance(m.get("content"), list):
            for c in m["content"]:
                if isinstance(c, dict) and c.get("type") == "tool_result" and c.get("tool_use_id") in pend:
                    sorties.append((pend[c["tool_use_id"]], len(json.dumps(c.get("content", ""))), tours))
    if not tours:
        return None
    relecture = collections.Counter()
    for lab, taille, t in sorties:
        relecture[lab] += taille * (tours - t)
    return dict(sub="/subagents/" in f, tours=tours, appels=appels, ctx0=ctx0,
                ctx_moyen=ctx_somme // tours, sleeps=sleeps, sans_modele=sans_modele,
                relecture=relecture, total=sum(u.values()), cache=u["cache_read_input_tokens"],
                pondere=sum(u[k] * POIDS[k] for k in USAGE))


def part_commentaires():
    fichiers = subprocess.run(["git", "ls-files", "*.go", "*.svelte", "frontend/src/*.js"],
                              capture_output=True, text=True).stdout.split()
    fichiers = [f for f in fichiers if not re.search(r"wailsjs/|i18n/help/|webui/dist", f)]
    c = t = 0
    for f in fichiers:
        for ligne in open(f, errors="ignore"):
            t += len(ligne)
            if re.match(r"\s*(//|/\*|\*|<!--)", ligne):
                c += len(ligne)
    return 100 * c / max(t, 1)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--depuis")
    ap.add_argument("--jusqu-a", dest="jusqua")
    a = ap.parse_args()
    racine = os.path.expanduser("~/.claude/projects")
    fichiers = [f for d in glob.glob(racine + "/*blunderDB*")
                for f in glob.glob(d + "/**/*.jsonl", recursive=True)]
    S = [s for s in (lire(f, a.depuis, a.jusqua) for f in fichiers) if s]
    if not S:
        print("aucun transcript dans la fenêtre")
        return
    G = lambda x: f"{x / 1e9:.2f} G" if x >= 1e9 else f"{x / 1e6:.0f} M"
    T, P = sum(s["total"] for s in S), sum(s["pondere"] for s in S)
    sub, prin = [s for s in S if s["sub"]], [s for s in S if not s["sub"]]
    print(f"fenêtre            {a.depuis or '…'} → {a.jusqua or '…'}")
    print(f"total              {G(T)} tokens ({G(P)} pondérés), relecture de cache {100 * sum(s['cache'] for s in S) / T:.0f} %")
    if prin:
        tours = sum(s["tours"] for s in prin)
        print(f"sessions principales {len(prin)} = {100 * sum(s['pondere'] for s in prin) / P:.0f} % ; "
              f"contexte moyen {sum(s['ctx_moyen'] * s['tours'] for s in prin) // tours // 1000}k ; "
              f"pire {max(s['ctx_moyen'] for s in prin) // 1000}k sur {max(prin, key=lambda s: s['ctx_moyen'])['tours']} tours")
    if sub:
        tours = sorted(s["tours"] for s in sub)
        long_ = [s for s in sub if s["appels"] > PLAFOND]
        print(f"sous-agents        {len(sub)} = {100 * sum(s['pondere'] for s in sub) / P:.0f} % ; "
              f"médiane {statistics.median(tours):.0f} tours, p90 {tours[int(.9 * len(tours))]}")
        print(f"au-delà de {PLAFOND} appels {len(long_)} agents = {100 * sum(s['pondere'] for s in long_) / P:.0f} % du coût")
    print(f"contexte fixe      {statistics.median(s['ctx0'] for s in S) // 1000}k au premier appel (médiane)")
    print(f"sleep              {sum(s['sleeps'] for s in S)} ; Agent() sans modèle explicite {sum(s['sans_modele'] for s in S)}")
    r = collections.Counter()
    for s in S:
        r.update(s["relecture"])
    tr = sum(r.values()) or 1
    print("relecture (estim.) " + ", ".join(f"{k} {100 * v / tr:.0f} %" for k, v in r.most_common()))
    print(f"commentaires       {part_commentaires():.0f} % des octets du code")


if __name__ == "__main__":
    main()
