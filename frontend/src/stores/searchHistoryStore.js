import { writable } from 'svelte/store';

// How many entries the search history keeps, oldest dropped first — shared by
// the two places that append to it (commandProcessor.js's `s`/`ss` commands
// and SearchPanel.svelte's own submit path) so the cap can't drift between
// them (#202).
export const MAX_SEARCH_HISTORY = 100;

// Store for search history
// Each entry contains: { command: string, position: object, timestamp: number }
export const searchHistoryStore = writable([]);

// Store for the last executed search (command + position JSON string)
// Used by AnkiPanel to create search-based decks
export const lastSearchStore = writable(null);
