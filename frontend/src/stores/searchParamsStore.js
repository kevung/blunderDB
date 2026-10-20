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

/**
 * @typedef {{ save: () => void, restore: () => void }} SearchForm the mounted panel's criteria:
 *   save writes them into searchParamsStore, restore reads them back from it
 */
/** @type {SearchForm | null} */
let liveForm = null;

/**
 * The Search panel declares its form while mounted. Its criteria live in the panel's own state
 * and reach searchParamsStore only at a search or at unmount; a library switch must still see
 * the criteria being typed, and hand the incoming library's criteria to the form on screen.
 * @param {SearchForm} form
 * @returns {() => void} detach
 */
export function attachSearchForm(form) {
    liveForm = form;
    return () => {
        if (liveForm === form) liveForm = null;
    };
}

/**
 * searchParamsStore as the library groups see it (libraryScope.js): reading it first flushes the
 * form on screen, writing it hands the value to that form. Everything else uses searchParamsStore.
 * @type {import('svelte/store').Writable<(SearchParams & Record<string, unknown>) | null>}
 */
export const searchCriteriaStore = {
    subscribe(run, invalidate) {
        liveForm?.save();
        return searchParamsStore.subscribe(run, invalidate);
    },
    set(value) {
        searchParamsStore.set(value);
        liveForm?.restore();
    },
    update(fn) {
        liveForm?.save();
        searchParamsStore.update(fn);
        liveForm?.restore();
    }
};
