#!/usr/bin/env python3
"""Collect, for a merge in progress, the translations both sides hold for the
entries the regenerated catalogues leave empty or fuzzy.

Writes `{ "<po path>": { "<msgid>": "<msgstr>" } }` for scripts/po-fill.py.

Strings are DECODED (PO escapes resolved) here because po-fill.py re-escapes
what it writes: carrying the raw escaped text would double every backslash
(`\\ ` of reStructuredText in the Japanese catalogues).

Usage: scripts/merge-carry-translations.py <out.json>
       scripts/merge-carry-translations.py --selftest
"""
import json
import pathlib
import re
import subprocess
import sys


def decode(raw):
    """Join the quoted lines of a msgid/msgstr and resolve its PO escapes."""
    joined = "".join(re.findall(r'^"(.*)"$', raw.strip(), re.M))
    return joined.encode().decode("unicode_escape").encode("latin-1").decode("utf-8")


def blocks(text):
    for b in text.split("\n\n"):
        i = re.search(r'^msgid ((?:".*"\n?)+)', b, re.M)
        s = re.search(r'^msgstr ((?:".*"\n?)+)', b, re.M)
        if i:
            yield b, decode(i.group(1)), decode(s.group(1)) if s else ""


def entries(text):
    return {k: v for _, k, v in blocks(text) if k and v}


def wanted(text):
    """msgids to (re)place: empty msgstr, or fuzzy entry."""
    return {k for b, k, v in blocks(text) if k and (not v or "#, fuzzy" in b)}


def main(out):
    table = {}
    for p in pathlib.Path("doc/source/locale").rglob("*.po"):
        path, txt = str(p), p.read_text(encoding="utf-8")
        need = wanted(txt)
        if not need:
            continue
        for rev in ("HEAD", "MERGE_HEAD"):
            r = subprocess.run(["git", "show", f"{rev}:{path}"], capture_output=True, text=True)
            if r.returncode:
                continue
            src = entries(r.stdout)
            for k in need:
                if k in src:
                    table.setdefault(path, {}).setdefault(k, src[k])
    pathlib.Path(out).write_text(json.dumps(table, ensure_ascii=False), encoding="utf-8")
    print(sum(len(v) for v in table.values()), "traductions reposées")


def selftest():
    sys.path.insert(0, str(pathlib.Path(__file__).parent))
    import importlib.util
    spec = importlib.util.spec_from_file_location("pofill", pathlib.Path(__file__).parent / "po-fill.py")
    pofill = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(pofill)
    import tempfile
    src = 'msgid "a \\\\ b"\nmsgstr "x\\\\ y é"\n'
    got = entries(src)
    assert got == {"a \\ b": "x\\ y é"}, got
    with tempfile.TemporaryDirectory() as d:
        po = pathlib.Path(d, "t.po")
        po.write_text('msgid "a \\\\ b"\nmsgstr ""\n', encoding="utf-8")
        pofill.fill_file(str(po), got)
        out = po.read_text(encoding="utf-8")
        assert 'msgstr "x\\\\ y é"' in out and "\\\\\\\\" not in out, out
    print("ok")


if __name__ == "__main__":
    if sys.argv[1:] == ["--selftest"]:
        selftest()
    elif len(sys.argv) == 2:
        main(sys.argv[1])
    else:
        print(__doc__, file=sys.stderr)
        sys.exit(2)
