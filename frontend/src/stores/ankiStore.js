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

// Whether the answer of the card under review is shown (ADR-0025). A card asks
// one question and gets one grade, so this is one boolean for the whole answer
// — not the Eval panel's three independently revealed zones.
//
// It lives here rather than in AnkiPanel because TabbedPanel destroys and
// remounts its child on every tab switch: component state would re-hide the
// answer as soon as the user went to look at the Eval panel or the comment,
// which is exactly the move this feature exists to encourage. What re-hides it
// is a change of question — the next card, a new session, leaving the review —
// never a change of view (rule 5).
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
