#!/bin/sh
# Refreshes the shared-mime-info and desktop-file caches so blunderdb-dbx.xml
# (the .dbx file association, #241) and blunderdb.desktop take effect right
# after install rather than only at the next full cache rebuild. Both
# commands are best-effort: neither is guaranteed present on every distro
# (a minimal container image, for one), and a missing cache refresh here is
# cosmetic, not a reason to fail the whole package install.
set -e

if command -v update-mime-database >/dev/null 2>&1; then
    update-mime-database /usr/share/mime || true
fi
if command -v update-desktop-database >/dev/null 2>&1; then
    update-desktop-database /usr/share/applications || true
fi

exit 0
