// The score map's arithmetic: which plays the grid holds, the lettering and the tint of a cell.
// Pure, so the view only draws.
import { pointsAway } from './awayScore.js';

/** A decision is a toss-up below this margin: shown as "≈". */
export const CLOSE_MARGIN = 0.01;
/** Above this margin the tint is the strong one. */
export const WIDE_MARGIN = 0.05;
/** Distinct plays given a colour; the others share the grey group. */
export const COLOURED_PLAYS = 3;

export const VERDICTS = ['no_double', 'double_take', 'double_pass', 'too_good'];

/** How close a cube cell's verdict is: the gap between not doubling and doubling and, when the
 * verdict is a double, the opponent's gap between taking and passing too, the smaller of the two.
 * @param {any} cell */
export function cubeMargin(cell) {
    const doubling = Math.min(cell.doubleTake, cell.doublePass);
    const toDouble = Math.abs(cell.noDouble - doubling);
    if (cell.noDouble > doubling) return toDouble;
    return Math.min(toDouble, Math.abs(cell.doubleTake - cell.doublePass));
}

/** The cell's margin over its runner-up, null when it has none (a lone legal play, a refusal).
 * @param {any} cell @param {string} decision */
export function cellMargin(cell, decision) {
    if (!cell || cell.refused) return null;
    if (decision === 'cube') return cubeMargin(cell);
    return cell.gap ?? null;
}

/** @param {number | null} margin */
export function tint(margin) {
    if (margin === null) return 'strong';
    if (margin < CLOSE_MARGIN) return 'close';
    return margin >= WIDE_MARGIN ? 'strong' : 'soft';
}

/** The key a cell groups under: its best play, or its verdict.
 * @param {any} cell @param {string} decision */
export function cellKey(cell, decision) {
    if (!cell || cell.refused) return '';
    return decision === 'cube' ? cell.verdict : (cell.move ?? '');
}

/**
 * The distinct best plays, the most frequent first, each with its letter and place in the
 * palette: the first COLOURED_PLAYS own a colour, the rest share the grey group (slot -1).
 * @param {Array<any>} cells
 * @param {string} decision
 * @returns {Array<{ key: string, letter: string, slot: number, count: number }>}
 */
export function playGroups(cells, decision) {
    const counts = new Map();
    for (const cell of cells) {
        const key = cellKey(cell, decision);
        if (key) counts.set(key, (counts.get(key) ?? 0) + 1);
    }
    const keys = [...counts.keys()].sort((a, b) => counts.get(b) - counts.get(a) || a.localeCompare(b));
    return keys.map((key, i) => ({ key, letter: i < 26 ? String.fromCharCode(65 + i) : '?', slot: i < COLOURED_PLAYS ? i : -1, count: counts.get(key) }));
}

/** Cells in the grid's order: away on roll N..1 down, away opponent N..1 across, so the 1-away
 * lines — the Crawford ones — close the grid.
 * @param {Array<any>} cells @param {number} length */
export function gridCells(cells, length) {
    /** @type {Array<Array<any>>} */
    const rows = [];
    for (let mine = length; mine >= 1; mine--) {
        const row = [];
        for (let theirs = length; theirs >= 1; theirs--) row.push(cells[(mine - 1) * length + (theirs - 1)] ?? null);
        rows.push(row);
    }
    return rows;
}

/** The away pair the position stands on, null at money. crawford says the position is the
 * Crawford game (one side 1-away, stored as 1), as a cell's own flag: a post-Crawford position
 * is never the Crawford cell.
 * @param {any} position @param {number} length */
export function positionCell(position, length) {
    const score = position?.score ?? [-1, -1];
    if (score[0] < 0 && score[1] < 0) return null;
    const onRoll = position.player_on_roll === 1 ? 1 : 0;
    const mine = pointsAway(score[onRoll]);
    const theirs = pointsAway(score[1 - onRoll]);
    if (!(mine >= 1 && theirs >= 1) || mine > length || theirs > length) return null;
    const crawford = mine === 1 ? theirs > 1 && score[onRoll] === 1 : theirs === 1 && score[1 - onRoll] === 1;
    return { mine, theirs, crawford };
}

/** The stored away score of a cell: 1-away is 1 in the Crawford game, 0 after it.
 * @param {number} away @param {boolean} crawford */
export const rawAway = (away, crawford) => (away === 1 ? (crawford ? 1 : 0) : away);

/** The position at the score of a cell, the player on roll keeping the roll.
 * @param {any} position @param {any} cell */
export function positionAtScore(position, cell) {
    const next = JSON.parse(JSON.stringify(position));
    const onRoll = position.player_on_roll === 1 ? 1 : 0;
    const crawford = !!cell.crawford;
    next.score = [0, 0];
    next.score[onRoll] = rawAway(cell.awayOnRoll, crawford && cell.awayOnRoll === 1);
    next.score[1 - onRoll] = rawAway(cell.awayOpponent, crawford && cell.awayOpponent === 1);
    next.id = 0;
    return next;
}

/** The verdicts' labels, written out so the translation catalogue can be audited against them. */
/** @type {Record<string, { short: string, long: string }>} */
export const VERDICT_LABELS = {
    no_double: { short: 'maps.verdict.short.no_double', long: 'maps.verdict.long.no_double' },
    double_take: { short: 'maps.verdict.short.double_take', long: 'maps.verdict.long.double_take' },
    double_pass: { short: 'maps.verdict.short.double_pass', long: 'maps.verdict.long.double_pass' },
    too_good: { short: 'maps.verdict.short.too_good', long: 'maps.verdict.long.too_good' }
};

/** The Eval panel's views, with the label each wears. */
export const VIEW_LABELS = { decision: 'maps.view.decision', scores: 'maps.view.scores', rolls: 'maps.view.rolls' };
