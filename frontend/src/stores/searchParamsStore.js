import { writable } from 'svelte/store';

// Persists the search panel's filter state across tab switches.
// Saved on every search execution; restored when the SearchPanel mounts.
export const searchParamsStore = writable(null);

// True while the last filter search found nothing: the panel says so where the user is looking,
// the status bar alone is easy to miss.
export const searchEmptyStore = writable(false);
