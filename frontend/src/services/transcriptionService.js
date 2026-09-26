/**
 * transcriptionService.js — the open library's drafts, for panels that only
 * need to know about them. A Transcription is written after every Action
 * (ADR-0045); on reopening, its drafts must also show in the Match panel
 * (integration.md §4). Typing and the open draft stay in the Transcription
 * panel.
 */

import { get } from 'svelte/store';
import { logger } from '../utils/logger.js';
import { ListTranscriptions } from '../../wailsjs/go/database/Database.js';
import { transcriptionListStore, transcriptionHistoryActionStore } from '../stores/transcriptionStore.js';
import { databaseLoadedStore } from '../stores/databaseStore.js';
import { activeTabStore } from '../stores/uiStore.js';

/**
 * Reloads the open library's drafts into `transcriptionListStore`. No draft
 * and a failed query both give an empty list: another file's drafts must never
 * survive an open.
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
 * A draft's list name: its label, else its two players, else `unnamed`.
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
 * Asks the open draft to undo or redo (`Ctrl+Z` / `Ctrl+Maj+Z`, ux.md §3). Only
 * posts the request: the stack is the Go `transcript.Editor`, the round trip
 * the panel's. Silent no-op without an open draft.
 *
 * @param {boolean} redo
 */
export function undoTranscription(redo = false) {
    transcriptionHistoryActionStore.set(redo ? 'redo' : 'undo');
}
