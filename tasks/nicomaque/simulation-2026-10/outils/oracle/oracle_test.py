"""Cas faits à la main : chaque attendu est recalculé sur papier depuis les règles du README."""

import os
import sys
import unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import oracle  # noqa: E402


def players(*ids, ratings=None):
    ratings = ratings or {}
    return [{"id": p, "name": p.upper(), "rating": ratings.get(p, 0)} for p in ids]


def m(phase, a, b, winner, mid=None, **kw):
    ev = {"type": "match", "phase": phase, "a": a, "b": b, "winner": winner, **kw}
    if mid:
        ev["id"] = mid
    return ev


def ranks(report):
    return {r["player"]: r["rank"] for r in report["final"]}


def phase_ranks(report, i):
    return {r["player"]: r["rank"] for r in report["phases"][i]["ranking"]}


SWISS = {"kind": "swiss_lives", "lives": 2, "mode": "continuous"}


class SwissLives(unittest.TestCase):
    def sheet(self, events, ids=("a", "b", "c", "d")):
        return {"name": "suisse", "players": players(*ids), "phases": [SWISS], "events": events}

    def test_vainqueur_finaliste_et_victoires(self):
        ev = [m(0, "a", "b", "a"), m(0, "c", "d", "c"), m(0, "a", "c", "a"),
              m(0, "b", "d", "b"), m(0, "c", "b", "c"), m(0, "a", "c", "c"), m(0, "a", "c", "a")]
        self.assertEqual(ranks(oracle.run(self.sheet(ev))), {"a": 1, "c": 2, "b": 3, "d": 4})

    def test_elimines_a_victoires_egales_ex_aequo(self):
        ids = ("a", "b", "c", "d", "e", "f")
        ev = [m(0, "a", "b", "a"), m(0, "c", "d", "c"), m(0, "e", "f", "e"),
              m(0, "b", "d", "b"), m(0, "a", "c", "a"),  # d éliminé, 0 victoire
              m(0, "f", "b", "b"),  # f éliminé, 0 victoire
              ]
        r = oracle.run(self.sheet(ev, ids))
        f = ranks(r)
        self.assertEqual(f["d"], f["f"])
        self.assertEqual(f["d"], 5)

    def test_retire_classe_sur_ses_victoires_derriere_les_vivants(self):
        ids = ("a", "b", "c", "d")
        ev = [m(0, "a", "b", "a"), m(0, "c", "d", "c"), m(0, "a", "c", "a"),
              {"type": "withdraw", "player": "a"},  # 2 victoires, invaincu
              m(0, "b", "d", "b")]  # d éliminé, 0 victoire
        f = ranks(oracle.run(self.sheet(ev, ids)))
        # en vie : c (1 vie), b (1 vie) ; puis a retiré (2 victoires), d (0)
        self.assertEqual(f["a"], 3)
        self.assertEqual(f["d"], 4)
        self.assertEqual(f["b"], f["c"])

    def test_forfait_compte_une_defaite_et_une_vie(self):
        ev = [m(0, "a", "b", "a", forfeit=True), m(0, "c", "d", "c"),
              m(0, "b", "d", "d", forfeit=True)]  # b : 2 défaites, éliminé
        r = oracle.run(self.sheet(ev))
        notes = {x["player"]: x["note"] for x in r["final"]}
        self.assertEqual(notes["b"], "0 victoires, 2 défaites")

    def test_correction_et_annulation(self):
        ev = [m(0, "a", "b", "a", "m1"), {"type": "correct", "match": "m1", "winner": "b"},
              m(0, "c", "d", "c", "m2"), {"type": "cancel", "match": "m2"},
              m(0, "c", "d", "d", "m3")]
        r = oracle.run(self.sheet(ev))
        f = ranks(r)
        self.assertEqual(f["b"], f["d"])  # 1 victoire, 2 vies
        self.assertEqual(f["a"], f["c"])  # 1 défaite

    def test_retardataire_entre_avec_toutes_ses_vies(self):
        ev = [m(0, "a", "b", "a"), {"type": "late", "player": {"name": "Zoé Lenoir", "rating": 6}},
              m(0, "zoé-lenoir", "c", "zoé-lenoir")]
        f = ranks(oracle.run(self.sheet(ev, ("a", "b", "c"))))
        self.assertEqual(f["zoé-lenoir"], f["a"])


