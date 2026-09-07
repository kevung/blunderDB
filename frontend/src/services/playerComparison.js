// Deux joueurs côte à côte (#282, fiche I.26).
//
// Module pur : deux lignes de la table Joueurs entrent, une liste de lignes
// comparables sort. Aucun calcul nouveau — tout est déjà mesuré par
// `PlayerTable` —, uniquement la décision de ce qui se compare et de ce qui
// ne se compare pas.
//
// **Trois indicateurs ne reçoivent pas de verdict, et c'est le cœur du
// module.**
//
// *La chance n'est pas une qualité.* Un joueur plus chanceux n'est pas
// meilleur, et afficher une flèche verte en face de sa chance transformerait
// une mesure (ADR-0010) en compliment. Elle est montrée, sans verdict.
//
// *Un nombre de blunders ne se compare pas brut.* Douze blunders sur cent
// décisions et douze sur mille ne disent pas la même chose. Le taux, lui, se
// compare — il est calculé ici, pour cent décisions, et c'est LUI qui porte le
// verdict ; le compte reste à côté, comme contexte.
//
// *Le volume n'est pas une performance.* Matchs, bilan et décisions situent ce
// que les taux valent ; les mettre en compétition ferait gagner celui qui a
// simplement joué davantage.

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
 * Qui l'emporte sur une ligne où le plus petit gagne. Rend null dès qu'une des
 * deux valeurs manque, ET en cas d'égalité : une égalité n'est pas une
 * victoire, et la marquer d'un côté serait un tirage au sort déguisé.
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
