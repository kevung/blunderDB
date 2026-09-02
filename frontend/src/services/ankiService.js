// Spaced-repetition decks: everything AnkiPanel does that is not drawing —
// what a deck's stored search means, how a deck is (re)synced with its
// source, how a review or cram session walks its cards, and the bookkeeping
// of the stores the rest of the app reads (the board shows the card, the
// status bar counts positions). The panel keeps the UI state (form fields,
// session counter, cram flag) and the status-bar messages; it calls here and
// catches.
//
// Pure helpers first (no I/O, unit-tested as such), then the functions that
// call the Wails bindings and write the stores.

import { get } from 'svelte/store';
import {
    CreateAnkiDeck,
    GetAllAnkiDecks,
    UpdateAnkiDeckParams,
    DeleteAnkiDeck,
    SyncAnkiDeck,
    SyncAnkiDeckWithPositions,
    GetAnkiDeckStats,
    GetAnkiDeckPositions,
    GetNextAnkiCard,
    GetRandomAnkiCard,
    ReviewAnkiCard,
    ResetAnkiDeck,
    GetAllCollections,
    LoadPositionsByFilters
} from '../../wailsjs/go/database/Database.js';
import { ankiDecksStore, selectedAnkiDeckStore, ankiReviewCardStore, ankiDeckStatsStore, ankiViewModeStore } from '../stores/ankiStore.js';
import { collectionsStore } from '../stores/collectionStore.js';
import { positionsStore, positionStore } from '../stores/positionStore.js';
import { currentPositionIndexStore } from '../stores/uiStore.js';
import { parseFilters } from '../commandProcessor.js';
import { buildSearchFilterPayload } from './searchFilterService.js';
import { logger } from '../utils/logger.js';

// ---------------------------------------------------------------------------
// Pure helpers
// ---------------------------------------------------------------------------

/**
 * What a search-backed deck stored as its source. Current decks store a JSON
 * document `{ command, position, ids }`; the first ones stored a plain
 * comma-separated list of position ids, which is read as ids only.
 *
 * @param {string} sourceCommand
 * @returns {{ ids: number[], command: string | null, position: object }}
 */
export function parseSourceCommand(sourceCommand) {
    let data;
    try {
        data = JSON.parse(sourceCommand);
    } catch {
        return { ids: parseLegacyIds(sourceCommand), command: null, position: {} };
    }
    if (!data || typeof data !== 'object') return { ids: [], command: null, position: {} };
    let position = {};
    if (data.position) {
        try {
            position = typeof data.position === 'string' ? JSON.parse(data.position) : data.position;
        } catch {
            position = {};
        }
    }
    return {
        ids: Array.isArray(data.ids) ? data.ids : [],
        command: typeof data.command === 'string' && data.command ? data.command : null,
        position
    };
}

function parseLegacyIds(text) {
    return String(text ?? '')
        .split(',')
        .map((s) => parseInt(s.trim(), 10))
        .filter((n) => !isNaN(n));
}

/**
 * The source document of a deck created from the current search: the command
 * and board that produced it, plus the ids it matched at creation time so a
 * card is never lost when the search stops matching its position.
 *
 * @param {{ command?: string, position?: string } | null} lastSearch
 * @param {number[]} positionIds
 * @returns {string}
 */
export function buildSearchSource(lastSearch, positionIds) {
    if (lastSearch && lastSearch.command) {
        return JSON.stringify({ command: lastSearch.command, position: lastSearch.position, ids: positionIds });
    }
    return JSON.stringify({ ids: positionIds });
}

/** Search results plus every stored id not among them, in that order, without duplicates. */
export function mergeIds(searchIds, storedIds) {
    const seen = new Set();
    const out = [];
    for (const id of [...searchIds, ...storedIds]) {
        if (seen.has(id)) continue;
        seen.add(id);
        out.push(id);
    }
    return out;
}

/** The FSRS card state as a word. */
export function stateLabel(state) {
    switch (state) {
        case 0:
            return 'New';
        case 1:
            return 'Learning';
        case 2:
            return 'Review';
        case 3:
            return 'Relearning';
        default:
            return '?';
    }
}

/** What a deck draws its cards from, as shown in the deck list. */
export function sourceLabel(deck, collections = []) {
    if (deck.sourceType === 'collection') {
        const coll = collections.find((c) => c.id === deck.sourceId);
        return coll ? coll.name : `Collection #${deck.sourceId}`;
    }
    if (!deck.sourceCommand) return 'Search';
    return parseSourceCommand(deck.sourceCommand).command ?? 'Search';
}

/**
 * The count a session starts at: where a paused FSRS session on the same deck
 * left off, otherwise zero. Cram sessions never resume.
 */
export function resumedSessionCount(pausedSession, deck, cram = false) {
    if (cram || !pausedSession || !deck || pausedSession.deckId !== deck.id) return 0;
    return pausedSession.sessionCount;
}

/** A study session needs due cards; a cram session only needs cards. */
export function canStudy(stats) {
    return !!stats && stats.dueCount > 0;
}

export function canCram(stats) {
    return !!stats && stats.totalCount > 0;
}

// ---------------------------------------------------------------------------
// Wails-backed operations
// ---------------------------------------------------------------------------

/** Reload the decks (and the collections the create form offers) into their stores. */
export async function loadDecks() {
    const decks = (await GetAllAnkiDecks()) || [];
    ankiDecksStore.set(decks);
    const colls = (await GetAllCollections()) || [];
    collectionsStore.set(colls);
    return decks;
}

/**
 * Re-run a search-backed deck's stored search and return the position ids
 * its cards should cover: the current results plus every id stored with the
 * deck. Errors are logged and yield an empty list, which `syncDeckCards`
 * treats as "leave the deck alone".
 */
