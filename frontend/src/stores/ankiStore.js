import { writable } from 'svelte/store';

// Store for all Anki decks
/** @type {import('svelte/store').Writable<any[]>} */
export const ankiDecksStore = writable([]);

// Store for the currently selected deck
/** @type {import('svelte/store').Writable<any>} */
export const selectedAnkiDeckStore = writable(null);

// Store for current review card (AnkiReviewCard from backend)
/** @type {import('svelte/store').Writable<any>} */
export const ankiReviewCardStore = writable(null);

// Store for deck stats
/** @type {import('svelte/store').Writable<any>} */
export const ankiDeckStatsStore = writable(null);

// Store for review mode ('list' = deck list, 'review' = reviewing cards, 'settings' = deck settings)
export const ankiViewModeStore = writable('list');

// Store for routing review key presses from App.svelte to AnkiPanel (rating 1-4, or 'back')
export const ankiReviewActionStore = writable(null);

// Store for paused review session: { deckId, sessionCount } or null
/** @type {import('svelte/store').Writable<any>} */
export const ankiPausedSessionStore = writable(null);

// Whether the answer of the card under review is shown (ADR-0025): one boolean, one question.
// Here because TabbedPanel remounts on every tab switch, and looking at the Eval panel must not
// re-hide it; only a change of question does (rule 5).
export const ankiAnswerShownStore = writable(false);

// Named helpers rather than bare .set(): the two call sites that matter are a
// key press and a change of card, and reading `hideAnkiAnswer()` at the top of
// advance() says what the line is for.
export function showAnkiAnswer() {
    ankiAnswerShownStore.set(true);
}

export function hideAnkiAnswer() {
    ankiAnswerShownStore.set(false);
}
