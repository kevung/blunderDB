/**
 * Les nombres de Neil Kazaross : une estimation de tête de l'équité du meneur,
 * `E ≃ 50 % + (écart de score) × Neil(n)`, où `n` est l'away du SUIVEUR.
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

/**
 * Le plus grand diviseur commun de deux entiers positifs.
 * @param {number} a
 * @param {number} b
 * @returns {number}
 */
function gcd(a, b) {
    return b === 0 ? a : gcd(b, a % b);
}

/**
 * Le nombre de Neil exact pour un suiveur à `n` points : entier + fraction
 * irréductible (`num`/`den`, `num` = 0 si le nombre est entier), `anchor`
 * vrai pour un point d'ancrage. `null` hors de la table (3..25).
 * @param {number} n
 * @returns {{ whole: number, num: number, den: number, anchor: boolean }|null}
 */
export function neilExact(n) {
    if (!Number.isInteger(n)) return null;
    for (let i = 0; i < NEIL_ANCHORS.length; i++) {
        const [away, value] = NEIL_ANCHORS[i];
        if (n > away) continue;
        if (n === away) return split(Math.round(value * 2), 2, true);
        if (i === 0) return null;
        const [prevAway, prevValue] = NEIL_ANCHORS[i - 1];
        // Les valeurs d'ancrage sont des demis : on calcule en demis pour rester entier.
        const span = away - prevAway;
        const num = Math.round(prevValue * 2) * span + (Math.round(value * 2) - Math.round(prevValue * 2)) * (n - prevAway);
        return split(num, 2 * span, false);
    }
    return null;
}

/**
 * @param {number} num
 * @param {number} den
 * @param {boolean} anchor
 * @returns {{ whole: number, num: number, den: number, anchor: boolean }}
 */
function split(num, den, anchor) {
    const whole = Math.floor(num / den);
    const rest = num - whole * den;
    const g = rest === 0 ? den : gcd(rest, den);
    return { whole, num: rest / g, den: rest === 0 ? 1 : den / g, anchor };
}

/** @type {Readonly<Record<string, string>>} */
const FRACTION_GLYPHS = Object.freeze({
    '1/2': '½',
    '1/3': '⅓',
    '2/3': '⅔',
    '1/4': '¼',
    '3/4': '¾',
    '1/5': '⅕',
    '2/5': '⅖',
    '3/5': '⅗',
    '4/5': '⅘',
    '1/6': '⅙',
    '5/6': '⅚',
    '1/8': '⅛',
    '3/8': '⅜',
    '5/8': '⅝',
    '7/8': '⅞'
});

/**
 * Un nombre de Neil comme on le dit : `6½`, `5⅔`, et, faute de glyphe, `3 5⁄12`.
 * @param {number} n away du suiveur
 * @returns {string} vide hors de la table
 */
export function formatNeilNumber(n) {
    const exact = neilExact(n);
    if (!exact) return '';
    if (exact.num === 0) return String(exact.whole);
    const glyph = FRACTION_GLYPHS[`${exact.num}/${exact.den}`];
    return glyph ? `${exact.whole}${glyph}` : `${exact.whole} ${exact.num}⁄${exact.den}`;
}
