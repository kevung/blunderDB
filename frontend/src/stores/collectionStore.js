import { writable } from 'svelte/store';

/** @typedef {import('../../wailsjs/go/models').database.Collection} Collection */

// Store for all collections
export const collectionsStore = writable(/** @type {Collection[]} */ ([]));

// Store for currently selected collection (clicked in panel)
export const selectedCollectionStore = writable(/** @type {Collection | null} */ (null));

// Store for positions in the selected collection: the loaded prefix as `{ id }` rows
export const collectionPositionsStore = writable(/** @type {{ id: number }[]} */ ([]));

// Store for the active collection in COLLECTION mode (double-clicked)
export const activeCollectionStore = writable(/** @type {Collection | null} */ (null));
