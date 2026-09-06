/**
 * La fiche de score (ADR-0040 règle 4).
 *
 * Une question de l'exercice Scores est UN score non ordonné. La fiche en
 * porte les deux faces — « vous » et « l'adversaire » — parce qu'une décision
 * de videau au score a besoin des deux : le point de prise corrigé combine les
 * valeurs de gammon des deux joueurs, et c'est le point de prise de
 * l'adversaire qui dit si le double passe.
 *
 * Chaque face ne rend que les cases que les tables définissent pour elle. Pas
 * de case « sans objet » à deviner : savoir que gv4 n'a pas d'objet à 3 away
 * est une convention, pas un calcul. D'où trois nombres à 2a-2a et quatorze au
 * plus, une seule colonne à score égal, et une ligne qu'aucune face ne définit
 * qui ne figure tout simplement pas.
 *
 * Ce module ne connaît ni Svelte, ni le plateau, ni Wails : il rend une
 * géométrie et des nombres, et c'est ce qui le rend vérifiable.
 */
import { REFERENCE_TABLES, referenceValue } from './referenceTables.js';

/** Les sept lignes de la fiche, dans l'ordre de la règle 4. */
export const SCORE_CARD_ROWS = Object.freeze(['tp2.live', 'tp2.last', 'tp4.live', 'tp4.last', 'gv1', 'gv2', 'gv4']);

/** Les away que le vivier couvre. */
export const SCORE_AWAY_MIN = 2;
export const SCORE_AWAY_MAX = 9;

/**
 * Les 36 scores non ordonnés de 2 à 9 away, `[petit, grand]`. Non ordonnés :
 * 3:5 et 5:3 sont le même score, et la fiche le montre en échangeant ses deux
 * colonnes — ce n'est pas une seconde question.
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
 * Construit la fiche d'un score. À score égal il n'y a qu'une colonne : les
 * deux faces liraient les mêmes cases, et les montrer deux fois serait une
 * redite, pas une comparaison.
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
 * Les nombres de la fiche, à plat et dans l'ordre de lecture (ligne par ligne,
 * face par face). C'est cette liste que la session parcourt : un nombre est
 * l'unité de faute, jamais la question entière ni la ligne.
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
