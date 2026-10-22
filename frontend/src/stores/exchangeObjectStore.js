// The object the « Autres bases » panel speaks of beside the position on screen: the match,
// tournament, collection or lesson last selected in its panel. A selection cleared in its panel
// clears it here when it was that one; the position stays the panel's subject in any case.
import { writable, get } from 'svelte/store';
import { selectedTournamentStore } from './tournamentStore.js';
import { selectedCollectionStore } from './collectionStore.js';
import { lessonStore } from './lessonStore.js';

/** @typedef {{ kind: 'match' | 'tournament' | 'collection' | 'lesson', id: number, label: string }} ExchangeObject */

export const exchangeObjectStore = writable(/** @type {ExchangeObject | null} */ (null));

/**
 * Notes the selection of kind: a row makes it the panel's object, null forgets it when it was
 * that kind's.
 * @param {ExchangeObject['kind']} kind
 * @param {{ id: number, label: string } | null} row
 */
export function noteSelection(kind, row) {
    if (row?.id) {
        const cur = get(exchangeObjectStore);
        if (cur?.kind !== kind || cur.id !== row.id || cur.label !== row.label) exchangeObjectStore.set({ kind, id: row.id, label: row.label });
        return;
    }
    if (get(exchangeObjectStore)?.kind === kind) exchangeObjectStore.set(null);
}

selectedTournamentStore.subscribe((t) => noteSelection('tournament', t ? { id: t.id, label: t.name } : null));
selectedCollectionStore.subscribe((c) => noteSelection('collection', c ? { id: c.id, label: c.name } : null));
lessonStore.subscribe((l) => {
    const lesson = /** @type {{ id: number, name: string } | null} */ (l);
    noteSelection('lesson', lesson ? { id: lesson.id, label: lesson.name } : null);
});
