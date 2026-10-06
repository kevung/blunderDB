/**
 * transcriptionService.js — requests addressed to the open draft from outside
 * the Transcription panel, which owns typing and the draft itself.
 */

import { transcriptionHistoryActionStore } from '../stores/transcriptionStore.js';

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
