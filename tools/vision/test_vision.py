import contextlib
import io
import os
import sqlite3
import tempfile
import unittest

import marks_export
import readable_hist


def make_db(path):
    con = sqlite3.connect(path)
    con.executescript(
        """
        CREATE TABLE game (id INTEGER PRIMARY KEY, match_id INTEGER, game_number INTEGER);
        CREATE TABLE move (id INTEGER PRIMARY KEY, game_id INTEGER, move_number INTEGER,
            dice_1 INTEGER, dice_2 INTEGER, roll_tick_ms INTEGER, tick_ms INTEGER);
        INSERT INTO game VALUES (1, 7, 1), (2, 7, 2), (3, 8, 1);
        INSERT INTO move VALUES (1, 1, 0, 3, 1, 1000, 4000), (2, 1, 1, 6, 5, NULL, NULL),
            (3, 2, 0, 2, 2, 9000, NULL), (4, 3, 0, 1, 1, 5, 5);
        """
    )
    con.commit()
    con.close()


class Vision(unittest.TestCase):
    def setUp(self):
        self.dir = tempfile.TemporaryDirectory()
        self.addCleanup(self.dir.cleanup)
        self.db = os.path.join(self.dir.name, "t.db")
        make_db(self.db)

    def test_export(self):
        self.assertEqual(
            marks_export.export_rows(self.db, 7), [(1, 1000, 4000, "31"), (3, 9000, "", "22")]
        )

    def test_export_read_only(self):
        con = sqlite3.connect(
            "file:" + self.db + "?mode=ro", uri=True
        )
        con.close()
        marks_export.export_rows(self.db, 7)
        con = sqlite3.connect(self.db)
        self.assertEqual(con.execute("select count(*) from move").fetchone()[0], 4)

    def test_hist(self):
        marks = os.path.join(self.dir.name, "m.csv")
        annot = os.path.join(self.dir.name, "a.csv")
        with open(marks, "w") as fh:
            marks_export.write_csv(marks_export.export_rows(self.db, 7), fh)
        with open(annot, "w") as fh:
            fh.write("n,t_lisible_ms\n1,1500\n2,\n3,13000\n")
        d = readable_hist.read_deltas(marks, annot)
        self.assertEqual(d, [500, 4000])
        s = readable_hist.summary(d)
        self.assertEqual(s["median"], 2250)
        self.assertEqual(s["outside"], 0.5)
        buf = io.StringIO()
        with contextlib.redirect_stdout(buf):
            self.assertEqual(readable_hist.main(["x", marks, annot, os.path.join(self.dir.name, "o.png")]), 0)
        self.assertIn("médiane", buf.getvalue())


if __name__ == "__main__":
    unittest.main()
