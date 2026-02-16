import { writable } from 'svelte/store';

// Store for all collections
export const collectionsStore = writable([]);

// Store for currently selected collection (clicked in panel)
export const selectedCollectionStore = writable(null);

// Store for positions in the selected collection
export const collectionPositionsStore = writable([]);

// Store for the active collection in COLLECTION mode (double-clicked)
export const activeCollectionStore = writable(null);
