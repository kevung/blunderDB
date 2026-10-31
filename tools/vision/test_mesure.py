import contextlib
import io
import json
import os
import tempfile
import unittest

import mesure

DIR = os.path.join(os.path.dirname(__file__), "fixtures")
EVENTS = os.path.join(DIR, "events.json")
MARKS = os.path.join(DIR, "marks.csv")
REST = [("dice.rest", "roll_tick_ms")]
BOTH = REST + [("dice.pickup", "tick_ms")]
W = [(10000, 20000)]


class Apparier(unittest.TestCase):
    def test_inside_window_is_zero_distance(self):
        self.assertEqual(mesure.apparier([15000], W, 500)[0], [(0, 0, 0)])

    def test_window_bounds_are_inclusive(self):
        self.assertEqual(mesure.apparier([10000], W, 500)[0], [(0, 0, 0)])
        self.assertEqual(mesure.apparier([20000], W, 500)[0], [(0, 0, 0)])

    def test_lower_bound_is_a_bound(self):
        # a point-like window at the mark only would miss this; the start must count.
        self.assertEqual(mesure.apparier([12000], [(10000, 20000)], 500)[0], [(0, 0, 0)])
        self.assertEqual(mesure.apparier([9400], W, 500)[0], [])

    def test_upper_bound_is_a_bound(self):
        self.assertEqual(mesure.apparier([20400], W, 500)[0], [(0, 0, 400)])
        self.assertEqual(mesure.apparier([20600], W, 500)[0], [])

    def test_tolerance_inclusive_on_both_sides(self):
        self.assertEqual(mesure.apparier([9500], W, 500)[0], [(0, 0, 500)])
        self.assertEqual(mesure.apparier([20500], W, 500)[0], [(0, 0, 500)])

    def test_tolerance_exclusive_beyond(self):
        self.assertEqual(mesure.apparier([9499], W, 500)[0], [])
        self.assertEqual(mesure.apparier([20501], W, 500)[0], [])

    def test_one_to_one(self):
        p, nd, nr = mesure.apparier([12000, 13000], W, 500)
        self.assertEqual((len(p), nd, nr), (1, 2, 1))
        p, nd, nr = mesure.apparier([15000], [(10000, 20000), (14000, 16000)], 500)
        self.assertEqual((len(p), nd, nr), (1, 1, 2))

    def test_closest_window_wins_not_first_listed(self):
        # one event, two windows within tolerance: A is listed first (distance 400), B is closer.
        p, _, _ = mesure.apparier([10400], [(5000, 10000), (10600, 15000)], 500)
        self.assertEqual(p, [(0, 1, 200)])
        p, _, _ = mesure.apparier([10300], [(5000, 10000), (10450, 15000)], 500)
        self.assertEqual(p, [(0, 1, 150)])

    def test_contested_event_goes_to_the_closest_pair(self):
        # two events, two windows: each event takes the window it is inside or closest to.
        p, _, _ = mesure.apparier([10100, 10400], [(5000, 10000), (10600, 15000)], 500)
        self.assertEqual(sorted(p), [(0, 0, 100), (1, 1, 200)])


class Fenetres(unittest.TestCase):
    def test_roll_window_starts_at_previous_action(self):
        self.assertEqual(mesure.charger_fenetres(MARKS, "roll_tick_ms"), [(5000, 10000), (14000, 20000), (None, 30000), (36000, 40000)])

    def test_action_window_starts_at_roll(self):
        self.assertEqual(mesure.charger_fenetres(MARKS, "tick_ms"), [(10000, 14000), (30000, 36000), (40000, 45000)])

    def test_backwards_window_is_not_evaluable(self):
        with tempfile.TemporaryDirectory() as d:
            p = os.path.join(d, "m.csv")
            with open(p, "w") as fh:
                fh.write("n,roll_tick_ms,tick_ms,dés,prev_tick_ms\n1,10000,,31,12000\n")
            self.assertEqual(mesure.charger_fenetres(p, "roll_tick_ms"), [(None, 10000)])

    def test_action_without_roll_uses_previous_action(self):
        with tempfile.TemporaryDirectory() as d:
            p = os.path.join(d, "m.csv")
            with open(p, "w") as fh:
                fh.write("n,roll_tick_ms,tick_ms,dés,prev_tick_ms\n1,,9000,,4000\n")
            self.assertEqual(mesure.charger_fenetres(p, "tick_ms"), [(4000, 9000)])


