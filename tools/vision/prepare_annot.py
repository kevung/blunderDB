#!/usr/bin/env python3
"""Prepare the moves to annotate for the t_lisible - roll_tick_ms bench.

Usage: prepare_annot.py <marks.csv> <annot.csv> [--n 50] [--offset-ms 0] [--back-ms 5000]
                        [--span-ms 9000] [--video v.mkv --clips dir]
marks.csv comes from marks_export.py. Picks n moves spread evenly over the
match (deterministic: same marks, same selection, no random draw) and writes
annot.csv: n, dés, roll_tick_ms, roll_mmss (where to look in the player), then
the empty t_lisible_ms to fill by hand. Video time = roll_tick_ms + offset-ms
(offset-ms: video time of the match clock's zero). With --video, prints the
ffmpeg command cutting, for each move, the clip [roll - back, roll - back +
span] to clips/<n>.mkv, one per line (run them with `sh`); the tool never
runs ffmpeg itself.
"""
import csv
import os
import sys


def pick(rows, n):
    """n rows spread evenly (first and last included); all of them when fewer."""
    if len(rows) <= n:
        return list(rows)
    if n == 1:
        return [rows[0]]
    return [rows[round(i * (len(rows) - 1) / (n - 1))] for i in range(n)]


def mmss(ms):
    """Player clock m:ss.mmm of a time in milliseconds."""
    sign = "-" if ms < 0 else ""
    ms = abs(ms)
    return f"{sign}{ms // 60000}:{ms % 60000 // 1000:02d}.{ms % 1000:03d}"


def parse_time(text):
    """Annotated time to ms: an integer in ms, or the player clock [h:]m:ss[.mmm]."""
    text = text.strip()
    if ":" not in text:
        return int(text)
    parts = text.split(":")
    secs = float(parts[-1])
    whole = 0
    for p in parts[:-1]:
        whole = whole * 60 + int(p)
    return whole * 60000 + int(round(secs * 1000))


def clip_command(video, clips_dir, n, roll_ms, offset_ms, back_ms, span_ms):
    start = max(0, roll_ms + offset_ms - back_ms)
    return (
        f"ffmpeg -y -ss {start / 1000:.3f} -i '{video}' -t {span_ms / 1000:.3f} "
        f"-an '{os.path.join(clips_dir, str(n))}.mkv'"
    )


def read_marks(path):
    with open(path, newline="", encoding="utf-8") as fh:
        return [r for r in csv.DictReader(fh) if (r.get("roll_tick_ms") or "").strip()]


def main(argv):
    args = argv[1:]
    opts = {"--n": "50", "--offset-ms": "0", "--back-ms": "5000", "--span-ms": "9000", "--video": "", "--clips": "clips"}
    pos = []
    i = 0
    while i < len(args):
        if args[i] in opts and i + 1 < len(args):
            opts[args[i]] = args[i + 1]
            i += 2
        else:
            pos.append(args[i])
            i += 1
    if len(pos) != 2:
        print(__doc__, file=sys.stderr)
        return 2
    n, offset, back, span = (int(opts[k]) for k in ("--n", "--offset-ms", "--back-ms", "--span-ms"))
    rows = pick(read_marks(pos[0]), n)
    with open(pos[1], "w", newline="", encoding="utf-8") as fh:
        w = csv.writer(fh, lineterminator="\n")
        w.writerow(["n", "dés", "roll_tick_ms", "roll_mmss", "t_lisible_ms"])
        for r in rows:
            roll = int(r["roll_tick_ms"])
            w.writerow([r["n"], r.get("dés", ""), roll, mmss(roll + offset), ""])
            if opts["--video"]:
                print(clip_command(opts["--video"], opts["--clips"], r["n"], roll, offset, back, span))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
