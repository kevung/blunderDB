#!/usr/bin/env python3
"""Feuille papier oracle : recalcule le classement d'une épreuve dirigée depuis la feuille JSON.

Réimplémentation indépendante des règles documentées du moteur Nicomaque
(backgammon-tournoi/docs/specification.md) ; aucun code de l'application n'est lu ni appelé.
Schéma d'entrée, règles et sources : README.md du même dossier.

Usage : python3 oracle.py feuille.json   → JSON sur la sortie standard.
"""

import json
import math
import sys
import unicodedata

BYE = "BYE"


class SheetError(Exception):
    """La feuille contredit les règles du format : le directeur a noté un match impossible."""


def slug(name):
    """Identifiant que blunderDB donne à un participant (minuscules, lettres et chiffres, tirets)."""
    out = []
    for ch in name.lower():
        if unicodedata.category(ch)[0] in ("L", "N"):
            out.append(ch)
        elif out and out[-1] != "-":
            out.append("-")
    s = "".join(out).strip("-")
    return s or "joueur"


def pow2ceil(n):
    p = 1
    while p < n:
        p *= 2
    return p


def phase_defaults(cfg):
    cfg = dict(cfg)
    kind = cfg.get("kind")
    if kind == "swiss_lives":
        if cfg.get("lives", 0) <= 0:
            cfg["lives"] = 2
        cfg.setdefault("target", 0)
    elif kind == "round_robin":
        if cfg.get("group_size", 0) <= 0:
            cfg["group_size"] = 4
        if cfg.get("qualifiers", 0) <= 0:
            cfg["qualifiers"] = 2
    elif kind in ("bracket", "lives_bracket"):
        if cfg.get("consolation"):
            raise SheetError("consolante non couverte par l'oracle (section principale seule)")
    else:
        raise SheetError("format non couvert par l'oracle : %r" % kind)
    if not cfg.get("entry"):
        cfg["entry"] = "survivors"
    cfg.setdefault("seeding", "")
    return cfg


def lives_for(cfg):
    return cfg["lives"] if cfg["kind"] == "swiss_lives" else 1


# ---------------------------------------------------------------- graphes (tableau, poules)


class GMatch:
    def __init__(self, key, src):
        self.key = key
        self.src = src  # [(joueur fixe ou None, index source ou -1)] * 2
        self.reset()

    def reset(self):
        self.players = [s[0] for s in self.src]
        self.done = self.walkover = False
        self.winner = self.loser = None
        self.match_id = None


def resolve(gms, withdrawn):
    """Propage les vainqueurs, résout exemptions et forfaits de retirés (match non lancé)."""
    changed = True
    while changed:
        changed = False
        for g in gms:
            for k in (0, 1):
                fixed, frm = g.src[k]
                p = fixed
                if p is None and frm >= 0 and gms[frm].done:
                    p = gms[frm].winner
                if p is not None and g.players[k] != p:
                    g.players[k] = p
                    changed = True
            a, b = g.players
            if g.done or a is None or b is None:
                continue
            if g.match_id is None and BYE not in (a, b) and (a in withdrawn or b in withdrawn):
                g.done = g.walkover = True
                g.winner, g.loser = (b, a) if (a in withdrawn and b not in withdrawn) else (a, b)
                changed = True
            elif a == BYE or b == BYE:
                g.done = g.walkover = True
                g.winner, g.loser = (b, BYE) if a == BYE else (a, BYE)
                changed = True


