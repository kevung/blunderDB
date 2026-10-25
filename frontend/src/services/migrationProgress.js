// The migration a stale library needs when it opens, in place (CTRL-O) or in a new group: its
// current phase and progress go to the status bar, and Escape cancels it (the open then rejects).

import { CancelLibraryOpen } from '../../wailsjs/go/gui/App.js';
import { EventsOn } from '../../wailsjs/runtime/runtime.js';
import { closeOnEscape } from './escapeService.js';
import { statusBarTextStore } from '../stores/uiStore.js';
import { tMsg, translate } from '../i18n';
import { logger } from '../utils/logger.js';

/**
 * The words for a phase the backend reports; a phase this build has no words for shows its id.
 * A table conversion is reported as `<table>_<major>_<minor>`, the table being what it works on.
 * @param {string} phase
 */
export function migrationPhaseLabel(phase) {
    const id = String(phase ?? '').replace(/_\d+_\d+$/, '');
    const key = 'libraryGroups.migrationPhase.' + id;
    const label = translate(key);
    return label === key ? String(phase ?? '') : label;
}

/**
 * Run open (a call that opens a library and may migrate it) with the migration shown and
 * cancellable.
 * @template T
 * @param {() => Promise<T>} open
 * @returns {Promise<T>}
 */
export async function withMigrationProgress(open) {
    const off = EventsOn('library-open:progress', (/** @type {{ phase: string, done: number, total: number }} */ p) => {
        statusBarTextStore.set(tMsg('libraryGroups.migrating', { phase: migrationPhaseLabel(p.phase), done: p.done, total: p.total }));
    });
    const offEscape = closeOnEscape(() => {
        CancelLibraryOpen().catch((err) => logger.error('Error calling CancelLibraryOpen:', err));
    });
    try {
        return await open();
    } finally {
        off?.();
        offEscape();
    }
}
