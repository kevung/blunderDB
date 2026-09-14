#!/usr/bin/env python3
"""Compare the RST inline markup of every documentation string with its translations.

Why this exists
---------------

The Sphinx build only warns about markup docutils *refuses loudly* — an
inline start-string without its end-string, a `:ref:` count that changed.
Three families of defects render wrongly and emit nothing at all (#427):

* **nested markup** — ``**bold with ``literal`` inside**`` is not markup in
  RST, in any language: the page shows the inner delimiters as text. The
  French source had it, so all nine sites did;
* **markup dropped by a translation** — a translation that lost its bold, its
  literals or its emphasis, typically because the msgid changed without the
  entry going fuzzy (``doc-i18n-check.sh`` sees nothing: the msgstr is not
  empty), or because an opening delimiter glued to a CJK letter is silently
  ignored;
* **links dropped by a translation** — a `` `text <url>`_ ``, a bare URL, a
  `:ref:` or a `:doc:` present in the source and gone from the translation.
  A bare URL is also lost *without being deleted*: docutils gives up on the
  standalone-URI rule for a whole paragraph as soon as a word like the
  Finnish ``blunderDB:n`` looks like an unknown URI scheme.

What it does
------------

Every string is parsed on its own by docutils' reStructuredText parser — the
parser only, no transforms, no Sphinx build: it takes seconds. That is what
Sphinx itself does with a msgstr (``publish_msgstr``), so a string that
parses wrongly here renders wrongly on the site. Reported:

``source``   a French msgid whose markup does not render: a docutils warning,
             or a delimiter left in the visible text (nesting).
``markup``   the same, in a msgstr.
``classes``  a msgstr that lost markup its msgid carries: bold (whether there
             is any — a translation may reorder ``**a** ``x`` **b**`` into one
             bold run) or inline literals (how many — a literal is a command,
             a flag, a path: no translation has a reason to drop one, and a
             missing one is how a msgid that grew without going fuzzy shows).
             Emphasis is not compared: the French source uses it for a foreign
             word or an interface label — ``(*flag*)``, ``onglet *Sauf*`` —
             that a translation rightly glosses away or puts in its own
             quotation marks. An emphasis docutils refuses still shows as a
             leaked ``*`` under ``markup``.
``links``    a link of the msgid missing from the msgstr: URIs and :ref:/:doc:
             targets by value (a target is never translated), named references
             by count (Sphinx maps a translated reference name back onto the
             original target).

Empty and fuzzy msgstr are skipped: Sphinx shows the French text for them, and
``doc-i18n-check.sh`` already reports them.

Usage
-----

    scripts/doc-markup-check.py [--pot-dir doc/build/gettext] [--langs "en de …"]

``--pot-dir`` checks the French sources through freshly generated .pot
templates (``doc-i18n-check.sh`` passes the ones it has just built); without it,
the msgids of the committed catalogues stand in for them. Exit status 1 when
anything is reported.
"""

import argparse
import collections
import glob
import os
import re
import sys

from babel.messages.pofile import read_po
from docutils import nodes, utils
from docutils.frontend import get_default_settings
from docutils.parsers.rst import Parser, roles

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
LOCALE = os.path.join(ROOT, "doc", "source", "locale")
LANGS = "en de el es fi it ja ru"


# A string holding none of these cannot carry inline markup or a link, and is
# not parsed: `*` and `` ` `` open every inline construct, `\` escapes, `://`
# and `@` are the standalone URI and e-mail rules, `]_` a footnote reference.
MARKUP_HINT = re.compile(r"[*`\\@]|://|\]_|\|\w")


# --- Sphinx roles --------------------------------------------------------------
#
# docutils does not know Sphinx's roles; unregistered, every :ref: would be an
# "unknown role" error. Cross-reference roles become an `xref` node carrying
# their target; the others a plain inline node.

class xref(nodes.Inline, nodes.TextElement):
    """A Sphinx cross-reference (:ref:, :doc:, …) and its target."""


EXPLICIT_TARGET = re.compile(r"^(.*?)\s*<([^<>]+)>$", re.S)


def xref_role(name, rawtext, text, lineno, inliner, options=None, content=None):
    text = utils.unescape(text)
    m = EXPLICIT_TARGET.match(text)
    title, target = (m.group(1), m.group(2)) if m else (text, text)
    node = xref(rawtext, title.lstrip("~!"), reftype=name, reftarget=target.lstrip("~!"))
    return [node], []


def inline_role(name, rawtext, text, lineno, inliner, options=None, content=None):
    return [nodes.inline(rawtext, utils.unescape(text), classes=[name])], []


for _name in ("ref", "doc", "numref", "download", "term", "any", "keyword", "option"):
    roles.register_local_role(_name, xref_role)
for _name in ("kbd", "guilabel", "menuselection", "file", "samp", "command",
              "program", "envvar", "abbr", "dfn", "mimetype", "regexp"):
    roles.register_local_role(_name, inline_role)


# --- Parsing -------------------------------------------------------------------

SETTINGS = get_default_settings(Parser)
SETTINGS.report_level = 5   # collect the messages in the tree, print nothing
SETTINGS.halt_level = 5
SETTINGS.warning_stream = False
PARSER = Parser()

# Text under these nodes is shown verbatim: delimiters there are content.
VERBATIM = (nodes.literal, nodes.literal_block, nodes.math, nodes.math_block,
            nodes.raw, nodes.comment, nodes.target, nodes.substitution_definition,
            nodes.system_message)

