#!/bin/sh
# Mirror of postinstall.sh: refreshes the shared-mime-info and desktop-file
# caches after blunderdb-dbx.xml/blunderdb.desktop are removed, so the .dbx
# association disappears immediately rather than lingering until the next
# full cache rebuild (#241). Best-effort, same reasoning as postinstall.sh.
set -e

if command -v update-mime-database >/dev/null 2>&1; then
    update-mime-database /usr/share/mime || true
fi
if command -v update-desktop-database >/dev/null 2>&1; then
    update-desktop-database /usr/share/applications || true
fi

exit 0
