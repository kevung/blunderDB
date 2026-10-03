import { writable } from 'svelte/store';

// Whether an exact duplicate is a plain skip at import. The Database holds it
// for the session and starts false; the store mirrors it for the settings tab.
export const skipDuplicatesStore = writable(false);
