import { writable } from 'svelte/store';

// Persists the search panel's filter state across tab switches.
// Saved on every search execution; restored when the SearchPanel mounts.
/**
 * @typedef {object} SearchParams
 * @property {ReturnType<typeof import('./positionStore.js').emptyPosition> | null} position board saved with the criteria
 * @property {ReturnType<typeof import('./searchExcludePositionStore.js').emptySearchBoardPosition> | null} excludePosition board of the exclusion structure
 * @property {string} structureMode
 * @property {Record<string, boolean>} filterEnabled
 * @property {boolean} searchInCurrentResults
 * @property {string} searchText
 * @property {string} commentMode
 * @property {string} movePattern
 * @property {number[]} matchIDsSelected
 * @property {number[]} tournamentIDsSelected
 * @property {string} playerName
 * @property {string} diceRollOption
 * @property {string} cubeSubType
 * @property {string} creationDateOption
 * @property {string} creationDateMin
 * @property {string} creationDateMax
 * @property {string} creationDateRangeMin
 * @property {string} creationDateRangeMax
 */
/** @type {import('svelte/store').Writable<(SearchParams & Record<string, unknown>) | null>} */
export const searchParamsStore = writable(null);

// True while the last filter search found nothing: the panel says so where the user is looking,
// the status bar alone is easy to miss.
export const searchEmptyStore = writable(false);
