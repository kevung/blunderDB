// Deux joueurs côte à côte : module pur, deux lignes de la table Joueurs
// entrent, des lignes comparables sortent. Aucun calcul nouveau ; seule la
// décision de ce qui reçoit un verdict.
//
// Trois indicateurs n'en reçoivent pas :
// - la chance n'est pas une qualité (ADR-0010) : montrée, sans verdict ;
// - un nombre de blunders ne se compare pas brut : le taux pour cent
//   décisions porte le verdict, le compte reste en contexte ;
// - le volume (matchs, bilan, décisions) situe les taux sans être une
//   performance.

/** Une ligne sans verdict : elle situe, elle ne départage pas. */
const CONTEXT = /** @type {const} */ ('context');
/** Une ligne où le plus petit gagne (une erreur moyenne). */
const LOWER = /** @type {const} */ ('lower');

/**
 * @typedef {{
 *   key: string, labelKey: string, kind: 'context'|'lower',
 *   a: string, b: string, better: 'a'|'b'|null
 * }} ComparisonLine
 */

/**
 * Un taux n'a de sens que mesuré ; sinon il n'est pas comparable.
 * @param {any} row
 * @param {string} key
 * @param {string} denominator
 * @returns {number|null}
 */
function rate(row, key, denominator) {
    if (!row || (row[denominator] ?? 0) <= 0) return null;
    const v = row[key];
    return typeof v === 'number' && !isNaN(v) ? v : null;
}

/**
 * Blunders pour cent décisions comptées, ou null si rien n'est compté.
 * @param {any} row
 * @returns {number|null}
 */
export function blunderRate(row) {
    if (!row || (row.decisions ?? 0) <= 0) return null;
    return (100 * (row.blunders ?? 0)) / row.decisions;
}

/**
 * @param {number|null} value
 * @param {number} [digits]
 */
function fmt(value, digits = 2) {
    return value === null ? '—' : value.toFixed(digits);
}

/**
 * Qui l'emporte quand le plus petit gagne ; null si une valeur manque ou en
 * cas d'égalité.
 * @param {number|null} va
 * @param {number|null} vb
 * @returns {'a'|'b'|null}
 */
function betterLower(va, vb) {
    if (va === null || vb === null || va === vb) return null;
    return va < vb ? 'a' : 'b';
}

/**
 * Les lignes de la comparaison, dans l'ordre où elles se lisent.
 *
 * @param {any} a
 * @param {any} b
 * @returns {ComparisonLine[]}
 */
export function compareRows(a, b) {
    /** @type {ComparisonLine[]} */
    const lines = [];

    /** @type {(key: string, labelKey: string, va: string, vb: string) => void} */
    const context = (key, labelKey, va, vb) => {
        lines.push({ key, labelKey, kind: CONTEXT, a: va, b: vb, better: null });
    };
    /** @type {(key: string, labelKey: string, va: number|null, vb: number|null, digits?: number) => void} */
    const lower = (key, labelKey, va, vb, digits = 2) => {
        lines.push({ key, labelKey, kind: LOWER, a: fmt(va, digits), b: fmt(vb, digits), better: betterLower(va, vb) });
    };

    context('matches', 'stats.playersColMatches', String(a?.matches ?? 0), String(b?.matches ?? 0));
    context('record', 'stats.playersColRecord', `${a?.wins ?? 0}–${a?.losses ?? 0}`, `${b?.wins ?? 0}–${b?.losses ?? 0}`);
    context('decisions', 'stats.playersColDecisions', String(a?.decisions ?? 0), String(b?.decisions ?? 0));

    lower('pr', 'stats.playersColPR', rate(a, 'pr', 'decisions'), rate(b, 'pr', 'decisions'));
    lower('pr_checker', 'stats.playersColPRChecker', rate(a, 'pr_checker', 'checker_decisions'), rate(b, 'pr_checker', 'checker_decisions'));
    lower('pr_cube', 'stats.playersColPRCube', rate(a, 'pr_cube', 'cube_decisions'), rate(b, 'pr_cube', 'cube_decisions'));
    lower('snowie_er', 'stats.playersColSnowie', rate(a, 'snowie_er', 'decisions'), rate(b, 'snowie_er', 'decisions'));
    lower('blunder_rate', 'stats.compareBlunderRate', blunderRate(a), blunderRate(b), 1);

    context('blunders', 'stats.playersColBlunders', String(a?.blunders ?? 0), String(b?.blunders ?? 0));
    context('luck', 'stats.playersColLuck', luck(a), luck(b));

    return lines;
}

/**
 * La chance, signée, par lancer mesuré — ou rien du tout.
 * @param {any} row
 */
function luck(row) {
    if (!row?.luck_known) return '—';
    const v = row.luck_rate_mp;
    return (v > 0 ? '+' : v < 0 ? '−' : '') + Math.abs(v).toFixed(1);
}
