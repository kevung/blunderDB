import { writable, derived } from 'svelte/store';

export const databasePathStore = writable('');

export const databaseLoadedStore = derived(
    databasePathStore,
    $databasePathStore => $databasePathStore !== ''
);

// ...existing code...