export async function resolveSearchDeckIds(sourceCommand) {
    try {
        const { ids: storedIds, command, position } = parseSourceCommand(sourceCommand);
        if (!command) return storedIds;

        let payload;
        if (command === 's') {
            // Bare `s`: the board structure is the whole filter.
            payload = buildSearchFilterPayload(position);
        } else {
            const filters = command
                .slice(1)
                .trim()
                .split(' ')
                .map((f) => f.trim());
            payload = buildSearchFilterPayload(position, parseFilters(filters, command), filters);
        }
        const results = await LoadPositionsByFilters(payload);
        return mergeIds(
            (results || []).map((p) => p.id),
            storedIds
        );
    } catch (e) {
        logger.error('Error executing search for deck sync:', e);
        return [];
    }
}

/** Bring a deck's cards in line with its source (collection or stored search). */
export async function syncDeckCards(deck) {
    if (deck.sourceType === 'search' && deck.sourceCommand) {
        const ids = await resolveSearchDeckIds(deck.sourceCommand);
        if (ids.length > 0) await SyncAnkiDeckWithPositions(deck.id, ids);
    } else {
        await SyncAnkiDeck(deck.id);
    }
}

/** Sync every deck (each failure logged, the others still synced) and reload the list. */
export async function syncAllDecksAndReload() {
    try {
        const decks = (await GetAllAnkiDecks()) || [];
        for (const deck of decks) {
            try {
                await syncDeckCards(deck);
            } catch (e) {
                logger.error(`Error syncing deck "${deck.name}":`, e);
            }
        }
        await loadDecks();
    } catch (e) {
        logger.error('Error auto-syncing decks:', e);
    }
}

/**
 * Create a deck and fill it from its source.
 * @returns {Promise<number>} the new deck's id
 */
export async function createDeck({ name, sourceType, sourceId, lastSearch = null, positionIds = [] }) {
    const search = sourceType === 'search';
    const sourceCommand = search ? buildSearchSource(lastSearch, positionIds) : '';
    const deckId = await CreateAnkiDeck(name, '', sourceType, search ? 0 : sourceId, sourceCommand);
    await syncDeckCards({ id: deckId, sourceType, sourceCommand });
    await loadDecks();
    return deckId;
}

/** Delete a deck; if it was the selected one, leave the detail and review views. */
export async function deleteDeck(deckId) {
    await DeleteAnkiDeck(deckId);
    if (get(selectedAnkiDeckStore)?.id === deckId) clearSelection();
    await loadDecks();
}

export function clearSelection() {
    selectedAnkiDeckStore.set(null);
    ankiReviewCardStore.set(null);
    ankiDeckStatsStore.set(null);
    ankiViewModeStore.set('list');
}

export async function refreshDeckStats(deckId) {
    const stats = await GetAnkiDeckStats(deckId);
    ankiDeckStatsStore.set(stats);
    return stats;
}

/** Select a deck: its stats, and its positions as the list the status bar counts. */
export async function selectDeck(deck) {
    selectedAnkiDeckStore.set(deck);
    await refreshDeckStats(deck.id);
    const deckPositions = (await GetAnkiDeckPositions(deck.id)) || [];
    positionsStore.set(deckPositions);
    if (deckPositions.length > 0) currentPositionIndexStore.set(0);
}

/** Wipe a deck's schedule; refresh its stats if it is the selected one. */
export async function resetDeck(deckId) {
    await ResetAnkiDeck(deckId);
    await loadDecks();
    if (get(selectedAnkiDeckStore)?.id === deckId) await refreshDeckStats(deckId);
}

/** Save FSRS parameters and refresh the selected deck from the reloaded list. */
export async function saveDeckParams(deckId, { requestRetention, maximumInterval, enableFuzz }) {
    await UpdateAnkiDeckParams(deckId, requestRetention, maximumInterval, enableFuzz);
    const decks = await loadDecks();
    const updated = decks.find((d) => d.id === deckId);
    if (updated && get(selectedAnkiDeckStore)?.id === deckId) selectedAnkiDeckStore.set(updated);
}

/** Put a card's position on the board and point the status bar at it. */
export function showCard(card) {
    positionStore.set(JSON.parse(JSON.stringify(card.position)));
    const idx = positionsStore.indexOf(card.position.id);
    if (idx >= 0) currentPositionIndexStore.set(idx);
}

/**
 * Start a session on a deck: resync it, load its positions, draw the first
 * card (the next due one, or a random one when cramming) and show it.
 * @returns {Promise<object | null>} the first card, or null when there is none
 */
export async function startSession(deck, { cram = false } = {}) {
    await syncDeckCards(deck);
    positionsStore.set((await GetAnkiDeckPositions(deck.id)) || []);
    const card = cram ? await GetRandomAnkiCard(deck.id, 0) : await GetNextAnkiCard(deck.id);
    if (!card) return null;
    ankiReviewCardStore.set(card);
    showCard(card);
    return card;
}

/**
 * Grade the current card and move to the next due one.
 * @returns {Promise<object | null>} the next card, or null when the session is over
 */
export async function reviewCard(card, rating) {
    const next = await ReviewAnkiCard(card.card.id, rating);
    return advance(next);
}

/** Draw the next random card of a cram session, never the one just shown. */
export async function nextCramCard(deck, card) {
    const next = await GetRandomAnkiCard(deck.id, card.position.id);
    return advance(next);
}

function advance(next) {
    if (next) {
        ankiReviewCardStore.set(next);
        showCard(next);
    } else {
        ankiReviewCardStore.set(null);
    }
    return next ?? null;
}
