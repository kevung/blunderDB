/**
 * transcriptionService.js — the drafts of the open library, for the panels
 * that only need to KNOW about them.
 *
 * A Transcription is a draft of a match being typed in (ADR-0045), written to
 * its row after every Action so that a crash costs at most the dice half
 * entered. The other half of that promise is the resumption: on reopening a
 * library, the drafts must be visible somewhere the user actually looks — the
 * Match panel, next to the matches they will become (integration.md §4) — and
 * not only in the tab they were typed in.
 *
 * So this module holds the two things such a caller needs: refreshing the
 * list, and naming a line of it. The typing itself, the gestures and the open
 * draft are the Transcription panel's business and stay there.
 */

import { get } from 'svelte/store';
import { logger } from '../utils/logger.js';
import { ListTranscriptions } from '../../wailsjs/go/database/Database.js';
import { transcriptionListStore, transcriptionHistoryActionStore } from '../stores/transcriptionStore.js';
import { databaseLoadedStore } from '../stores/databaseStore.js';
import { activeTabStore } from '../stores/uiStore.js';

/**
 * Reloads the drafts of the open library into `transcriptionListStore`.
 *
 * A library with no draft, and a library that could not be asked, both come
 * out as an empty list: a draft belongs to the library that names its two
 * players, so the list of another file must never survive an open.
 *
 * @returns {Promise<any[]>} the drafts, most recently updated first.
 */
export async function refreshTranscriptionDrafts() {
    if (!get(databaseLoadedStore)) {
        transcriptionListStore.set([]);
        return [];
    }
    try {
        const drafts = (await ListTranscriptions()) ?? [];
        transcriptionListStore.set(drafts);
        return drafts;
    } catch (error) {
        logger.error('Error listing the transcription drafts:', error);
        transcriptionListStore.set([]);
        return [];
    }
}

/**
 * What a draft is called in a list: the label the row carries, else the two
 * players, else the caller's word for a draft that has not said who is
 * playing yet.
 *
 * @param {{label?: string, player1?: string, player2?: string}} draft
 * @param {string} unnamed
 */
export function draftLabel(draft, unnamed) {
    if (draft?.label) return draft.label;
    const players = [draft?.player1, draft?.player2].filter(Boolean);
    return players.length ? players.join(' — ') : unnamed;
}

/** Brings the Transcription tab forward, where a draft is typed. */
export function showTranscriptionTab() {
    activeTabStore.set('transcription');
}

/**
 * Asks the open draft to undo, or to redo (`Ctrl+Z` / `Ctrl+Maj+Z`, ux.md §3).
 *
 * It only POSTS the request: the stack lives in Go, in the draft's
 * `transcript.Editor`, and the round trip belongs to the panel that holds the
 * draft. Nothing happens when no draft is open, which is what the shortcut does
 * everywhere else in the application — nothing, silently.
 *
 * @param {boolean} redo
 */
export function undoTranscription(redo = false) {
    transcriptionHistoryActionStore.set(redo ? 'redo' : 'undo');
}