# Delimiters that survived into the visible text. The text is docutils'
# null-escaped form, so an escaped `\*` is "\x00*" and never matches.
LEAKS = [
    ("``", re.compile(r"(?<!\x00)``")),
    ("**", re.compile(r"(?<![\x00*])\*\*")),
    ("*…*", re.compile(r"(?<![\x00*\w])\*[^\s*\x00][^*]*(?<![\s\x00])\*(?![*\w])")),
    ("`…`", re.compile(r"(?<![\x00`])`[^\s`\x00][^`]*(?<![\s\x00])`")),
    (":role:", re.compile(r"(?<![\x00\w]):[a-z]+:(?=`)")),
]


class Rendering:
    """What a string renders as: its warnings, leaked delimiters, markup, links."""

    def __init__(self, text):
        doc = utils.new_document("<po>", SETTINGS)
        PARSER.parse(text, doc)
        self.warnings = [m.astext().split("\n")[0]
                         for m in doc.findall(nodes.system_message)
                         if m["level"] >= 2]
        self.leaks = []
        for lit in doc.findall(nodes.literal):
            # A closing ``...`` docutils refused (glued to `（` or a CJK
            # letter) does not warn when a later one closes the literal: the
            # literal swallows the text in between, delimiters included.
            if "``" in lit.astext():
                self.leaks.append(f"literal swallowing a delimiter: {lit.astext()!r}")
        for t in doc.findall(nodes.Text):
            if any(isinstance(a, VERBATIM) for a in _ancestors(t)):
                continue
            for label, rx in LEAKS:
                if rx.search(str(t)):
                    self.leaks.append(f"{label} in {t.astext()!r}")
        self.strong = sum(1 for _ in doc.findall(nodes.strong))
        self.literals = sum(1 for _ in doc.findall(nodes.literal))
        self.uris = collections.Counter()
        self.xrefs = collections.Counter()
        self.named = 0
        for ref in doc.findall(nodes.reference):
            if "refuri" in ref:
                self.uris[ref["refuri"]] += 1
            else:
                self.named += 1
        for x in doc.findall(xref):
            self.xrefs[f":{x['reftype']}:`{x['reftarget']}`"] += 1

    @property
    def broken(self):
        return self.warnings + self.leaks


def _ancestors(node):
    node = node.parent
    while node is not None:
        yield node
        node = node.parent


_cache = {}


def render(text):
    r = _cache.get(text)
    if r is None:
        r = _cache[text] = Rendering(text)
    return r


def needs_parse(*texts):
    return any(MARKUP_HINT.search(t) for t in texts)


# --- Checks --------------------------------------------------------------------

def compare(msgid, msgstr):
    """Yield (category, message) for a translated entry."""
    if not needs_parse(msgid, msgstr):
        return
    src, tr = render(msgid), render(msgstr)
    for problem in tr.broken:
        if problem not in src.broken:
            yield "markup", problem
    if src.strong and not tr.strong:
        yield "classes", "no bold in the translation"
    if tr.literals < src.literals:
        yield "classes", f"{src.literals - tr.literals} inline literal(s) missing"
    for uri in src.uris - tr.uris:
        yield "links", f"link {uri} missing"
    for target in src.xrefs - tr.xrefs:
        yield "links", f"{target} missing"
    if tr.named < src.named:
        yield "links", f"{src.named - tr.named} named reference(s) missing"


def location(message):
    return ", ".join(f"{os.path.basename(f)}:{n}" for f, n in message.locations[:1]) or "?"


def read_catalogue(path):
    with open(path, "rb") as f:
        return read_po(f, abort_invalid=True)


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    ap.add_argument("--pot-dir", help="check the French sources through these .pot templates")
    ap.add_argument("--langs", default=LANGS, help="space-separated languages (default: all eight)")
    args = ap.parse_args()

    reports = []          # (category, where, message)
    per = collections.Counter()

    # (a) the French source.
    sources = {}
    if args.pot_dir:
        files = sorted(glob.glob(os.path.join(args.pot_dir, "*.pot")))
        if not files:
            print(f"no .pot template in {args.pot_dir}", file=sys.stderr)
            return 2
    else:
        files = sorted(glob.glob(os.path.join(LOCALE, args.langs.split()[0], "LC_MESSAGES", "*.po")))
    for path in files:
        for m in read_catalogue(path):
            if m.id and isinstance(m.id, str):
                sources.setdefault(m.id, location(m))
    for msgid, where in sources.items():
        if needs_parse(msgid):
            for problem in render(msgid).broken:
                reports.append(("source", f"fr {where}", problem))
                per["source", "fr"] += 1

    # (b), (c) every translation.
    for lang in args.langs.split():
        for path in sorted(glob.glob(os.path.join(LOCALE, lang, "LC_MESSAGES", "*.po"))):
            rel = os.path.relpath(path, ROOT)
            for m in read_catalogue(path):
                if not m.id or not isinstance(m.id, str) or not m.string or m.fuzzy:
                    continue
                for category, problem in compare(m.id, m.string):
                    reports.append((category, f"{lang} {location(m)} ({rel}:{m.lineno})", problem))
                    per[category, lang] += 1

    for category, where, problem in reports:
        print(f"[{category}] {where}: {problem}")
    if not reports:
        print("doc markup: every string renders its markup and keeps its links.")
        return 0
    print()
    for (category, lang), n in sorted(per.items()):
        print(f"  {category:8} {lang}: {n}")
    print(f"doc markup: {len(reports)} problem(s) — fix the .rst (source) or the msgstr; "
          "in Japanese, an escaped space `\\ ` (written `\\\\ ` in a .po) where docutils "
          "refuses a boundary next to CJK text.")
    return 1


if __name__ == "__main__":
    sys.exit(main())
