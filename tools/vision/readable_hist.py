#!/usr/bin/env python3
"""Histogram of t_lisible - roll_tick_ms.

Usage: readable_hist.py <marks.csv> <annot.csv> [out.png]
marks.csv comes from marks_export.py; annot.csv holds n, t_lisible_ms typed by
hand. Rows with an empty t_lisible_ms are ignored. Prints median, p10, p90 and
the share outside [-1 s ; +2 s]; plots with matplotlib when available
(out.png, default readable_hist.png), else a text histogram.
"""
import csv
import math
import sys

LOW_MS, HIGH_MS = -1000, 2000
BIN_MS = 500


def read_deltas(marks_path, annot_path):
    with open(marks_path, newline="", encoding="utf-8") as fh:
        roll = {int(r["n"]): int(r["roll_tick_ms"]) for r in csv.DictReader(fh)}
    deltas = []
    with open(annot_path, newline="", encoding="utf-8") as fh:
        for r in csv.DictReader(fh):
            t = (r.get("t_lisible_ms") or "").strip()
            if not t or not (r.get("n") or "").strip():
                continue
            n = int(r["n"])
            if n in roll:
                deltas.append(int(t) - roll[n])
    return deltas


def percentile(sorted_vals, p):
    """Linear interpolation between closest ranks."""
    k = (len(sorted_vals) - 1) * p / 100
    lo, hi = math.floor(k), math.ceil(k)
    return sorted_vals[lo] + (sorted_vals[hi] - sorted_vals[lo]) * (k - lo)


def summary(deltas):
    s = sorted(deltas)
    out = sum(1 for d in s if d < LOW_MS or d > HIGH_MS)
    return {
        "count": len(s),
        "median": percentile(s, 50),
        "p10": percentile(s, 10),
        "p90": percentile(s, 90),
        "outside": out / len(s),
    }


def bins(deltas):
    counts = {}
    for d in deltas:
        counts[math.floor(d / BIN_MS)] = counts.get(math.floor(d / BIN_MS), 0) + 1
    return counts


def text_hist(deltas):
    counts = bins(deltas)
    lines = []
    for b in range(min(counts), max(counts) + 1):
        c = counts.get(b, 0)
        lines.append(f"{b * BIN_MS / 1000:+6.1f} s | {'#' * c} {c}")
    return "\n".join(lines)


def plot(deltas, path):
    try:
        import matplotlib

        matplotlib.use("Agg")
        import matplotlib.pyplot as plt
    except ImportError:
        return False
    fig, ax = plt.subplots()
    lo, hi = min(deltas), max(deltas)
    edges = list(range(math.floor(lo / BIN_MS) * BIN_MS, hi + 2 * BIN_MS, BIN_MS))
    ax.hist([d / 1000 for d in deltas], bins=[e / 1000 for e in edges])
    ax.axvspan(LOW_MS / 1000, HIGH_MS / 1000, alpha=0.12)
    ax.set_xlabel("t_lisible - roll_tick_ms (s)")
    ax.set_ylabel("coups")
    fig.savefig(path, dpi=120)
    return True


def main(argv):
    if len(argv) not in (3, 4):
        print(__doc__, file=sys.stderr)
        return 2
    deltas = read_deltas(argv[1], argv[2])
    if not deltas:
        print("aucune annotation renseignée", file=sys.stderr)
        return 1
    s = summary(deltas)
    print(f"coups annotés : {s['count']}")
    print(f"médiane : {s['median'] / 1000:+.2f} s")
    print(f"p10     : {s['p10'] / 1000:+.2f} s")
    print(f"p90     : {s['p90'] / 1000:+.2f} s")
    print(f"hors [{LOW_MS / 1000:+.0f} ; {HIGH_MS / 1000:+.0f}] s : {s['outside'] * 100:.0f} %")
    print(text_hist(deltas))
    out = argv[3] if len(argv) == 4 else "readable_hist.png"
    if plot(deltas, out):
        print(f"tracé : {out}")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
