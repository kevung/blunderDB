import json
import os
import tempfile
import unittest

import mesure

DIR = os.path.join(os.path.dirname(__file__), "fixtures")
EVENTS = os.path.join(DIR, "events.json")
MARKS = os.path.join(DIR, "marks.csv")
REST = [("dice.rest", "roll_tick_ms")]


class Apparier(unittest.TestCase):
    def test_tolerance_bound_included(self):
        p, _, _ = mesure.apparier([10500], [10000], 500)
        self.assertEqual(len(p), 1)
        p, _, _ = mesure.apparier([9500], [10000], 500)
        self.assertEqual(len(p), 1)

    def test_tolerance_bound_excluded_beyond(self):
        self.assertEqual(mesure.apparier([10501], [10000], 500)[0], [])
        self.assertEqual(mesure.apparier([9499], [10000], 500)[0], [])

    def test_one_to_one(self):
        p, nd, nr = mesure.apparier([10000, 10100], [10000], 500)
        self.assertEqual((len(p), nd, nr), (1, 2, 1))
        p, nd, nr = mesure.apparier([10000], [10000, 10100], 500)
        self.assertEqual((len(p), nd, nr), (1, 1, 2))

    def test_closest_pair_first_not_first_come(self):
        # 39.7 s comes first in time, 40.1 s is the closest to the mark.
        p, _, _ = mesure.apparier([39700, 40100], [40000], 500)
        self.assertEqual(p, [(1, 0, 100)])

    def test_closest_pair_first_across_marks(self):
        # event 10.6 is within tol of both marks; it must go to the closer one (10.5 s).
        p, _, _ = mesure.apparier([10600], [10100, 10500], 500)
        self.assertEqual(p, [(0, 1, 100)])

    def test_lead_shifts_the_truth_point(self):
        # the rest precedes the mark by ~3 s: with lead 3000 it matches, without it does not.
        self.assertEqual(mesure.apparier([7000], [10000], 500, 0)[0], [])
        self.assertEqual(mesure.apparier([7000], [10000], 500, 3000)[0], [(0, 0, 0)])
        self.assertEqual(mesure.apparier([13000], [10000], 500, 3000)[0], [])

    def test_signed_offset(self):
        p, _, _ = mesure.apparier([10200], [10000], 500)
        self.assertEqual(p[0][2], 200)


class Mesurer(unittest.TestCase):
    def test_fixture_counts(self):
        r = mesure.mesurer(EVENTS, MARKS, REST)
        t = r["total"]
        self.assertEqual((t["tp"], t["fp"], t["fn"]), (3, 3, 1))
        self.assertAlmostEqual(t["f1"], 0.6)
        self.assertEqual(r["ecart_median_ms"], 200)

    def test_event_type_filter(self):
        r = mesure.mesurer(EVENTS, MARKS, REST)
        self.assertEqual(r["total"]["fp"], 3)  # dice.thrown at 10.0 s is not counted

    def test_empty_cell_is_not_a_mark(self):
        self.assertEqual(mesure.charger_reperes(MARKS, "roll_tick_ms"), [10000, 20000, 30000, 40000])
        self.assertEqual(mesure.charger_reperes(MARKS, "tick_ms"), [14000, 36000, 45000, 50000])

    def test_state_filter(self):
        r = mesure.mesurer(EVENTS, MARKS, REST, etats={"sure"})
        t = r["total"]
        # 30.501 (check) and 40.2 (uncertain) are gone: 40.3 now matches the 40 s mark.
        self.assertEqual((t["tp"], t["fp"], t["fn"]), (3, 1, 1))

    def test_pairs_union(self):
        r = mesure.mesurer(EVENTS, MARKS, REST + [("dice.pickup", "tick_ms")])
        self.assertEqual(set(r["paires"]), {"dice.rest=roll_tick_ms", "dice.pickup=tick_ms"})
        t = r["total"]
        self.assertEqual((t["tp"], t["fp"], t["fn"]), (4, 3, 4))

    def test_ms_rounding(self):
        with tempfile.TemporaryDirectory() as d:
            p = os.path.join(d, "e.json")
            with open(p, "w") as fh:
                json.dump([{"type": "dice.rest", "mediaTime": 10.5, "state": "sure"}], fh)
            self.assertEqual(mesure.charger_evenements(p), [("dice.rest", 10500)])
        self.assertEqual(mesure.ms(30.501), 30501)

    def test_empty_is_perfect(self):
        self.assertEqual(mesure.scores(0, 0, 0)["f1"], 1.0)
        self.assertIsNone(mesure.scores(0, 0, 3)["precision"])


class Navigation(unittest.TestCase):
    def test_model(self):
        reperes = [10000, 20000, 30000, 40000]
        paires = [(0, 0, 0), (1, 1, 500), (2, 3, -200)]
        sans, avec = mesure.gain_navigation(reperes, paires, 2000)
        self.assertEqual(sans, 40000)
        # marks 0, 1, 3 matched (jump + |offset|); mark 2 unmatched: full 10 s traversal.
        self.assertEqual(avec, (2000 + 0) + (2000 + 500) + 10000 + (2000 + 200))

    def test_fixture_gain(self):
        n = mesure.mesurer(EVENTS, MARKS, REST)["navigation"]
        self.assertEqual((n["sans_ms"], n["avec_ms"]), (40000, 16700))
        self.assertAlmostEqual(n["gain"], 1 - 16700 / 40000)

    def test_no_marks_no_gain(self):
        with tempfile.TemporaryDirectory() as d:
            p = os.path.join(d, "m.csv")
            with open(p, "w") as fh:
                fh.write("n,roll_tick_ms,tick_ms,dés\n")
            self.assertIsNone(mesure.mesurer(EVENTS, p, REST)["navigation"]["gain"])


class Cli(unittest.TestCase):
    def test_report(self):
        import contextlib
        import io

        buf = io.StringIO()
        with contextlib.redirect_stdout(buf):
            self.assertEqual(mesure.main(["x", EVENTS, MARKS]), 0)
        self.assertIn("| total | 3 | 3 | 1 |", buf.getvalue())
        self.assertIn("gain 58.2 %", buf.getvalue())


if __name__ == "__main__":
    unittest.main()
