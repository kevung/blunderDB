/**
 * transcriptionStore.js — the open transcription draft, as the panel sees it.
 *
 * The panel is a CLIENT of the Go engine (ADR-0045 rule 9): every gesture goes
 * to `ApplyTranscriptionGesture` and comes back as a whole annotated document —
 * the position of each Action, the scores, the Crawford games, the
 * Inconsistencies, where the Cursor landed. Nothing in this file derives any of
 * that; it holds what came back so the panel, the board and the status bar all
 * read one value.
 *
 * The undo stack is deliberately NOT here. It exists once, in Go, in
 * `transcript.Editor` — the same place the Action being typed lives, because
 * that Entry is not serialisable and a gesture is therefore applied against a
 * live document held on the Go side (see pkg/blunderdb/database/db_transcription.go).
 * What this store keeps of it is what the panel has to draw: whether there is
 * anything to undo or redo. T1.7 fills those in from what the gesture reports.
 *
 * Sizing: this store survives the panel, which TabbedPanel unmounts on every tab
 * change (see its header comment). A draft opened in the panel must still be
 * open when the user comes back from the Eval tab, so the state lives here and
 * not in the component.
 */

import { writable, derived } from 'svelte/store';

/**
 * The draft currently open, or null when none is.
 *
 * `id` is the transcription row; `annotated` is the last `transcript.Annotated`
 * the Go side returned, verbatim.
 *
 * @type {import('svelte/store').Writable<{id: number, annotated: any} | null>}
 */
export const transcriptionStore = writable(null);

/**
 * The drafts of the open library, as `ListTranscriptions` returns them (most
 * recently updated first). The panel's list reads this; T1.2's creation form and
 * the Match panel's "draft in progress" line will read the same one.
 *
 * @type {import('svelte/store').Writable<any[]>}
 */
export const transcriptionListStore = writable([]);

/**
 * What the panel needs to enable or disable its undo/redo buttons. Written from
 * what a gesture reports (T1.7); false until then, which is what an unopened
 * draft is anyway.
 */
export const transcriptionHistoryStore = writable({ canUndo: false, canRedo: false });

/**
 * The Action the Cursor is on, or null. Derived rather than stored: the Cursor
 * is a fact of the annotated document, and a second copy of it would be one
 * more thing to keep in step after every gesture.
 */
export const transcriptionCursorStore = derived(transcriptionStore, ($t) => {
    if (!$t?.annotated) return null;
    const actions = $t.annotated.actions ?? [];
    const at = $t.annotated.cursor ?? 0;
    return actions[at] ?? null;
});

/** Puts a draft — the `TranscriptionState` a Go binding returned — in hand. */
export function setTranscription(state) {
    transcriptionStore.set(state ? { id: state.id, annotated: state.annotated } : null);
}

/**
 * Lets go of the open draft. Called when the draft is closed and when the
 * library changes — a draft belongs to the library it names two players of, and
 * a stale one would keep answering for a row id of another file.
 */
export function clearTranscription() {
    transcriptionStore.set(null);
    transcriptionHistoryStore.set({ canUndo: false, canRedo: false });
}
