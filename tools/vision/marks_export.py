#!/usr/bin/env python3
"""Export the Repères and dice of one match to CSV: n, roll_tick_ms, tick_ms, dés, prev_tick_ms
(prev_tick_ms : tick_ms de l'action précédente de la même partie, vide si elle n'en a pas).

Usage: marks_export.py <base.db> <match_id> [out.csv]   (stdout without out.csv)
The database is opened read-only. Only moves with a roll_tick_ms are kept;
n is the 1-based rank of the move in the match (game, then move number).
"""
import csv
import sqlite3
import sys
from pathlib import Path

QUERY = """
SELECT g.id, m.roll_tick_ms, m.tick_ms, m.dice_1, m.dice_2
FROM move m JOIN game g ON g.id = m.game_id
WHERE g.match_id = ?
ORDER BY g.game_number, m.move_number, m.id
"""


def export_rows(db_path, match_id):
    con = sqlite3.connect(Path(db_path).resolve().as_uri() + "?mode=ro", uri=True)
    try:
        rows = con.execute(QUERY, (match_id,)).fetchall()
    finally:
        con.close()
    out = []
    prev_game, prev_tick = None, None
    for n, (game, roll, tick, d1, d2) in enumerate(rows, 1):
        if game != prev_game:
            prev_game, prev_tick = game, None
        prev = "" if prev_tick is None else prev_tick
        prev_tick = tick
        if roll is None:
            continue
        dice = f"{d1}{d2}" if d1 and d2 else ""
        out.append((n, roll, "" if tick is None else tick, dice, prev))
    return out


def write_csv(rows, fh):
    w = csv.writer(fh, lineterminator="\n")
    w.writerow(["n", "roll_tick_ms", "tick_ms", "dés", "prev_tick_ms"])
    w.writerows(rows)


def main(argv):
    if len(argv) not in (3, 4):
        print(__doc__, file=sys.stderr)
        return 2
    rows = export_rows(argv[1], int(argv[2]))
    if len(argv) == 4:
        with open(argv[3], "w", newline="", encoding="utf-8") as fh:
            write_csv(rows, fh)
    else:
        write_csv(rows, sys.stdout)
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
