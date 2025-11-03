import { writable } from 'svelte/store';

// Store for search history
// Each entry contains: { command: string, position: object, timestamp: number }
export const searchHistoryStore = writable([]);