class Mesurer(unittest.TestCase):
    def test_fixture_counts(self):
        r = mesure.mesurer(EVENTS, MARKS, REST)
        t = r["total"]
        self.assertEqual((t["tp"], t["fp"], t["fn"]), (3, 4, 0))
        self.assertAlmostEqual(t["f1"], 0.6)
        self.assertEqual(r["sans_fenetre"], 1)
        self.assertEqual(r["distance_mediane_ms"], 0)

    def test_event_type_filter(self):
        # dice.thrown at 7.0 s would be a fifth false positive, dice.pickup a sixth
        self.assertEqual(mesure.mesurer(EVENTS, MARKS, REST)["total"]["fp"], 4)

    def test_state_filter(self):
        t = mesure.mesurer(EVENTS, MARKS, REST, etats={"sure"})["total"]
        self.assertEqual((t["tp"], t["fp"], t["fn"]), (2, 2, 1))

    def test_pairs_union(self):
        r = mesure.mesurer(EVENTS, MARKS, BOTH)
        self.assertEqual(set(r["paires"]), {"dice.rest=roll_tick_ms", "dice.pickup=tick_ms"})
        t = r["total"]
        self.assertEqual((t["tp"], t["fp"], t["fn"]), (4, 4, 2))
        self.assertEqual(r["distance_mediane_ms"], 100)

    def test_ms_rounding(self):
        with tempfile.TemporaryDirectory() as d:
            p = os.path.join(d, "e.json")
            with open(p, "w") as fh:
                json.dump([{"type": "dice.rest", "mediaTime": 10.5, "state": "sure"}], fh)
            self.assertEqual(mesure.charger_evenements(p), [("dice.rest", 10500)])
        self.assertEqual(mesure.ms(30.501), 30501)

    def test_empty_cell_is_not_a_mark(self):
        self.assertEqual(len(mesure.charger_fenetres(MARKS, "tick_ms")), 3)

    def test_empty_is_perfect(self):
        self.assertEqual(mesure.scores(0, 0, 0)["f1"], 1.0)
        self.assertIsNone(mesure.scores(0, 0, 3)["precision"])


class Navigation(unittest.TestCase):
    def test_model(self):
        cibles = [(10000, 0), (20000, 400), (30000, None), (40000, 0)]
        sans, avec = mesure.gain_navigation(cibles, 0, 2000)
        self.assertEqual(sans, 40000)
        self.assertEqual(avec, 2000 + 2400 + 10000 + 2000)

    def test_false_positive_costs_a_jump(self):
        self.assertEqual(mesure.gain_navigation([(10000, 0)], 3, 2000)[1], 2000 + 3 * 2000)

    def test_unsorted_targets_are_read_in_time_order(self):
        self.assertEqual(mesure.gain_navigation([(20000, None), (10000, None)], 0, 2000), (20000, 20000))

    def test_fixture_gain(self):
        n = mesure.mesurer(EVENTS, MARKS, REST)["navigation"]
        self.assertEqual((n["sans_ms"], n["avec_ms"]), (40000, 24400))
        self.assertAlmostEqual(n["gain"], 1 - 24400 / 40000)

    def test_traversal_is_shared_between_pairs(self):
        # Separate traversals would give 40 + 25 s; one timeline over 7 marks gives 45 s.
        n = mesure.mesurer(EVENTS, MARKS, BOTH)["navigation"]
        self.assertEqual((n["sans_ms"], n["avec_ms"]), (45000, 37600))

    def test_no_marks_no_gain(self):
        with tempfile.TemporaryDirectory() as d:
            p = os.path.join(d, "m.csv")
            with open(p, "w") as fh:
                fh.write("n,roll_tick_ms,tick_ms,dés,prev_tick_ms\n")
            self.assertIsNone(mesure.mesurer(EVENTS, p, REST)["navigation"]["gain"])


class Cli(unittest.TestCase):
    def test_report(self):
        buf = io.StringIO()
        with contextlib.redirect_stdout(buf):
            self.assertEqual(mesure.main(["x", EVENTS, MARKS]), 0)
        self.assertIn("| total | 3 | 4 | 0 |", buf.getvalue())
        self.assertIn("gain 39.0 %", buf.getvalue())


if __name__ == "__main__":
    unittest.main()
