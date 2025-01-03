import { writable } from 'svelte/store';

export const metaStore = writable({
    expectedVersion: '0.2.0' // Example expected version
});
