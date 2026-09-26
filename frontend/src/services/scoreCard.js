/**
 * La fiche de score (ADR-0040 règle 4) : un score non ordonné, vu des deux
 * faces, car le point de prise corrigé combine les valeurs de gammon des deux
 * joueurs. Chaque face ne rend que les cases que les tables définissent pour
 * elle (gv4 n'existe pas à 3 away) : de 3 à 14 nombres, une seule colonne à
 * score égal. Sans Svelte, plateau ni Wails.
 */
import { REFERENCE_TABLES, referenceValue } from './referenceTables.js';

/** Les sept lignes de la fiche, dans l'ordre de la règle 4. */
export const SCORE_CARD_ROWS = Object.freeze(['tp2.live', 'tp2.last', 'tp4.live', 'tp4.last', 'gv1', 'gv2', 'gv4']);

/** Les away que le vivier couvre. */
export const SCORE_AWAY_MIN = 2;
export const SCORE_AWAY_MAX = 9;

/**
 * Les 36 scores non ordonnés de 2 à 9 away, `[petit, grand]` : 3:5 et 5:3 sont
 * la même question, colonnes échangées.
 * @type {readonly [number, number][]}
 */
export const UNORDERED_SCORES = Object.freeze(
    (() => {
        /** @type {[number, number][]} */
        const scores = [];
        for (let a = SCORE_AWAY_MIN; a <= SCORE_AWAY_MAX; a++) {
            for (let b = a; b <= SCORE_AWAY_MAX; b++) scores.push([a, b]);
        }
        return scores;
    })()
);

/**
 * @typedef {object} ScoreCardFace
 * @property {'you'|'opponent'} face
 * @property {number} away l'away de cette face
 * @property {number} opponentAway l'away de l'autre
 *
 * @typedef {object} ScoreCardCell
 * @property {string} type une des SCORE_CARD_ROWS
 * @property {'you'|'opponent'} face
 * @property {number} away
 * @property {number} value
 * @property {number} precision les décimales à afficher
 *
 * @typedef {object} ScoreCard
 * @property {number} awayYou
 * @property {number} awayOpponent
 * @property {boolean} level le score est-il égal (une seule colonne) ?
 * @property {ScoreCardFace[]} faces
 * @property {{type: string, cells: (ScoreCardCell|null)[]}[]} rows
 */

/**
 * Construit la fiche d'un score ; une seule colonne à score égal.
 *
 * @param {number} awayYou @param {number} awayOpponent
 * @returns {ScoreCard}
 */
export function buildScoreCard(awayYou, awayOpponent) {
    const level = awayYou === awayOpponent;
    /** @type {ScoreCardFace[]} */
    const faces = level
        ? [{ face: 'you', away: awayYou, opponentAway: awayOpponent }]
        : [
              { face: 'you', away: awayYou, opponentAway: awayOpponent },
              { face: 'opponent', away: awayOpponent, opponentAway: awayYou }
          ];

    const rows = [];
    for (const type of SCORE_CARD_ROWS) {
        const cells = faces.map((f) => {
            const value = referenceValue(type, f.away, f.opponentAway);
            if (value === null) return null;
            return { type, face: f.face, away: f.away, value, precision: REFERENCE_TABLES[type].precision };
        });
        if (cells.some((cell) => cell !== null)) rows.push({ type, cells });
    }
    return { awayYou, awayOpponent, level, faces, rows };
}

/**
 * Les nombres de la fiche, à plat, ligne par ligne et face par face : le
 * nombre est l'unité de faute.
 * @param {ScoreCard} card
 * @returns {ScoreCardCell[]}
 */
export function scoreCardNumbers(card) {
    const out = [];
    for (const row of card.rows) {
        for (const cell of row.cells) if (cell) out.push(cell);
    }
    return out;
}
