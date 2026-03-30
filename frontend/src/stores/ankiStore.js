import { writable } from 'svelte/store';

// Store for all Anki decks
export const ankiDecksStore = writable([]);

// Store for the currently selected deck
export const selectedAnkiDeckStore = writable(null);

// Store for current review card (AnkiReviewCard from backend)
export const ankiReviewCardStore = writable(null);

// Store for deck stats
export const ankiDeckStatsStore = writable(null);

// Store for review mode ('list' = deck list, 'review' = reviewing cards, 'settings' = deck settings)
export const ankiViewModeStore = writable('list');

// Store for routing review key presses from App.svelte to AnkiPanel (rating 1-4, or 'back')
export const ankiReviewActionStore = writable(null);

// Store for paused review session: { deckId, sessionCount } or null
export const ankiPausedSessionStore = writable(null);
