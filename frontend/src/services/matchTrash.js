// Trashing a match from the Matches panel or from its sheet: one confirmation, one trip through
// the trash (restorable from the `trash` command), one set of refreshes for what depends on it.
import { get } from 'svelte/store';
import { confirmAction } from './confirmService.js';
import { TrashMatch } from '../../wailsjs/go/database/Database.js';
import { writingOnLibrary, libraryMark } from '../stores/groupActivityStore.js';
import { logger } from '../utils/logger.js';
import { t, tMsg } from '../i18n';
import { matchPanelRefreshTriggerStore, dbMutationCounterStore, positionReloadTriggerStore, statusBarTextStore } from '../stores/uiStore';

/**
 * Asks, trashes, then runs `afterTrash` (what the caller keeps in step: its list, its selection)
 * before the stores that make every dependent panel read again.
 * @param {{ id: number, player1_name?: string, player2_name?: string }} match
 * @param {() => (void | Promise<void>)} [afterTrash]
 */
export async function trashMatchEntry(match, afterTrash) {
    const mark = libraryMark();
    if (!(await confirmAction(get(t)('match.confirmDelete', { player1: match.player1_name, player2: match.player2_name }), { confirmLabel: get(t)('common.delete') }))) return;
    try {
        if (!(await writingOnLibrary(mark, async () => (await TrashMatch(match.id), true)))) return;
        await afterTrash?.();
        matchPanelRefreshTriggerStore.update((n) => n + 1);
        dbMutationCounterStore.update((n) => n + 1);
        // The deleted positions leave the board's list.
        positionReloadTriggerStore.update((n) => n + 1);
        statusBarTextStore.set(tMsg('match.matchDeleted'));
    } catch (error) {
        logger.error('Error deleting match:', error);
        statusBarTextStore.set(tMsg('match.errorDeleting'));
    }
}
