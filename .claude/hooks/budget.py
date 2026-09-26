#!/usr/bin/env python3
"""Garde mécanique du budget de contexte (ADR-0055).

PreToolUse  Bash : refuse `sleep N` ; refuse `cat` d'un seul fichier de plus de GROS lignes.
PreToolUse  Read : refuse la lecture sans `limit` d'un fichier texte de plus de GROS lignes.
PostToolUse      : sous-agent — compte ses appels et, au plafond, ordonne de rendre
                   « à reprendre » ; session principale — au-delà de CTX_MAX tokens de
                   contexte, demande de consigner l'état et de proposer /clear.
Ne bloque jamais sur une entrée inattendue : toute erreur laisse passer l'appel.
"""
import json, os, re, sys

PLAFOND = int(os.environ.get("BLUNDERDB_PLAFOND_APPELS", "150"))
CTX_MAX = int(os.environ.get("BLUNDERDB_CTX_MAX", "200000"))
GROS = 400
RAPPEL = 10

BINAIRE = re.compile(r"\.(png|jpe?g|gif|webp|svg|pdf|ipynb|db|gz|bd|bin)$", re.I)


def lignes(chemin):
    try:
        if not os.path.isfile(chemin) or BINAIRE.search(chemin):
            return 0
        with open(chemin, "rb") as f:
            return sum(1 for _ in f)
    except OSError:
        return 0


def refuse(raison):
    print(json.dumps({"hookSpecificOutput": {
        "hookEventName": "PreToolUse",
        "permissionDecision": "deny",
        "permissionDecisionReason": raison,
    }}))


def ajoute(texte):
    print(json.dumps({"hookSpecificOutput": {
        "hookEventName": "PostToolUse",
        "additionalContext": texte,
    }}))


def pre(e):
    outil, entree = e.get("tool_name"), e.get("tool_input") or {}
    cwd = e.get("cwd") or os.getcwd()
    if outil == "Bash":
        cmd = entree.get("command", "")
        if re.search(r"(^|[;&|(]\s*|\bdo\s+|\bthen\s+)sleep\s+\d", cmd):
            return refuse("Pas de sleep (ADR-0055) : n'attendez ni la CI ni un serveur. "
                          "Rendez votre verdict, ou utilisez Monitor / run_in_background.")
        m = re.fullmatch(r"\s*cat(?:\s+-\w+)*\s+(\S+)\s*", cmd)
        if m:
            chemin = os.path.join(cwd, os.path.expanduser(m.group(1).strip("'\"")))
            n = lignes(chemin)
            if n > GROS:
                return refuse(f"{m.group(1)} fait {n} lignes (ADR-0055) : `grep -n` pour situer, "
                              "puis `sed -n 'a,bp'` sur la plage utile.")
    elif outil == "Read" and not entree.get("limit"):
        chemin = entree.get("file_path", "")
        n = lignes(chemin)
        if n > GROS:
            return refuse(f"{os.path.basename(chemin)} fait {n} lignes (ADR-0055) : situez la "
                          "zone avec grep -n, puis Read avec offset/limit. Une lecture "
                          "partielle suffit pour Edit.")


def contexte(transcript):
    """Taille du contexte au dernier tour, lue dans la fin du transcript."""
    try:
        with open(transcript, "rb") as f:
            f.seek(0, 2)
            f.seek(max(0, f.tell() - 400_000))
            fin = f.read().decode("utf-8", "ignore").splitlines()
    except OSError:
        return 0
    for ligne in reversed(fin):
        if '"usage"' not in ligne:
            continue
        try:
            u = json.loads(ligne)["message"]["usage"]
        except (ValueError, KeyError, TypeError):
            continue
        return sum(u.get(k) or 0 for k in
                   ("input_tokens", "cache_read_input_tokens", "cache_creation_input_tokens"))
    return 0


def post(e):
    agent = e.get("agent_id")
    d = os.path.join(os.environ.get("XDG_RUNTIME_DIR") or "/tmp", "blunderdb-budget")
    os.makedirs(d, exist_ok=True)
    cle = agent or e.get("session_id") or "principal"
    f = os.path.join(d, re.sub(r"[^A-Za-z0-9_-]", "_", cle))
    try:
        n = int(open(f).read() or 0) + 1
    except (OSError, ValueError):
        n = 1
    with open(f, "w") as out:
        out.write(str(n))
    if agent:
        if n >= PLAFOND and (n - PLAFOND) % RAPPEL == 0:
            ajoute(f"PLAFOND ATTEINT ({n} appels, plafond {PLAFOND}) — ADR-0055. Arrêtez-vous : "
                   "committez ce qui est stable, puis rendez le contrat de retour avec le verdict "
                   "« à reprendre » et l'état de reprise en trois lignes (branche et dernier "
                   "commit, dernier rouge et son motif, hypothèse en cours).")
        return
    if n % 25:
        return
    ctx = contexte(e.get("transcript_path", ""))
    if ctx > CTX_MAX:
        ajoute(f"Contexte à {ctx // 1000}k tokens (seuil {CTX_MAX // 1000}k, ADR-0055) : chaque "
               "tour le relit en entier. Terminez l'étape en cours, consignez l'état de reprise, "
               "et proposez à l'utilisateur un /clear avant la suite.")


def main():
    try:
        e = json.load(sys.stdin)
        {"PreToolUse": pre, "PostToolUse": post}.get(e.get("hook_event_name"), lambda _: None)(e)
    except Exception:  # une garde ne doit jamais casser un appel
        return


if __name__ == "__main__":
    main()
