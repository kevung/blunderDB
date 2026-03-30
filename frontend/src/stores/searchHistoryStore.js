import { writable } from 'svelte/store';

// Store for search history
// Each entry contains: { command: string, position: object, timestamp: number }
export const searchHistoryStore = writable([]);

// Store for the last executed search (command + position JSON string)
// Used by AnkiPanel to create search-based decks
export const lastSearchStore = writable(null);
