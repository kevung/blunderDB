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

import { initialKeyState } from '../services/transcriptionKeys.js';

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
    resetTranscriptionKeys();
    // Le filtre par point appartient au jet en cours (T2.2) : sans brouillon
    // ouvert il n'y a plus de jet, donc plus rien à filtrer.
    transcriptionPointFilterStore.set([]);
    transcriptionCandidateStepsStore.set([]);
}

/**
 * L'état de la machine à touches (services/transcriptionKeys.js) : la phase, les
 * dés en cours de saisie, le candidat sélectionné.
 *
 * Il vit ici et pas dans le composant pour la même raison que le brouillon :
 * TabbedPanel démonte le panneau à chaque changement d'onglet, et un jet à
 * moitié tapé ne doit pas disparaître parce que l'utilisateur est allé voir
 * l'onglet Eval. Il n'est pas persisté non plus — c'est l'Entry, que le moteur
 * garde en mémoire et qu'un plantage a le droit de perdre (ADR-0045 règle 1).
 */
export const transcriptionKeyStore = writable(initialKeyState());

/** Repart d'une saisie vide : à l'ouverture d'un brouillon, et à sa fermeture. */
export function resetTranscriptionKeys() {
    transcriptionKeyStore.set(initialKeyState());
}

/**
 * Les points de départ cliqués sur le plateau, qui réduisent la liste des
 * candidats (T2.2, services/transcriptionFilter.js).
 *
 * Un état d'AFFICHAGE, et rien d'autre : il ne crée aucune Action, ne part
 * jamais au moteur et disparaît avec le jet. Il vit ici parce que deux surfaces
 * le regardent — le plateau, qui le pose au clic, et le panneau, qui montre la
 * liste réduite — et qu'un troisième chemin entre les deux aurait été un
 * chemin de plus à tenir en phase.
 *
 * @type {import('svelte/store').Writable<number[]>}
 */
export const transcriptionPointFilterStore = writable([]);

/**
 * Les candidats du jet en cours, réduits à ce que le filtre lit : leurs pas.
 *
 * Le plateau doit savoir de quels points part un coup pour décider si un clic
 * le concerne, et il n'a pas à connaître le panneau pour cela. La liste
 * complète (notations, équités, rangs) reste dans le composant : ici ne passe
 * que ce que le geste utilise.
 *
 * @type {import('svelte/store').Writable<{steps: {from: number, to: number}[]}[]>}
 */
export const transcriptionCandidateStepsStore = writable([]);

/** Plus de filtre : à chaque nouveau jet, à chaque déplacement du Cursor. */
export function resetTranscriptionPointFilter() {
    transcriptionPointFilterStore.set([]);
}
