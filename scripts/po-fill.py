#!/usr/bin/env python3
"""Fill the empty (or fuzzy) msgstr of a catalogue WITHOUT re-wrapping the rest.

Why this exists
---------------

`scripts/doc-po-update.sh` regenerates the eight catalogues through
sphinx-intl, which writes them with Babel's line wrapping. Every general
purpose .po library wraps differently — polib's default width is 78, Babel's
is 76, and msgcat's is its own — so loading a catalogue and saving it back
rewrites EVERY entry in the file. The real change then hides inside a
two-thousand-line diff, and CLAUDE.md's rule ("never re-wrap a .po") is what
that costs.

This script edits the file as TEXT. It touches only the entries it is asked
to fill, and it wraps their msgstr with Babel's own `normalize`, so the
result is byte-identical to what sphinx-intl would have written had the
translation been there all along.

Usage
-----

    scripts/po-fill.py <translations.json>

The JSON is `{ "<po file path>": { "<msgid>": "<msgstr>", ... }, ... }`. A
msgid that is not in the file, or whose msgstr is already filled and not
fuzzy, is left alone and reported. An entry filled here loses its `#, fuzzy`
flag, since it has just been translated.
"""

import json
import re
import sys

from babel.messages.pofile import normalize

# An entry is a run of non-blank lines; entries are separated by blank lines.
ENTRY_SEP = "\n\n"

QUOTED = re.compile(r'^"(.*)"$')


def unquote_block(lines):
    """Join the quoted continuation lines of a msgid/msgstr into one string."""
    out = []
    for line in lines:
        m = QUOTED.match(line.strip())
        if not m:
            break
        out.append(m.group(1))
    return "".join(out).encode().decode("unicode_escape").encode("latin-1").decode("utf-8")


def parse_entry(entry):
    """Return (msgid, msgstr, is_fuzzy) for one entry block, or (None, …)."""
    lines = entry.split("\n")
    msgid = msgstr = None
    fuzzy = any(line.startswith("#,") and "fuzzy" in line for line in lines)
    for i, line in enumerate(lines):
        if line.startswith("msgid "):
            msgid = unquote_block([line[len("msgid "):]] + lines[i + 1:])
        elif line.startswith("msgstr "):
            msgstr = unquote_block([line[len("msgstr "):]] + lines[i + 1:])
    return msgid, msgstr, fuzzy


def fill_entry(entry, translation):
    """Return the entry with its msgstr replaced and its fuzzy flag dropped."""
    lines = entry.split("\n")
    out = []
    i = 0
    while i < len(lines):
        line = lines[i]
        if line.startswith("#,"):
            # Keep the other flags; a filled entry is no longer fuzzy.
            flags = [f.strip() for f in line[2:].split(",") if f.strip() != "fuzzy"]
            if flags:
                out.append("#, " + ", ".join(flags))
            i += 1
            continue
        if line.startswith("msgstr "):
            out.append("msgstr " + normalize(translation, prefix="", width=76))
            i += 1
            # Swallow the old continuation lines.
            while i < len(lines) and QUOTED.match(lines[i].strip()):
                i += 1
            continue
        out.append(line)
        i += 1
    return "\n".join(out)


def fill_file(path, translations):
    with open(path, encoding="utf-8") as f:
        text = f.read()
    trailing = "\n" if text.endswith("\n") else ""
    entries = text.rstrip("\n").split(ENTRY_SEP)

    filled, missing = 0, dict(translations)
    for n, entry in enumerate(entries):
        msgid, msgstr, fuzzy = parse_entry(entry)
        if msgid is None or msgid not in translations:
            continue
        if msgstr and not fuzzy:
            missing.pop(msgid, None)
            continue
        entries[n] = fill_entry(entry, translations[msgid])
        missing.pop(msgid, None)
        filled += 1

    with open(path, "w", encoding="utf-8") as f:
        f.write(ENTRY_SEP.join(entries) + trailing)
    return filled, list(missing)


def main():
    if len(sys.argv) != 2:
        print(__doc__, file=sys.stderr)
        return 2
    with open(sys.argv[1], encoding="utf-8") as f:
        wanted = json.load(f)

    status = 0
    for path, translations in wanted.items():
        filled, missing = fill_file(path, translations)
        print(f"{path}: filled {filled}")
        for msgid in missing:
            print(f"  ! not filled (absent, or already translated): {msgid[:70]}")
            status = 1
    return status


if __name__ == "__main__":
    sys.exit(main())
