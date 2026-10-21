/**
 * Les nombres de Neil Kazaross : une estimation de tête de l'équité du meneur,
 * `E ≃ 50 % + (écart de score) × Neil(n)`, où `n` est l'away du SUIVEUR.
 * Source : Kit Woolsey, « Match Equity Table », GammOnLine 1999
 * (https://bkgm.com/articles/GOL/demo/equity.htm).
 *
 * L'estimation est confrontée à la MET Kazaross-XG2 (stores/metTable) pour
 * montrer où le raccourci dévie ; elle ne vaut rien quand le meneur est à 1
 * ou 2 points, où la MET la remplace.
 */
import { tableData as metTable } from '../stores/metTable';

/** Les points d'ancrage (away du suiveur → nombre), croissants en away. @type {ReadonlyArray<[number, number]>} */
export const NEIL_ANCHORS = Object.freeze([
    [3, 10],
    [4, 9],
    [5, 8],
    [6, 7],
    [8, 6],
    [11, 5],
    [15, 4],
    [19, 3.5],
    [25, 3]
]);

/** En deçà, le raccourci ne tient pas : la MET le remplace. */
export const NEIL_MIN_LEADER_AWAY = 3;

/**
 * Le nombre de Neil pour un suiveur à `n` points, interpolé linéairement entre
 * deux ancrages, ou `null` hors de la table (3..25).
 * @param {number} n
 * @returns {number|null}
 */
export function neilNumber(n) {
    for (let i = 0; i < NEIL_ANCHORS.length; i++) {
        const [away, value] = NEIL_ANCHORS[i];
        if (n === away) return value;
        if (n < away) {
            if (i === 0) return null;
            const [prevAway, prevValue] = NEIL_ANCHORS[i - 1];
            return prevValue + ((value - prevValue) * (n - prevAway)) / (away - prevAway);
        }
    }
    return null;
}

/**
 * L'équité de match Kazaross-XG2, en %, d'un joueur à `away` contre un
 * adversaire à `oppAway`, ou `null` hors de la table (1..15).
 * @param {number} away
 * @param {number} oppAway
 * @returns {number|null}
 */
export function metEquity(away, oppAway) {
    const value = metTable[away - 1]?.[oppAway - 1];
    return Number.isFinite(value) ? value : null;
}

/**
 * L'estimation du meneur à `leaderAway` contre un suiveur à `trailerAway`.
 * `estimate` est la formule de Neil, bornée à 100 ; `fallback` dit qu'elle ne
 * s'applique pas (meneur à 1 ou 2) et que `estimate` est alors la MET. `gap`
 * est l'écart estimation − MET, `null` en repli. `null` si le meneur ne mène pas.
 *
 * @param {number} leaderAway
 * @param {number} trailerAway
 * @returns {{ estimate: number, met: number|null, gap: number|null, fallback: boolean }|null}
 */
export function neilEstimate(leaderAway, trailerAway) {
    if (!(leaderAway < trailerAway) || leaderAway < 1) return null;
    const met = metEquity(leaderAway, trailerAway);
    if (leaderAway < NEIL_MIN_LEADER_AWAY) {
        return met == null ? null : { estimate: met, met, gap: null, fallback: true };
    }
    const neil = neilNumber(trailerAway);
    if (neil == null) return null;
    const estimate = Math.min(100, 50 + (trailerAway - leaderAway) * neil);
    return { estimate, met, gap: met == null ? null : estimate - met, fallback: false };
}

/** @type {ReadonlyArray<[number, string]>} */
const FRACTIONS = Object.freeze([
    [1 / 2, '½'],
    [1 / 3, '⅓'],
    [2 / 3, '⅔'],
    [1 / 4, '¼'],
    [3 / 4, '¾']
]);

/**
 * Un nombre de Neil comme on le dit : `3½`, `5⅔`, sinon deux décimales.
 * @param {number} value
 * @returns {string}
 */
export function formatNeilNumber(value) {
    const whole = Math.floor(value + 1e-9);
    const rest = value - whole;
    if (rest < 1e-9) return String(whole);
    const glyph = FRACTIONS.find(([f]) => Math.abs(f - rest) < 1e-6)?.[1];
    return glyph ? `${whole}${glyph}` : value.toFixed(2);
}