class SwissThenLivesBracket(unittest.TestCase):
    def sheet(self, extra):
        ev = [m(0, "a", "b", "a"), m(0, "c", "d", "c"), m(0, "b", "d", "b"), m(0, "a", "c", "a"),
              {"type": "next_phase"}] + extra
        return {"name": "t1", "players": players("a", "b", "c", "d"),
                "phases": [dict(SWISS, target=4), {"kind": "lives_bracket"}], "events": ev}

    def test_bascule_exemption_et_classement(self):
        r = oracle.run(self.sheet([
            {"type": "draw", "slots": ["a", None, "b", "c"]},
            m(1, "b", "c", "b"), m(1, "a", "b", "b")]))
        self.assertEqual(r["phases"][0]["qualified"], ["a", "b", "c"])
        byes = r["phases"][1]["expected_byes"]
        self.assertEqual((byes["size"], byes["bye_pairs"], byes["bye_by_lives"]), (4, [0], ["a"]))
        self.assertEqual(ranks(r), {"b": 1, "a": 2, "c": 3, "d": 4})
        self.assertEqual(r["warnings"], [])

    def test_joueur_a_deux_vies_non_exempte_est_signale(self):
        r = oracle.run(self.sheet([{"type": "draw", "slots": ["a", "b", "c", None]}]))
        self.assertTrue(any("2 vies" in w for w in r["warnings"]))

    def test_match_hors_tableau_refuse(self):
        with self.assertRaises(oracle.SheetError):
            oracle.run(self.sheet([{"type": "draw", "slots": ["a", None, "b", "c"]},
                                   m(1, "a", "c", "a")]))


class Bracket(unittest.TestCase):
    def sheet(self, ev, ids=("a", "b", "c", "d"), **cfg):
        return {"name": "ko", "players": players(*ids, ratings=cfg.pop("ratings", None)),
                "phases": [dict({"kind": "bracket"}, **cfg)], "events": ev}

    def test_demi_finalistes_ex_aequo(self):
        r = oracle.run(self.sheet([{"type": "draw", "slots": ["a", "b", "c", "d"]},
                                   m(0, "a", "b", "a"), m(0, "c", "d", "c"), m(0, "a", "c", "a")]))
        self.assertEqual(ranks(r), {"a": 1, "c": 2, "b": 3, "d": 3})

    def test_retire_ex_aequo_avec_les_perdants_du_match_qui_l_attendait(self):
        r = oracle.run(self.sheet([{"type": "draw", "slots": ["a", "b", "c", "d"]},
                                   m(0, "a", "b", "a"), {"type": "withdraw", "player": "c"},
                                   m(0, "a", "d", "a")]))
        self.assertEqual(ranks(r), {"a": 1, "d": 2, "b": 3, "c": 3})

    def test_tetes_de_serie_par_cote(self):
        r = oracle.run(self.sheet([{"type": "draw"}], ids=("a", "b", "c"), seeding="rating",
                                  ratings={"a": 3, "b": 5, "c": 7}))
        self.assertEqual(r["phases"][0]["slots"], ["a", "BYE", "b", "c"])

    def test_exemptions_surnumeraires_aux_premieres_paires(self):
        ids = ("a", "b", "c", "d", "e")
        r = oracle.run(self.sheet([{"type": "draw", "slots": ["a", None, "b", None, "c", None, "d", "e"]}],
                                  ids=ids))
        byes = r["phases"][0]["expected_byes"]
        self.assertEqual((byes["size"], byes["extra_byes"], byes["bye_pairs"]), (8, 3, [0, 1, 2]))
        self.assertEqual(r["warnings"], [])
        self.assertEqual({x["note"] for x in r["final"]}, {"en cours"})


POULES = [{"kind": "round_robin", "length": 5, "group_size": 4, "qualifiers": 2},
          {"kind": "bracket", "length": 9}]

POOL_A = [m(0, "a1", "a2", "a1"), m(0, "a1", "a3", "a1"), m(0, "a1", "a4", "a1"),
          m(0, "a2", "a3", "a2"), m(0, "a2", "a4", "a2"), m(0, "a3", "a4", "a3")]
