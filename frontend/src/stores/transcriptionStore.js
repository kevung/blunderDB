/**
 * transcriptionStore.js — the open transcription draft, as the panel sees it.
 *
 * The panel is a CLIENT of the Go engine (ADR-0045 rule 9): every gesture comes back as a whole
 * annotated document, held here verbatim so panel, board and status bar read one value.
 * The undo stack lives once, in Go (`transcript.Editor`, beside the unserialisable Entry); only
 * `can_undo`/`can_redo` are mirrored here. State lives in stores because TabbedPanel unmounts
 * the panel on every tab change.
 */

import { writable, derived } from 'svelte/store';

import { initialKeyState } from '../services/transcriptionKeys.js';

/**
 * The draft currently open, or null: `id` is the transcription row, `annotated` the last
 * `transcript.Annotated` returned, verbatim.
 *
 * @type {import('svelte/store').Writable<{id: number, annotated: any} | null>}
 */
export const transcriptionStore = writable(null);

/**
 * The drafts of the open library, as `ListTranscriptions` returns them (most recent first).
 *
 * @type {import('svelte/store').Writable<any[]>}
 */
export const transcriptionListStore = writable([]);

/** Undo/redo availability, from each gesture; false after a restart — the stack is in memory only (ADR-0045 rule 1). */
export const transcriptionHistoryStore = writable({ canUndo: false, canRedo: false });

/**
 * The undo or redo the global dispatcher asked for, `null` once served. Ctrl combos are always
 * global (`isAlwaysGlobal`), so the dispatcher posts here and the panel, which owns the round
 * trip, performs it — like `ankiReviewActionStore`.
 *
 * @type {import('svelte/store').Writable<'undo'|'redo'|null>}
 */
export const transcriptionHistoryActionStore = writable(null);

/** The Action the Cursor is on, or null — derived, the Cursor being a fact of the document. */
export const transcriptionCursorStore = derived(transcriptionStore, ($t) => {
    if (!$t?.annotated) return null;
    const actions = $t.annotated.actions ?? [];
    const at = $t.annotated.cursor ?? 0;
    return actions[at] ?? null;
});

/** @param {any} state a `TranscriptionState` returned by a Go binding */
export function setTranscription(state) {
    transcriptionStore.set(state ? { id: state.id, annotated: state.annotated } : null);
    // Both sides of the stack come back with every gesture: they are read off
    // the live transcript.Editor on the Go side, which is the only place the
    // stack exists (ADR-0045 rule 1 — a crash loses it, and that is the promise).
    transcriptionHistoryStore.set({ canUndo: state?.can_undo === true, canRedo: state?.can_redo === true });
}

/**
 * Ce que la barre de match (`MatchInfoBar`, ux.md §5) dit du brouillon ouvert : longueur, score,
 * Crawford, partie, videau, trait ; `null` sans brouillon. Le panneau les pose, la barre les lit
 * (ADR-0048 décision 2).
 *
 * @type {import('svelte/store').Writable<null | {lengthKey: string, lengthParams: Record<string, any>, score: number[] | null, crawford: boolean, gameNumber: number, cubeKey: string, cubeParams: Record<string, any>, onRoll: string, player1?: string, player2?: string}>}
 */
export const transcriptionInfoStore = writable(null);

/**
 * L'Action attendue pour la barre d'état (ux.md §5, ADR-0048 décision 2) : une clé i18n et ses
 * paramètres, ou `null`. Un état, toujours présent tant qu'un brouillon est ouvert.
 *
 * @type {import('svelte/store').Writable<null | {key: string, params?: Record<string, any>}>}
 */
export const transcriptionPromptStore = writable(null);

/**
 * La réponse TRANSITOIRE d'un geste sans effet (ADR-0048 décision 9) — « rien à annuler »,
 * « aucune Action sous le curseur » —, remplacée par l'Action attendue après [NOTICE_MS].
 * Sans elle, `Ctrl+Z` sur une pile vide (vide par construction sur un brouillon réouvert),
 * `x`/`Suppr`/`s` en bout de document ou `Retour arrière` sans dé se taisaient comme un bug.
 *
 * @type {import('svelte/store').Writable<null | {key: string, params?: Record<string, any>}>}
 */
