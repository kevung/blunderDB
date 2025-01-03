import { writable } from 'svelte/store';

export const metaStore = writable({
    applicationVersion: '0.2.0' // Example application version
});
