import { writable } from 'svelte/store';

export const metaStore = writable({
    applicationVersion: '0.4.0',
});
