import { writable } from 'svelte/store';

// Persists the search panel's filter state across tab switches.
// Saved on every search execution; restored when the SearchPanel mounts.
export const searchParamsStore = writable(null);
