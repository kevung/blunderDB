// The Pile gesture (CONTEXT.md "Pile"): one key, one toolbar button, and the same gesture takes
// a position off again. It acts on the position the board shows — a stored one, a draft on the
// scratch board, and later a Duel's — through the one backend call, which writes a draft to the
// library first, as a position brought in on its own.
import { writable, get } from 'svelte/store';
import { IsPositionOnPile, TogglePile } from '../../wailsjs/go/database/Database.js';
import { positionStore } from '../stores/positionStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { setStatusBarMessage } from './databaseService.js';
import { tMsg } from '../i18n';
import { logger } from '../utils/logger.js';

// Whether the position on the board is on the Pile: the board's marker and the toolbar button.
export const onPileStore = writable(false);

let sequence = 0;

// Reads the marker for the position now on the board. A draft (id 0) is not on the Pile, and
// asking creates nothing. An answer for a position since left behind is dropped.
export async function refreshPileState() {
    const mine = ++sequence;
    const id = get(positionStore)?.id ?? 0;
    if (!id || !get(databasePathStore)) {
        onPileStore.set(false);
        return;
    }
    try {
        const on = await IsPositionOnPile(id);
        if (mine === sequence) onPileStore.set(!!on);
    } catch (error) {
        logger.error('Pile state:', error);
        if (mine === sequence) onPileStore.set(false);
    }
}

export async function togglePile() {
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('status.noDbOpenedFirst'));
        return;
    }
    const position = get(positionStore);
    try {
        const result = await TogglePile(position);
        // A draft has just been written: the board now shows a stored position's marker, and the
        // position keeps its id 0 until the user saves or reloads — the marker is the answer.
        sequence++;
        onPileStore.set(result.onPile);
        setStatusBarMessage(tMsg(result.onPile ? (result.brought ? 'status.pileBrought' : 'status.pileOn') : 'status.pileOff'));
    } catch (error) {
        logger.error('Error toggling the Pile:', error);
        setStatusBarMessage(tMsg('status.pileError'));
    }
}
