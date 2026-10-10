import { writable, get } from 'svelte/store';
import { LoadMetadata, SaveMetadata, GradeQuizChecker } from '../../wailsjs/go/database/Database.js';
import { LegalMoves } from '../../wailsjs/go/gui/App.js';
import { quizPlayStore, armBoardMove } from '../stores/quizPlayStore.js';
import { ankiAnswerShownStore, showAnkiAnswer } from '../stores/ankiStore.js';
import { newPlay, completedPlay } from './quizPlay.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { loadLibrarySettings, DEFAULT_SETTINGS } from './librarySettingsService.js';
import { logger } from '../utils/logger.js';

// « Répondre au damier » : une option par paquet qui fait jouer le coup d'une carte de pions sur
// le damier, le fait juger par le quiz (`GradeQuizChecker`) et en tire une NOTE PROPOSÉE. Le
// joueur garde la main : il valide la suggestion ou la corrige avant de noter. L'auto-notation
// reste le défaut (ADR-0040) ; les cartes de videau et de score ne se jouent pas au damier.
//
// L'option vit dans les MÉTADONNÉES de la base, pas dans une colonne du paquet : elle suit la
// bibliothèque sans schéma nouveau (comme l'objectif de progression).

const KEY_PREFIX = 'anki_board_answer_';

/** Au-dessous de ce temps, une bonne réponse mérite « Facile » plutôt que « Bien ». */
export const FAST_ANSWER_MS = 10000;

/** @type {Map<number, boolean>} */
const enabled = new Map();

// Les identifiants de paquet se recoupent d'une base à l'autre : l'option lue dans l'une ne doit
// pas répondre pour l'autre.
databasePathStore.subscribe(() => enabled.clear());

/**
 * Le paquet répond-il au damier ?
 * @param {number} deckId
 */
export async function boardAnswerEnabled(deckId) {
    if (enabled.has(deckId)) return /** @type {boolean} */ (enabled.get(deckId));
    let on = false;
    try {
        const meta = (await LoadMetadata()) || {};
        on = meta[KEY_PREFIX + deckId] === '1';
    } catch (err) {
        logger.error('could not read the board-answer option:', err);
    }
    enabled.set(deckId, on);
    return on;
}

/**
 * Écrit l'option en relisant les autres métadonnées : la table est un dictionnaire unique, qu'une
 * écriture partielle effacerait.
 * @param {number} deckId @param {boolean} on
 */
export async function setBoardAnswer(deckId, on) {
    const meta = (await LoadMetadata()) || {};
    // `SaveMetadata` ne fait que remplacer : une clé retirée du dictionnaire reviendrait au
    // relancement, d'où un « 0 » écrit.
    meta[KEY_PREFIX + deckId] = on ? '1' : '0';
    await SaveMetadata(meta);
    enabled.set(deckId, on);
}

/**
 * La carte est-elle une décision de pions qui se joue au damier ? Les cartes de score n'ont pas de
 * position ; une décision de videau n'a pas de coup à jouer.
 * @param {any} card
 */
export function isBoardPlayable(card) {
    const position = card?.position;
    if (!position?.dice) return false;
    return position.decision_type !== 1 && position.dice[0] > 0 && position.dice[1] > 0;
}

/**
 * La note que le verdict suggère : bonne réponse → 4 si elle est rapide, 3 sinon ; erreur sous le
 * seuil du blunder → 2 ; blunder ou coup illégal → 1. Un coup légal que l'analyse ne classe pas
 * n'a pas de coût connu : aucune suggestion.
 * @param {{legal: boolean, matched: boolean, errorMp: number}} verdict
 * @param {number} elapsedMs
 * @param {number} [blunderMp] le seuil du blunder de la bibliothèque (Réglages)
 * @returns {1|2|3|4|null}
 */
export function suggestRating(verdict, elapsedMs, blunderMp = DEFAULT_SETTINGS.blunderThresholdMP) {
    if (!verdict.legal) return 1;
    if (!verdict.matched) return null;
    if (verdict.errorMp === 0) return elapsedMs <= FAST_ANSWER_MS ? 4 : 3;
    return verdict.errorMp < blunderMp ? 2 : 1;
}

/**
 * L'état de la réponse au damier : `play` pendant que le joueur joue, `graded` une fois jugée.
 * Nul hors d'une carte qui se joue au damier.
 * @type {import('svelte/store').Writable<null | {phase: 'play'} | {phase: 'graded', verdict: any, suggested: number|null, elapsedMs: number}>}
 */
export const ankiBoardAnswerStore = writable(null);

let armedCardId = /** @type {number|null} */ (null);
let armedAt = 0;
// Le damier de quiz est partagé avec l'Entraînement : on ne rend que ce qu'on a posé.
let ownsBoard = false;

/**
 * Arme le damier pour la carte, si le paquet le demande. Idempotent pour une même carte : la
 * relecture des statistiques du paquet ne doit ni rejouer le coup ni remettre le chrono à zéro.
 * @param {any} deck @param {any} card
 */
export async function armBoardAnswer(deck, card) {
    const id = card?.card?.id ?? null;
    if (id === null || id === armedCardId) return;
    disarmBoardAnswer();
    armedCardId = id;
    if (get(ankiAnswerShownStore) || deck?.sourceType === 'scores' || !isBoardPlayable(card)) return;
    if (!(await boardAnswerEnabled(deck.id))) return;
    let plays;
    try {
        plays = await LegalMoves(card.position);
    } catch (err) {
        logger.error('could not load the legal moves for a board answer:', err);
        return;
    }
    // La carte a pu changer pendant l'aller-retour.
    if (armedCardId !== id || !plays?.length) return;
    armBoardMove(newPlay(card.position, plays), () => void validateBoardAnswer(card));
    ownsBoard = true;
    armedAt = Date.now();
    ankiBoardAnswerStore.set({ phase: 'play' });
}

/** Rend le damier sans oublier le verdict : la réponse est montrée, la suggestion reste lisible. */
export function releaseBoard() {
    if (ownsBoard) quizPlayStore.set(null);
    ownsBoard = false;
    // Plus rien à jouer : seul un verdict déjà rendu reste lisible.
    if (get(ankiBoardAnswerStore)?.phase === 'play') ankiBoardAnswerStore.set(null);
}

/** Quitte la carte : le damier est rendu, la suggestion oubliée. */
export function disarmBoardAnswer() {
    releaseBoard();
    armedCardId = null;
    ankiBoardAnswerStore.set(null);
}

/**
 * Juge le coup joué, montre la réponse et propose une note. Sans coup complet, rien n'est jugé.
 * Le chrono s'arrête au geste, pas au retour du juge.
 * @param {any} card
 */
export async function validateBoardAnswer(card) {
    const state = get(quizPlayStore);
    const play = ownsBoard && state ? completedPlay(state) : null;
    if (!play) return;
    const elapsedMs = Date.now() - armedAt;
    const id = card.card.id;
    let verdict;
    try {
        verdict = await GradeQuizChecker(card.position.id, play.result.board);
    } catch (err) {
        logger.error('could not grade the board answer:', err);
        return;
    }
    const { blunderThresholdMP } = await loadLibrarySettings();
    if (armedCardId !== id) return;
    ankiBoardAnswerStore.set({ phase: 'graded', verdict, suggested: suggestRating(verdict, elapsedMs, blunderThresholdMP), elapsedMs });
    releaseBoard();
    showAnkiAnswer();
}
