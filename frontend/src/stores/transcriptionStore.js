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
 * anything to undo or redo, filled in from what every gesture reports
 * (`can_undo`/`can_redo` on the TranscriptionState).
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
 * what a gesture reports; false for an unopened draft, and false again after a
 * restart — the stack is in memory and nowhere else (ADR-0045 rule 1).
 */
export const transcriptionHistoryStore = writable({ canUndo: false, canRedo: false });

/**
 * The undo or redo the global dispatcher asked for, `null` when it has been
 * served. `Ctrl+Z` is a Ctrl combo and Ctrl combos are always global
 * (keyboardService.js's `isAlwaysGlobal`, which exists so that a shortcut added
 * to one list does not die behind another): the dispatcher therefore posts the
 * gesture here and the panel, which owns the draft and the round trip, performs
 * it. Same shape as `ankiReviewActionStore`, and for the same reason.
 *
 * @type {import('svelte/store').Writable<'undo'|'redo'|null>}
 */
export const transcriptionHistoryActionStore = writable(null);

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
    // Both sides of the stack come back with every gesture: they are read off
    // the live transcript.Editor on the Go side, which is the only place the
    // stack exists (ADR-0045 rule 1 — a crash loses it, and that is the promise).
    transcriptionHistoryStore.set({ canUndo: state?.can_undo === true, canRedo: state?.can_redo === true });
}

/**
 * Ce que la BARRE DE MATCH dit du brouillon ouvert : longueur, score, Crawford,
 * numéro de partie, videau, camp au trait. `null` quand aucun brouillon ne l'est.
 *
 * Ces six faits étaient sept pastilles dans le panneau (ADR-0048 décision 2), ce
 * que `ux.md` §5 n'avait jamais demandé : il les plaçait dans `MatchInfoBar`,
 * qui est déjà au-dessus du plateau — donc lue au moment où l'œil est sur le
 * plateau, qui est le bon moment pour « Kévin au trait, videau à 2 ». Le panneau
 * les POSE ici et ne les dessine plus ; la barre les lit.
 *
 * @type {import('svelte/store').Writable<null | {lengthKey: string, lengthParams: object, score: number[], crawford: boolean, gameNumber: number, cubeKey: string, cubeParams: object, onRoll: string}>}
 */
export const transcriptionInfoStore = writable(null);

/**
 * L'Action attendue, en un mot, pour la BARRE D'ÉTAT (`ux.md` §5, jamais câblé
 * avant ADR-0048 décision 2). Une clé i18n et ses paramètres, ou `null`.
 *
 * C'est un ÉTAT : il y en a toujours exactement un tant qu'un brouillon est
 * ouvert, et il ne s'efface pas tout seul.
 *
 * @type {import('svelte/store').Writable<null | {key: string, params?: object}>}
 */
export const transcriptionPromptStore = writable(null);

/**
 * La réponse TRANSITOIRE d'un geste sans effet (ADR-0048 décision 9) : « rien à
 * annuler », « aucune Action sous le curseur ». Elle cède la place à la phrase
 * de l'Action attendue au bout de [NOTICE_MS].
 *
 * Pourquoi elle existe. Quatre gestes n'ont rien à faire dans un état
 * parfaitement ordinaire et se taisaient tous les quatre : `Ctrl+Z` sur une pile
 * vide — vide PAR CONSTRUCTION sur un brouillon réouvert, la pile vivant dans le
 * `transcript.Editor` de la session (ADR-0045 règle 1) —, `x`/`Suppr`/`s` en
 * bout de document, `Retour arrière` sans dé saisi. Le seul de la famille qui
 * disait quelque chose était un bouton grisé, et c'est celui que la décision 3
 * supprime. Une promesse prise exprès qui ne se dit jamais est indiscernable
 * d'un bug.
 *
 * @type {import('svelte/store').Writable<null | {key: string, params?: object}>}
 */
export const transcriptionNoticeStore = writable(null);

