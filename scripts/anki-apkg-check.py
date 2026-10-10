#!/usr/bin/env python3
"""Import .apkg files with Anki's own importer, in order, into a fresh collection.

The oracle of the .apkg export: not a dependency, a check run by hand or by
TestAnkiImportsThePackage when BLUNDERDB_ANKI_PYTHON names a Python that has
the "anki" package (pip install anki). Importing the same deck twice must
update its notes, never duplicate them.

    python3 scripts/anki-apkg-check.py first.apkg [second.apkg ...]
"""

import os
import sys
import tempfile

from anki.collection import Collection
from anki.import_export_pb2 import (
    IMPORT_ANKI_PACKAGE_UPDATE_CONDITION_IF_NEWER,
    ImportAnkiPackageOptions,
    ImportAnkiPackageRequest,
)


def main(paths):
    with tempfile.TemporaryDirectory() as tmp:
        col = Collection(os.path.join(tmp, "collection.anki2"))
        for path in paths:
            log = col.import_anki_package(
                ImportAnkiPackageRequest(
                    package_path=os.path.abspath(path),
                    options=ImportAnkiPackageOptions(
                        merge_notetypes=True,
                        update_notes=IMPORT_ANKI_PACKAGE_UPDATE_CONDITION_IF_NEWER,
                        with_scheduling=False,
                    ),
                )
            ).log
            print(
                f"{os.path.basename(path)}: new={len(log.new)} updated={len(log.updated)} "
                f"duplicate={len(log.duplicate)} conflicting={len(log.conflicting)}"
            )
        notes = col.find_notes("")
        cards = col.find_cards("")
        media = os.listdir(col.media.dir())
        print(f"notes={len(notes)} cards={len(cards)} media={len(media)}")
        for nid in notes:
            note = col.get_note(nid)
            card = note.cards()[0]
            question = card.question()
            if "<img" not in question:
                sys.exit(f"note {note.guid}: no board on the front")
            print(f"  {note.guid} deck={col.decks.name(card.did)} answer={note['Answer']!r}")
        col.close()


if __name__ == "__main__":
    main(sys.argv[1:])
