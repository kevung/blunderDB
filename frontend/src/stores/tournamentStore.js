import { writable } from 'svelte/store';
import { databasePathStore } from './databaseStore.js';

/** @typedef {import('../../wailsjs/go/models').domain.Tournament} Tournament */

export const tournamentsStore = writable(/** @type {Tournament[]} */ ([]));
export const selectedTournamentStore = writable(/** @type {Tournament | null} */ (null));
export const tournamentMatchesStore = writable(/** @type {import('../../wailsjs/go/models').domain.Match[]} */ ([]));

/** ID of a tournament another panel asks the Tournaments panel to open; the panel consumes it once its list is loaded. */
export const tournamentOpenRequestStore = writable(/** @type {number | null} */ (null));

/** Tournaments the list is restricted to (a Stats count), or null for all of them. */
export const tournamentIDsFilterStore = writable(/** @type {number[] | null} */ (null));

// The ids name tournaments of one database: another one opens unrestricted.
let shownPath = /** @type {string | undefined} */ (undefined);
databasePathStore.subscribe((path) => {
    if (shownPath !== undefined && path !== shownPath) tournamentIDsFilterStore.set(null);
    shownPath = path;
});
