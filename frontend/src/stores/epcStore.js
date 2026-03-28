import { writable } from 'svelte/store';

// EPC data store: holds the current EPC computation results
// { bottomEPC: EPCResult | null, topEPC: EPCResult | null, error: string | null }
export const epcDataStore = writable({
    bottomEPC: null,
    topEPC: null,
    error: null
});
