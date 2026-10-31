import contextlib
import io
import os
import tempfile
import unittest

import prepare_annot
import readable_hist

HERE = os.path.dirname(os.path.abspath(__file__))
MARKS = os.path.join(HERE, "fixtures", "marks.csv")  # roll_tick_ms 10000, 20000, 30000, 40000


class Summary(unittest.TestCase):
    def test_percentile_interpolates_between_ranks(self):
        v = [0, 1000, 2000, 3000, 4000]
        self.assertEqual(readable_hist.percentile(v, 50), 2000)
        self.assertEqual(readable_hist.percentile(v, 10), 400)
        self.assertEqual(readable_hist.percentile(v, 90), 3600)
        self.assertEqual(readable_hist.percentile([0, 1000], 50), 500)

    def test_summary_sorts_before_taking_quantiles(self):
        s = readable_hist.summary([3000, 0, 4000, 1000, 2000])
        self.assertEqual((s["median"], s["p10"], s["p90"]), (2000, 400, 3600))

    def test_outside_is_strict_and_both_sides(self):
        # -1000 and +2000 are inside; -1001 and +2001 are outside
        s = readable_hist.summary([-1001, -1000, 0, 2000, 2001])
        self.assertEqual(s["outside"], 0.4)
        self.assertEqual(readable_hist.summary([-1000, 2000])["outside"], 0)

    def test_bins_are_half_seconds_floored(self):
        self.assertEqual(readable_hist.bins([-1, 0, 499, 500, -500, -501]), {-2: 1, -1: 2, 0: 2, 1: 1})

    def test_text_hist_labels_in_seconds(self):
        out = readable_hist.text_hist([1000, 1200])
        self.assertEqual(out, " +1.0 s | ## 2".replace(" +1.0", "  +1.0"))


class Deltas(unittest.TestCase):
    def annot(self, body):
        d = tempfile.TemporaryDirectory()
        self.addCleanup(d.cleanup)
        p = os.path.join(d.name, "a.csv")
        with open(p, "w", encoding="utf-8") as fh:
            fh.write("n,t_lisible_ms\n" + body)
        return p

    def test_sign_is_readable_minus_roll(self):
        # readable 1.5 s after the roll mark: positive; 2 s before: negative
        self.assertEqual(readable_hist.read_deltas(MARKS, self.annot("1,11500\n2,18000\n")), [1500, -2000])

    def test_empty_unknown_rows_and_player_clock(self):
        p = self.annot("1,\n99,5\n3,0:31.250\n")
        self.assertEqual(readable_hist.read_deltas(MARKS, p), [1250])

    def test_extra_columns_are_ignored(self):
        d = tempfile.TemporaryDirectory()
        self.addCleanup(d.cleanup)
        p = os.path.join(d.name, "a.csv")
        with open(p, "w", encoding="utf-8") as fh:
            fh.write("n,dés,roll_tick_ms,roll_mmss,t_lisible_ms\n1,31,10000,0:10.000,10400\n")
        self.assertEqual(readable_hist.read_deltas(MARKS, p), [400])

    def test_main_prints_seconds(self):
        buf = io.StringIO()
        d = tempfile.TemporaryDirectory()
        self.addCleanup(d.cleanup)
        with contextlib.redirect_stdout(buf):
            readable_hist.main(["x", MARKS, self.annot("1,11500\n2,21500\n"), os.path.join(d.name, "o.png")])
        self.assertIn("médiane : +1.50 s", buf.getvalue())


class Prepare(unittest.TestCase):
    def test_pick_even_spread_deterministic(self):
        rows = list(range(11))
        self.assertEqual(prepare_annot.pick(rows, 3), [0, 5, 10])
        self.assertEqual(prepare_annot.pick(rows, 11), rows)
        self.assertEqual(prepare_annot.pick(rows, 50), rows)
        self.assertEqual(prepare_annot.pick(rows, 1), [0])
        self.assertEqual(prepare_annot.pick(rows, 2), [0, 10])

    def test_mmss(self):
        self.assertEqual(prepare_annot.mmss(0), "0:00.000")
        self.assertEqual(prepare_annot.mmss(61250), "1:01.250")
        self.assertEqual(prepare_annot.mmss(-500), "-0:00.500")

    def test_parse_time(self):
        self.assertEqual(prepare_annot.parse_time("1500"), 1500)
        self.assertEqual(prepare_annot.parse_time("1:01.250"), 61250)
        self.assertEqual(prepare_annot.parse_time("1:00:01.5"), 3601500)
        self.assertEqual(prepare_annot.parse_time(" 0:02.5 "), 2500)

    def test_clip_command_starts_before_roll_and_clamps(self):
        c = prepare_annot.clip_command("v.mkv", "out", 7, 10000, 5000, 9000)
        self.assertEqual(c, "ffmpeg -y -ss 5.000 -i 'v.mkv' -t 9.000 -an 'out/7.mkv'")
        self.assertIn("-ss 0.000", prepare_annot.clip_command("v.mkv", "out", 7, 1000, 5000, 9000))

    def test_main_writes_template_with_empty_annotation(self):
        with tempfile.TemporaryDirectory() as d:
            out = os.path.join(d, "a.csv")
            buf = io.StringIO()
            with contextlib.redirect_stdout(buf):
                self.assertEqual(
                    prepare_annot.main(["x", MARKS, out, "--n", "2", "--video", "v.mkv"]), 0
                )
            lines = open(out, encoding="utf-8").read().splitlines()
            self.assertEqual(lines[0], "n,dés,roll_tick_ms,roll_mmss,t_lisible_ms")
            self.assertEqual(lines[1], "1,31,10000,0:10.000,")
            self.assertEqual(lines[2], "4,51,40000,0:40.000,")
            self.assertEqual(buf.getvalue().count("ffmpeg"), 2)
            self.assertIn("-ss 5.000", buf.getvalue())


if __name__ == "__main__":
    unittest.main()