export const transcriptionNoticeStore = writable(null);

/**
 * Un cran de molette au-dessus du plateau (ADR-0048 décision 11), posé par `App.svelte` et servi
 * par le panneau, qui possède la sélection. `at` distingue deux crans égaux successifs, qu'un
 * magasin dédoublonnerait.
 *
 * @type {import('svelte/store').Writable<null | {delta: number, at: number}>}
 */
export const transcriptionWheelStore = writable(null);

/** Combien de temps une réponse transitoire reste à l'écran. */
export const NOTICE_MS = 1500;

/** @type {ReturnType<typeof setTimeout> | undefined} */
let noticeTimer = undefined;

/**
 * Pose une réponse transitoire, en remplaçant la précédente.
 * @param {string} key
 * @param {object} [params]
 */
export function noticeTranscription(key, params = undefined) {
    clearTimeout(noticeTimer);
    transcriptionNoticeStore.set({ key, params });
    noticeTimer = setTimeout(() => transcriptionNoticeStore.set(null), NOTICE_MS);
}

/** Efface la réponse transitoire et son minuteur (démontage, fermeture). */
export function clearTranscriptionNotice() {
    clearTimeout(noticeTimer);
    noticeTimer = undefined;
    transcriptionNoticeStore.set(null);
}

/**
 * Lets go of the open draft, also on library change: a stale draft would answer for a row id of
 * another file.
 */
export function clearTranscription() {
    transcriptionStore.set(null);
    transcriptionHistoryStore.set({ canUndo: false, canRedo: false });
    // La barre de match et la barre d'état parlent du brouillon OUVERT : sans
    // brouillon elles n'ont plus rien à dire (ADR-0048 décisions 2 et 9).
    transcriptionInfoStore.set(null);
    transcriptionPromptStore.set(null);
    transcriptionWheelStore.set(null);
    clearTranscriptionNotice();
    transcriptionHistoryActionStore.set(null);
    resetTranscriptionKeys();
    // Un clic sur le videau resté sans réponse ne doit pas servir le brouillon
    // suivant (T2.5).
    transcriptionCubeRequestStore.set(null);
    // Le sens du plateau appartient au brouillon : « le joueur 1 » change d'un brouillon à l'autre.
    transcriptionBoardSwapStore.set(false);
}

/**
 * L'état de la machine à touches (services/transcriptionKeys.js) : phase, dés en saisie,
 * candidat sélectionné. Ici pour survivre au démontage du panneau ; non persisté, comme l'Entry
 * (ADR-0045 règle 1).
 */
export const transcriptionKeyStore = writable(initialKeyState());

/** Repart d'une saisie vide : à l'ouverture d'un brouillon, et à sa fermeture. */
export function resetTranscriptionKeys() {
    transcriptionKeyStore.set(initialKeyState());
}

/**
 * Le videau cliqué sur le plateau, `null` une fois servi. Le plateau pose la demande sans juger ;
 * le panneau, qui tient le brouillon, en fait un double ou l'ignore.
 *
 * @type {import('svelte/store').Writable<'double'|null>}
 */
export const transcriptionCubeRequestStore = writable(null);

/**
 * Le plateau du brouillon est-il montré RETOURNÉ, joueur 2 en bas ? Le panneau montre le joueur 1
 * en bas, comme le mode Match : suivre le camp au trait ferait basculer le damier à chaque
 * demi-coup. Préférence d'affichage seulement — à ne pas confondre avec `swap_players`
 * (TranscriptionMetadata.svelte), qui échange les joueurs dans le document.
 *
 * @type {import('svelte/store').Writable<boolean>}
 */
export const transcriptionBoardSwapStore = writable(false);
