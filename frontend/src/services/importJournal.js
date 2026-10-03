// latestJournalEntries keeps, per file, its last line: a file tried again by a
// resumed batch shows its latest outcome, not the failure it had before.
export function latestJournalEntries(entries) {
    const last = new Map();
    for (const e of entries ?? []) {
        last.delete(e.path);
        last.set(e.path, e);
    }
    return [...last.values()];
}