def build_bracket(slots):
    size = len(slots)
    rounds, gms, prev = [], [], []
    for i in range(size // 2):
        gms.append(GMatch("main.0.%d" % i, [(slots[2 * i], -1), (slots[2 * i + 1], -1)]))
        prev.append(len(gms) - 1)
    rounds.append(prev)
    r = 1
    while len(prev) > 1:
        idx = []
        for i in range(len(prev) // 2):
            gms.append(GMatch("main.%d.%d" % (r, i), [(None, prev[2 * i]), (None, prev[2 * i + 1])]))
            idx.append(len(gms) - 1)
        rounds.append(idx)
        prev = idx
        r += 1
    return gms, rounds


def seed_order(size):
    order, n = [1], 1
    while n < size:
        order = [y for x in order for y in (x, 2 * n + 1 - x)]
        n *= 2
    return order


# ---------------------------------------------------------------- état d'une épreuve


class Match:
    def __init__(self, ev, order):
        self.id = ev.get("id") or "m%d" % order
        self.phase = ev["phase"]
        self.a, self.b = ev["a"], ev["b"]
        self.winner = ev["winner"]
        self.forfeit = bool(ev.get("forfeit"))
        self.cancelled = False
        self.section = None  # "main", ("pool", i), ("barrage", i), "swiss"
        self.key = None  # index du GMatch
        if self.winner not in (self.a, self.b):
            raise SheetError("match %s : vainqueur %r absent du match" % (self.id, self.winner))

    @property
    def loser(self):
        return self.b if self.winner == self.a else self.a


class Phase:
    def __init__(self, index, cfg):
        self.index, self.cfg, self.kind = index, cfg, cfg["kind"]
        self.entrants, self.lives = [], {}
        self.drawn = False
        self.gms, self.rounds, self.slots = [], [], None
        self.pools = []  # [(nom, [joueurs], [index GMatch])]
        self.barrages = {}  # index de poule → {"players": [...], "spots": n}
        self.byes = {}
        self.expected_byes = None

    def enter(self, p, lives):
        if p not in self.lives:
            self.entrants.append(p)
            self.lives[p] = lives


class Tournament:
    def __init__(self, sheet):
        self.sheet = sheet
        self.name = sheet.get("name", "")
        self.n26 = sheet.get("n26", "repechage")
        self.players, self.order = {}, []
        self.withdrawn = set()
        self.matches, self.match_order = {}, []
        self.warnings = []
        self.repechages = []
        self.phases = [Phase(i, phase_defaults(c)) for i, c in enumerate(sheet["phases"])]
        self.current = 0
        for pl in sheet["players"]:
            self._register(pl)
        for ev in sheet.get("events", []):
            self.apply(ev)

    # -------- inscription
    def _register(self, pl, slot=None):
        pid = pl.get("id") or slug(pl["name"])
        pl = dict(pl, id=pid)
        if pid not in self.players:
            self.order.append(pid)
        self.players[pid] = pl
        self.withdrawn.discard(pid)
        ph = self.phases[self.current]
        if slot is not None:
            if not ph.drawn or ph.slots is None or ph.slots[slot] != BYE:
                raise SheetError("retardataire %s : la place %d n'est pas une exemption libre" % (pid, slot))
            ph.slots[slot] = pid
            g = ph.gms[slot // 2]
            g.src[slot % 2] = (pid, -1)
            ph.enter(pid, 1)
        elif self.current == 0 and not ph.drawn and (ph.kind == "swiss_lives" or not self._started(ph)):
            ph.enter(pid, lives_for(ph.cfg))
        else:
            self.warnings.append("retardataire %s inscrit sans entrer dans une phase (non classé)" % pid)
        return pid

    def _started(self, ph):
        return any(self.matches[m].phase == ph.index for m in self.match_order)

    # -------- événements
    def apply(self, ev):
        t = ev["type"]
        if t == "match":
            self._match(ev)
        elif t == "correct":
            m = self._get(ev["match"])
            if ev["winner"] not in (m.a, m.b):
                raise SheetError("correction de %s : vainqueur absent du match" % m.id)
            m.winner, m.cancelled = ev["winner"], False
            if "forfeit" in ev:
                m.forfeit = bool(ev["forfeit"])
        elif t == "cancel":
            self._get(ev["match"]).cancelled = True
        elif t == "withdraw":
            if ev["player"] not in self.players:
                raise SheetError("retrait d'un joueur inconnu : %s" % ev["player"])
            self.withdrawn.add(ev["player"])
        elif t == "late":
            self._register(ev["player"], ev.get("slot"))
        elif t == "bye":
            ph = self.phases[ev.get("phase", self.current)]
            ph.byes[ev["player"]] = ph.byes.get(ev["player"], 0) + 1
        elif t == "draw":
            self._draw(self.phases[ev.get("phase", self.current)], ev)
        elif t == "next_phase":
            self._next_phase(ev)
        else:
            raise SheetError("événement inconnu : %r" % t)

    def _get(self, mid):
        if mid not in self.matches:
            raise SheetError("match inconnu : %s" % mid)
        return self.matches[mid]

    def _match(self, ev):
        m = Match(ev, len(self.match_order) + 1)
        if m.id in self.matches:
            raise SheetError("match %s noté deux fois" % m.id)
        ph = self.phases[m.phase]
        for p in (m.a, m.b):
            if p not in ph.lives:
                raise SheetError("match %s : %s n'est pas entrant de la phase %d" % (m.id, p, m.phase))
            if p in self.withdrawn:
                raise SheetError("match %s : %s est retiré" % (m.id, p))
        if ph.kind == "swiss_lives":
            self.evaluate(ph)
            for p in (m.a, m.b):
                if self.remaining(ph, p) <= 0:
                    self.warnings.append("match %s : %s était déjà éliminé" % (m.id, p))
            m.section = "swiss"
        elif ph.kind in ("bracket", "lives_bracket"):
            self._link_graph(ph, m, range(len(ph.gms)), "main")
        else:
            self._place_rr(ph, m, ev.get("section"))
        self.matches[m.id] = m
        self.match_order.append(m.id)

    def _link_graph(self, ph, m, candidates, section):
        if not ph.drawn:
            raise SheetError("match %s joué avant le tirage de la phase %d" % (m.id, ph.index))
        self.evaluate(ph)
        for i in candidates:
            g = ph.gms[i]
            if not g.done and g.match_id is None and set(g.players) == {m.a, m.b}:
                m.section, m.key = section, i
                return True
        if section == "main":
            raise SheetError("match %s : %s contre %s n'est pas un match attendu du tableau" % (m.id, m.a, m.b))
        return False

    def _place_rr(self, ph, m, hint):
        if not ph.drawn:
            raise SheetError("match %s joué avant le tirage des poules" % m.id)
        pool = next((i for i, (_, pls, _) in enumerate(ph.pools) if m.a in pls and m.b in pls), None)
        if pool is None:
            raise SheetError("match %s : %s et %s ne sont pas dans la même poule" % (m.id, m.a, m.b))
        if hint != "barrage" and self._link_graph(ph, m, ph.pools[pool][2], ("pool", pool)):
            return
        self.evaluate(ph)
        bs = self._barrage(ph, pool)
        if bs is None or m.a not in bs["players"] or m.b not in bs["players"]:
            raise SheetError("match %s : ni match de poule restant, ni barrage entre %s et %s" % (m.id, m.a, m.b))
        alive = self.barrage_alive(ph, pool)
        if m.a not in alive or m.b not in alive or len(alive) <= bs["spots"]:
            raise SheetError("match %s : barrage déjà tranché ou joueur éliminé du barrage" % m.id)
        m.section = ("barrage", pool)

    # -------- tirages
    def _draw(self, ph, ev):
        if ph.drawn:
            raise SheetError("phase %d tirée deux fois" % ph.index)
        if ph.kind == "round_robin":
            groups = ev["groups"]
            n, gs = len(ph.entrants), ph.cfg["group_size"]
            ng = max(1, math.ceil(n / gs))
            if sorted(p for g in groups for p in g) != sorted(ph.entrants):
                raise SheetError("tirage des poules : les poules ne contiennent pas les entrants")
            sizes = sorted(len(g) for g in groups)
            if len(groups) != ng or sizes[-1] - sizes[0] > 1:
                self.warnings.append("tirage des poules : %d poules de tailles %s, attendu %d poules équilibrées"
                                     % (len(groups), sizes, ng))
            for i, g in enumerate(groups):
                idx = []
                for x in range(len(g)):
                    for y in range(x + 1, len(g)):
                        ph.gms.append(GMatch("pool%d.%d.%d" % (i, x, y), [(g[x], -1), (g[y], -1)]))
                        idx.append(len(ph.gms) - 1)
                ph.pools.append(("Poule " + chr(ord("A") + i), list(g), idx))
        else:
            self._bracket_draw(ph, ev.get("slots"))
        ph.drawn = True

    def _bracket_draw(self, ph, noted):
        deux = [p for p in ph.entrants if ph.lives[p] >= 2]
        une = [p for p in ph.entrants if ph.lives[p] < 2]
        total = 2 * len(deux) + len(une)
        size = max(2, pow2ceil(total))
        npairs, extra = size // 2, size - total
        if ph.cfg["seeding"] == "rating":
            ordre = self._by_strength(deux) + self._by_strength(une)
            expected = [BYE] * size
            for place, tete in enumerate(seed_order(size)):
                if tete <= len(ordre):
                    expected[place] = ordre[tete - 1]
            bye_pairs = [i for i in range(npairs) if BYE in expected[2 * i:2 * i + 2]]
        else:
            expected = None
            order = list(range(0, npairs, 2)) + list(range(1, npairs, 2))
            taken = set(order[:len(deux)])
            free = [i for i in range(npairs) if i not in taken]
            bye_pairs = sorted(taken | set(free[:min(extra, len(une))]))
        ph.expected_byes = {"size": size, "sum_lives": total, "extra_byes": extra,
                            "bye_pairs": bye_pairs, "bye_by_lives": sorted(deux)}
        if expected is not None:
            ph.expected_byes["slots"] = expected
        if noted is None:
            if expected is None:
                raise SheetError("tirage aléatoire du tableau : la feuille doit noter les places (slots)")
            noted = expected
        slots = [BYE if s in (None, BYE) else s for s in noted]
        if len(slots) != size:
            raise SheetError("tableau de %d places noté, %d attendues" % (len(slots), size))
        if sorted(s for s in slots if s != BYE) != sorted(ph.entrants):
            raise SheetError("le tableau noté ne contient pas exactement les entrants de la phase")
        if expected is not None and slots != expected:
            self.warnings.append("têtes de série : places notées différentes du placement par cote")
        got = [i for i in range(npairs) if BYE in slots[2 * i:2 * i + 2]]
        if got != bye_pairs:
            self.warnings.append("exemptions aux paires %s, attendues aux paires %s" % (got, bye_pairs))
        for p in deux:
            i = slots.index(p) // 2
            if BYE not in slots[2 * i:2 * i + 2]:
                self.warnings.append("%s (2 vies) n'est pas exempt du premier tour" % p)
        ph.slots = slots
        ph.gms, ph.rounds = build_bracket(slots)

    def _by_strength(self, ids):
        def key(p):
            r = self.players[p].get("rating") or 0
            return (1, 0.0, p) if r <= 0 else (0, r, p)
        return sorted(ids, key=key)

    # -------- passage de phase
    def _next_phase(self, ev):
        prev = self.phases[self.current]
        if self.current + 1 >= len(self.phases):
            raise SheetError("next_phase après la dernière phase")
        self.evaluate(prev)
        if prev.kind == "swiss_lives" and not self.swiss_done(prev):
            self.warnings.append("passage de phase alors que le suisse n'est pas terminé")
        nxt = self.phases[self.current + 1]
        entry = nxt.cfg["entry"]
        if entry == "all":
            for p in self.order:
                if p not in self.withdrawn:
                    nxt.enter(p, lives_for(nxt.cfg))
        elif entry.startswith("top:") and len(entry) > 4:
            n = int(entry[4:]) if entry[4:].isdigit() else 0
            for p, rk, _ in self.phase_ranking(prev):
                if rk <= n and p not in self.withdrawn:
                    nxt.enter(p, lives_for(nxt.cfg))
        else:
            surv = self.survivors(prev)
            if prev.kind == "round_robin":
                surv = self._n26(prev, surv, ev.get("repechage", {}))
            for p in surv:
                if p in self.withdrawn:
                    continue
                lv = self.remaining(prev, p)
                if nxt.kind not in ("lives_bracket", "swiss_lives"):
                    lv = lives_for(nxt.cfg)
                nxt.enter(p, lv)
        self.current += 1

    def _n26(self, ph, qualified, hints):
        """Qualifié de poule retiré avant le tableau. Moteur : sa place est perdue (issue #26).
        Hypothèse « repechage » : le suivant non retiré de sa poule prend la place."""
        if self.n26 != "repechage":
            return qualified
        out = list(qualified)
        for i, (name, pls, idx) in enumerate(ph.pools):
            gone = [p for p in qualified if p in pls and p in self.withdrawn]
            for p in gone:
                wins = self.rr_wins(ph, i)
                cands = [c for c in pls if c not in out and c not in self.withdrawn]
                if not cands:
                    continue
                best = max(wins[c] for c in cands)
                tied = sorted(c for c in cands if wins[c] == best)
                pick = hints.get(name) if isinstance(hints, dict) else None
                if pick is None:
                    if len(tied) > 1:
                        self.warnings.append("N26 %s : repêchage indécis entre %s (noter 'repechage')"
                                             % (name, tied))
                        continue
                    pick = tied[0]
                out[out.index(p)] = pick
                self.repechages.append(pick)
        return out

    # -------- recalcul complet d'une phase
    def evaluate(self, ph):
        ph.wins = {p: 0 for p in ph.entrants}
        ph.losses = {p: 0 for p in ph.entrants}
        ph.elim_order = []
        for g in ph.gms:
            g.reset()
        live = [self.matches[i] for i in self.match_order
                if self.matches[i].phase == ph.index and not self.matches[i].cancelled]
        for m in live:
            if m.key is not None:
                ph.gms[m.key].match_id = m.id
        for m in live:
            ph.wins[m.winner] = ph.wins.get(m.winner, 0) + 1
            ph.losses[m.loser] = ph.losses.get(m.loser, 0) + 1
            if self.remaining(ph, m.loser) == 0 and ph.lives.get(m.loser, 0) > 0:
                ph.elim_order.append(m.loser)
            if m.key is not None:
                g = ph.gms[m.key]
                g.done, g.winner, g.loser = True, m.winner, m.loser
        resolve(ph.gms, self.withdrawn)

    def remaining(self, ph, p):
        if p in self.withdrawn:
            return 0
        return max(0, ph.lives.get(p, 0) - ph.losses.get(p, 0))

    def alive(self, ph):
        return [p for p in ph.entrants if self.remaining(ph, p) > 0]

    def swiss_done(self, ph):
        alive = self.alive(ph)
        target = ph.cfg.get("target", 0)
        if target > 0 and sum(self.remaining(ph, p) for p in ph.entrants) <= target:
            return True
        return len(alive) <= 1

    def survivors(self, ph):
        if ph.kind == "round_robin":
            return self.rr_qualified(ph)
        if ph.kind in ("bracket", "lives_bracket"):
            if not ph.drawn:
                return list(ph.entrants)
            r = self.bracket_ranking(ph)
            if all(g.done for g in ph.gms):
                return [r[0][0]]
            return [p for p, _, note in r if note == "en cours"]
        return self.alive(ph)

    # -------- poules
    def rr_wins(self, ph, i):
        w = {p: 0 for p in ph.pools[i][1]}
        for gi in ph.pools[i][2]:
            g = ph.gms[gi]
            if g.done and not g.walkover:
                w[g.winner] += 1
        return w

    def _sorted_pool(self, w):
        return sorted(sorted(w), key=lambda p: -w[p])

    def rr_tie(self, ph, i):
        w = self.rr_wins(ph, i)
        ids = self._sorted_pool(w)
        q = ph.cfg["qualifiers"]
        if q >= len(ids) or w[ids[q - 1]] != w[ids[q]]:
            return [], 0
        v, spots, tied = w[ids[q - 1]], q, []
        for p in ids:
            if w[p] > v:
                spots -= 1
            elif w[p] == v:
                tied.append(p)
        return tied, spots

    def _pool_done(self, ph, i):
        return all(ph.gms[gi].done for gi in ph.pools[i][2])

    def _barrage(self, ph, i):
        """Le barrage d'une poule, créé quand la poule est finie sur une égalité."""
        if i not in ph.barrages and self._pool_done(ph, i):
            tied, spots = self.rr_tie(ph, i)
            if tied:
                ph.barrages[i] = {"players": tied, "spots": spots}
        return ph.barrages.get(i)

    def barrage_alive(self, ph, i):
        losses = {}
        for mid in self.match_order:
            m = self.matches[mid]
            if m.phase == ph.index and m.section == ("barrage", i) and not m.cancelled:
                losses[m.loser] = losses.get(m.loser, 0) + 1
        return [p for p in ph.barrages[i]["players"] if losses.get(p, 0) < 2]

    def rr_qualified(self, ph):
        out = []
        for i in range(len(ph.pools)):
            w = self.rr_wins(ph, i)
            ids = self._sorted_pool(w)
            tied, spots = self.rr_tie(ph, i)
            if not tied:
                out += ids[:min(ph.cfg["qualifiers"], len(ids))]
                continue
            v = w[tied[0]]
            out += [p for p in ids if w[p] > v]
            if self._barrage(ph, i) is not None:
                alive = self.barrage_alive(ph, i)
                if len(alive) <= spots:
                    out += alive
        return out

    def rr_ranking(self, ph):
        qual = set(self.rr_qualified(ph))
        if self.n26 == "repechage" and self.current > ph.index:
            qual |= set(self.repechages)
        score, note = {}, {}
        for i, (name, _, _) in enumerate(ph.pools):
            for p, w in self.rr_wins(ph, i).items():
                score[p], note[p] = w, "%s, %d victoires" % (name, w)
                if p in self.withdrawn:
                    note[p] = "retiré, " + note[p]
                elif p in qual:
                    score[p] += 100
                    note[p] += ", qualifié"
        ids = sorted(sorted(ph.entrants), key=lambda p: -score.get(p, 0))
        return self._ranks(ids, score, note)

    # -------- tableau
    def bracket_ranking(self, ph):
        if not ph.drawn:
            return [(p, 1, "en attente du tirage") for p in ph.entrants]
        score = {p: -1 for p in ph.entrants}
        note = {p: "non classé" for p in ph.entrants}

        def sortie(p, r, label):
            cand = 3000 + r
            if score.get(p, -1) < 0 or cand > score[p]:
                score[p], note[p] = cand, label

        nr = len(ph.rounds)
        for r, idx in enumerate(ph.rounds):
            label = round_label(r, nr)
            for i in idx:
                g = ph.gms[i]
                if not g.done:
                    for p in g.players:
                        if p not in (None, BYE) and p in self.withdrawn:
                            sortie(p, r, label)
                    continue
                if g.walkover and g.loser not in self.withdrawn:
                    continue
                sortie(g.loser, r, label)
        last = ph.gms[-1]
        if last.done and last.winner != BYE:
            score[last.winner], note[last.winner] = 3999, "vainqueur"
        for p in ph.entrants:
            if p in self.withdrawn:
                note[p] = "retiré, " + note[p]
            if score[p] < 0:
                score[p] = 0
                if note[p] == "non classé":
                    score[p], note[p] = 5000, "en cours"
        return self._ranks(list(ph.entrants), score, note)

    # -------- suisse
    def lives_ranking(self, ph):
        self.evaluate(ph)
        alive = self.alive(ph)
        score, note = {}, {}
        for p in ph.entrants:
            w, l, rem = ph.wins.get(p, 0), ph.losses.get(p, 0), self.remaining(ph, p)
            score[p], note[p] = w, "%d victoires, %d défaites" % (w, l)
            if rem > 0:
                score[p] = w + 1000 + rem
                note[p] = "vainqueur" if len(alive) == 1 else "en vie (%d vies)" % rem
            if p in self.withdrawn:
                score[p] = w
                note[p] = "retiré, %d victoires, %d défaites" % (w, l)
        if len(alive) == 1 and ph.elim_order:
            last = ph.elim_order[-1]
            score[last], note[last] = 999, "finaliste"
        return self._ranks(list(ph.entrants), score, note)

    @staticmethod
    def _ranks(ids, score, note):
        ids = sorted(ids, key=lambda p: -score.get(p, 0))  # tri stable
        out, rank = [], 1
        for i, p in enumerate(ids):
            if i > 0 and score.get(p, 0) != score.get(ids[i - 1], 0):
                rank = i + 1
            out.append((p, rank, note.get(p, "")))
        return out

    def phase_ranking(self, ph):
        self.evaluate(ph)
        if ph.kind in ("bracket", "lives_bracket"):
            return self.bracket_ranking(ph)
        if ph.kind == "round_robin":
            return self.rr_ranking(ph)
        return self.lives_ranking(ph)

    def ranking(self):
        """Classement général : de la dernière phase atteinte à la première, offset = taille."""
        out, seen, offset = [], set(), 0
        for ph in reversed(self.phases[:self.current + 1]):
            r = self.phase_ranking(ph)
            for p, rk, nt in r:
                if p in seen:
                    continue
                seen.add(p)
                out.append((p, rk + offset, nt))
            offset += len(r)
        out.sort(key=lambda x: x[1])
        res, prev, prev_rank = [], None, 0
        for pos, (p, rk, nt) in enumerate(out):
            if rk != prev:
                prev, prev_rank = rk, pos + 1
            res.append((p, prev_rank, nt))
        return res

    # -------- sortie
    def report(self):
        def rows(r):
            r = sorted(r, key=lambda x: (x[1], x[0]))
            return [{"rank": rk, "player": p, "name": self.players[p].get("name", p), "note": nt}
                    for p, rk, nt in r]

        phases = []
        for ph in self.phases[:self.current + 1]:
            d = {"index": ph.index, "kind": ph.kind, "ranking": rows(self.phase_ranking(ph))}
            if ph.kind == "round_robin":
                d["pools"] = [{"name": n, "players": pls} for n, pls, _ in ph.pools]
                d["barrages"] = {ph.pools[i][0]: b for i, b in ph.barrages.items()}
            if ph.index < self.current:
                d["qualified"] = self.phases[ph.index + 1].entrants
            if ph.expected_byes is not None:
                d["expected_byes"] = ph.expected_byes
                d["slots"] = ph.slots
            phases.append(d)
        return {"name": self.name, "final": rows(self.ranking()), "phases": phases,
                "warnings": self.warnings}


def round_label(r, nr):
    left = nr - r
    return {1: "finale", 2: "demi-finale", 3: "quart de finale"}.get(left, "tour %d" % (r + 1))


def run(sheet):
    """Une épreuve, ou une Rencontre ({"rencontre", "epreuves": [...]}) classée épreuve par épreuve."""
    if "epreuves" in sheet:
        return {"rencontre": sheet.get("rencontre", ""),
                "epreuves": [Tournament(e).report() for e in sheet["epreuves"]]}
    return Tournament(sheet).report()


def main(argv):
    if len(argv) != 2:
        sys.stderr.write("usage : oracle.py feuille.json\n")
        return 2
    with open(argv[1], encoding="utf-8") as f:
        sheet = json.load(f)
    try:
        out = run(sheet)
    except SheetError as e:
        sys.stderr.write("feuille incohérente : %s\n" % e)
        return 1
    json.dump(out, sys.stdout, ensure_ascii=False, indent=2)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
