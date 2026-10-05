import { writable, get } from 'svelte/store';
import { positionsStore, matchContextStore, browsingLibrary, listedIds } from './positionStore.js';
import { libraryCountsStore } from './libraryCountsStore.js';

// Les blunders de la liste à l'écran (recherche, collection, match, deck), à côté de ceux de la
// bibliothèque. Compté par le chemin de la recherche dans une liste (`E>x` restreint aux
// identifiants de la liste, comme `ss`), donc au seuil de la bibliothèque et selon la même règle
// que le lien qui les ouvre. Une liste trop longue n'est pas comptée : mieux vaut « ? » qu'un
// décompte qui fait attendre.

/** Longueur au-delà de laquelle une liste n'est pas comptée. */
export const MAX_COUNTED_LIST = 20000;

/** @type {import('svelte/store').Writable<number | null>} null : pas de compte propre à la liste (bibliothèque entière, liste trop longue ou échec). */
export const listBlunderCountStore = writable(null);

let generation = 0;

/** Recompte les blunders de la liste à l'écran ; sans objet quand la liste est la bibliothèque. */
export async function refreshListBlunderCount() {
    const mine = ++generation;
    const counts = get(libraryCountsStore);
    const done = (/** @type {number | null} */ n) => {
        if (mine === generation) listBlunderCountStore.set(n);
    };
    if (!counts || counts.blunders == null || browsingLibrary()) return done(null);

    const match = get(matchContextStore);
    try {
        /** @type {number[]} */
        let ids;
        if (match.isMatchMode) {
            ids = [...new Set(match.movePositions.map((m) => m.position?.id).filter((id) => id != null))];
        } else {
            if (get(positionsStore).length > MAX_COUNTED_LIST) return done(null);
            ids = await listedIds();
        }
        if (ids.length === 0) return done(0);
        if (ids.length > MAX_COUNTED_LIST) return done(null);
        const { CountPositionsByFilters } = await import('../../wailsjs/go/database/Database.js');
        const n = await CountPositionsByFilters(/** @type {any} */ ({ moveErrorFilter: `E>${counts.blunderThresholdMP}`, restrictToPositionIDs: ids.join(',') }));
        done(Number.isInteger(n) ? n : null);
    } catch {
        done(null);
    }
}