POOL_B = [m(0, "b1", "b2", "b1"), m(0, "b1", "b3", "b1"), m(0, "b1", "b4", "b1"),
          m(0, "b2", "b3", "b2"), m(0, "b3", "b4", "b3"), m(0, "b4", "b2", "b4"),
          # barrage à 2 vies : b2, b3, b4 pour une place
          m(0, "b2", "b3", "b2"), m(0, "b2", "b4", "b2"), m(0, "b3", "b4", "b3"), m(0, "b2", "b3", "b2")]
IDS8 = ("a1", "a2", "a3", "a4", "b1", "b2", "b3", "b4")
DRAW8 = {"type": "draw", "groups": [["a1", "a2", "a3", "a4"], ["b1", "b2", "b3", "b4"]]}


class Poules(unittest.TestCase):
    def sheet(self, extra, n26=None):
        s = {"name": "poules", "players": players(*IDS8), "phases": POULES,
             "events": [DRAW8] + POOL_A + POOL_B + extra}
        if n26:
            s["n26"] = n26
        return s

    def test_poules_barrage_puis_tableau(self):
        r = oracle.run(self.sheet([
            {"type": "next_phase"}, {"type": "draw", "slots": ["a1", "b2", "b1", "a2"]},
            m(1, "a1", "b2", "a1"), m(1, "a2", "b1", "a2"), m(1, "a1", "a2", "a1")]))
        self.assertEqual(r["phases"][0]["barrages"], {"Poule B": {"players": ["b2", "b3", "b4"], "spots": 1}})
        self.assertEqual(phase_ranks(r, 0), {"a1": 1, "b1": 1, "a2": 3, "b2": 4,
                                             "a3": 5, "b3": 5, "b4": 5, "a4": 8})
        self.assertEqual(ranks(r), {"a1": 1, "a2": 2, "b1": 3, "b2": 3,
                                    "a3": 5, "b3": 5, "b4": 5, "a4": 8})

    def test_n26_repechage_du_suivant_de_poule(self):
        r = oracle.run(self.sheet([{"type": "withdraw", "player": "a2"}, {"type": "next_phase"}]))
        self.assertEqual(sorted(r["phases"][0]["qualified"]), ["a1", "a3", "b1", "b2"])
        p0 = phase_ranks(r, 0)
        self.assertLess(p0["a3"], p0["a2"])  # le repêché passe devant le retiré

    def test_n26_regle_actuelle_du_moteur_place_perdue(self):
        r = oracle.run(self.sheet([{"type": "withdraw", "player": "a2"}, {"type": "next_phase"},
                                   {"type": "draw", "slots": ["a1", None, "b1", "b2"]}], n26="moteur"))
        self.assertEqual(sorted(r["phases"][0]["qualified"]), ["a1", "b1", "b2"])
        self.assertEqual(r["phases"][1]["expected_byes"]["bye_pairs"], [0])

    def test_walkover_contre_un_retire_ne_compte_pas(self):
        s = {"name": "p", "players": players("x1", "x2", "x3", "x4"), "phases": POULES[:1],
             "events": [{"type": "draw", "groups": [["x1", "x2", "x3", "x4"]]},
                        m(0, "x1", "x4", "x1"), {"type": "withdraw", "player": "x4"},
                        m(0, "x1", "x2", "x1"), m(0, "x1", "x3", "x1"), m(0, "x2", "x3", "x2")]}
        r = oracle.run(s)
        self.assertEqual(ranks(r), {"x1": 1, "x2": 2, "x3": 3, "x4": 3})


class Rencontre(unittest.TestCase):
    def test_chaque_epreuve_classee_separement(self):
        a = {"name": "A", "players": players("a", "b"), "phases": [{"kind": "bracket"}],
             "events": [{"type": "draw", "slots": ["a", "b"]}, m(0, "a", "b", "b")]}
        b = {"name": "B", "players": players("a", "b"), "phases": [SWISS],
             "events": [m(0, "a", "b", "a"), m(0, "a", "b", "a")]}
        r = oracle.run({"rencontre": "Week-end", "epreuves": [a, b]})
        self.assertEqual([ranks(e) for e in r["epreuves"]], [{"b": 1, "a": 2}, {"a": 1, "b": 2}])


if __name__ == "__main__":
    unittest.main()
