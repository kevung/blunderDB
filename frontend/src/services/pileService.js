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

// Reads the marker for the position now on the board, by what the library holds: a stored
// position by its id, a draft by its hash — so a draft already put on the Pile keeps its marker
// when the board re-emits it. Asking creates nothing. An answer for a position since left behind
// is dropped.
export async function refreshPileState() {
    const mine = ++sequence;
    if (!get(databasePathStore)) {
        onPileStore.set(false);
        return;
    }
    try {
        const on = await IsPositionOnPile(get(positionStore));
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
        sequence++;
        onPileStore.set(result.onPile);
        setStatusBarMessage(tMsg(result.onPile ? (result.brought ? 'status.pileBrought' : 'status.pileOn') : 'status.pileOff'));
    } catch (error) {
        logger.error('Error toggling the Pile:', error);
        setStatusBarMessage(tMsg('status.pileError'));
    }
}