/**
 * Un cran de molette donné AU-DESSUS DU PLATEAU (ADR-0048 décision 11).
 *
 * Le plateau ne connaît pas la liste des candidats, et le panneau ne reçoit pas
 * les événements du plateau : le dispatcher de `App.svelte` pose donc le cran
 * ici et le panneau, qui possède la sélection, le sert. Même forme que
 * `transcriptionCubeRequestStore` et que `Ctrl+Z`, pour la même raison.
 *
 * `at` distingue deux crans identiques qui se suivent : un magasin dédoublonne
 * les valeurs égales, et deux `{delta: 1}` de suite n'en feraient qu'un.
 *
 * @type {import('svelte/store').Writable<null | {delta: number, at: number}>}
 */
export const transcriptionWheelStore = writable(null);

/** Combien de temps une réponse transitoire reste à l'écran. */
export const NOTICE_MS = 1500;

let noticeTimer = null;

/** Pose une réponse transitoire, en remplaçant celle qui traînait. */
export function noticeTranscription(key, params = undefined) {
    clearTimeout(noticeTimer);
    transcriptionNoticeStore.set({ key, params });
    noticeTimer = setTimeout(() => transcriptionNoticeStore.set(null), NOTICE_MS);
}

/** Efface la réponse transitoire et son minuteur (démontage, fermeture). */
export function clearTranscriptionNotice() {
    clearTimeout(noticeTimer);
    noticeTimer = null;
    transcriptionNoticeStore.set(null);
}

/**
 * Lets go of the open draft. Called when the draft is closed and when the
 * library changes — a draft belongs to the library it names two players of, and
 * a stale one would keep answering for a row id of another file.
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
    // Le filtre par point appartient au jet en cours (T2.2) : sans brouillon
    // ouvert il n'y a plus de jet, donc plus rien à filtrer.
    transcriptionPointFilterStore.set([]);
    transcriptionCandidateStepsStore.set([]);
    // Un clic sur le videau resté sans réponse ne doit pas servir le brouillon
    // suivant (T2.5).
    transcriptionCubeRequestStore.set(null);
    // Le sens du plateau appartient au brouillon regardé, pas à la session :
    // « le joueur 1 » n'est pas la même personne d'un brouillon à l'autre, et
    // une inversion retenue montrerait le suivant à l'envers sans qu'on l'ait
    // demandé.
    transcriptionBoardSwapStore.set(false);
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

/**
 * Le videau cliqué sur le plateau (T2.5), `null` quand la demande a été servie.
 *
 * Le plateau POSE la demande, le panneau la sert : c'est le chemin de
 * `transcriptionHistoryActionStore` pour `Ctrl+Z`, et il est ici pour la même
 * raison — le geste naît sur une surface qui ne tient ni le brouillon ni
 * l'aller-retour Wails. Le plateau ne juge donc rien : il dit « le videau a été
 * cliqué », et c'est le panneau, qui sait ce que le document attend, qui en
 * fait un double ou qui laisse tomber la demande.
 *
 * @type {import('svelte/store').Writable<'double'|null>}
 */
export const transcriptionCubeRequestStore = writable(null);

/**
 * Le plateau du brouillon est-il montré RETOURNÉ, joueur 2 en bas ?
 *
 * Pourquoi il existe. Hors transcription, le plateau montre toujours le camp au
 * trait en bas : une position de la bibliothèque est enregistrée normalisée, le
 * camp au trait EST le joueur 0, et rien n'oscille. Un brouillon, lui, est une
 * partie qui se déroule : le trait change à chaque demi-coup, et la même règle y
 * faisait basculer le damier d'un tour sur l'autre — les pions de celui qu'on
 * vient de regarder passaient en haut, ceux d'en face descendaient, et l'œil
 * refaisait le trajet à chaque jet. Le panneau montre donc le JOUEUR 1 en bas,
 * comme le fait déjà le mode Match, et le trait se lit aux dés, qui changent de
 * côté.
 *
 * Ce que ce magasin n'est PAS : le geste `swap_players` de l'en-tête
 * (TranscriptionMetadata.svelte), qui échange les deux joueurs DANS le document
 * — les noms, les camps, les Actions. Ici rien n'est modifié : c'est une
 * préférence d'affichage, et le brouillon enregistré est le même dans les deux
 * sens.
 *
 * @type {import('svelte/store').Writable<boolean>}
 */
export const transcriptionBoardSwapStore = writable(false);
