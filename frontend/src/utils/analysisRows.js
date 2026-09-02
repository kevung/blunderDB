// The formatted rows of an analysis, built once for every surface that shows
// them.
//
// Two DOM tables (CubeVerdictTable, CandidateMovesTable) and the canvas that
// clipboardService paints for "copy board with analysis" each used to carry
// their own formatEquity and their own .toFixed(2) — four copies of one rule,
// and the copied image drifted from the screen: a centred cube was captioned
// "Redouble" on the image, an absent error cell was painted +0.000, the played
// action was matched with a narrower vocabulary than the panel's. This module
// is the one place a number becomes a cell. The tables and the canvas consume
// rows that are already strings, and only lay them out — which is also what
// keeps ADR-0019 honest on the frontend: the figures leave here exactly as the
// backend supplied them, formatted, never converted.
//
// Pure on purpose: no store, no DOM, no i18n import. The translation function
// is a parameter, so the canvas (non-reactive `translate`) and a component
// (reactive `$t`) get identical labels from identical code.

import { CUBE_OPTIONS, DECISION_STATE } from './cubeDecision.js';
import { normalizeCubeAction } from './cubeAction.js';

// An absent fact — PositionFactsTable's own mark, so a value that does not
// exist reads the same way in every table of the panel.
export const DASH = '—';
// Défi's placeholder (ADR-0020 rule 7): the value is replaced in place.
export const HIDDEN = '···';

// The single equity rule: three decimals, an explicit sign. Absent → null, so
// the caller decides what an absence means on its surface (see below).
export function formatEquity(value) {
    if (value == null || Number.isNaN(value)) return null;
    return (value >= 0 ? '+' : '') + value.toFixed(3);
}

// The single probability rule: two decimals, on the scale the value arrives in
// (the backend's percentages — no ×100 here, ADR-0019).
export function formatChance(value) {
    if (value == null || Number.isNaN(value)) return null;
    return value.toFixed(2);
}

// ---------------------------------------------------------------------------
// Cube decision block (ADR-0020)
// ---------------------------------------------------------------------------

// cubeValue is the log2 exponent everywhere in blunderDB (see the XGID
// contract), so >= 1 means the cube has already been turned at least once:
// the options are redoubles.
const OPTION_LABEL_KEYS = {
    no_double: ['analysis.noDouble', 'analysis.noRedouble'],
    double_take: ['analysis.doubleTake', 'analysis.redoubleTake'],
    double_pass: ['analysis.doublePass', 'analysis.redoublePass']
};

// Legacy action strings: AnalysisPanel's isPlayedCubeAction speaks the
// vocabulary stored in playedCubeAction ("Double", "Take", …), so the
// canonical keys are translated back at the call rather than changing a
// contract two panels and the board already share.
export function isPlayedOption(key, isPlayedCubeAction) {
    if (key === 'no_double') return isPlayedCubeAction('No Double');
    if (key === 'double_take') return isPlayedCubeAction('Double') && isPlayedCubeAction('Take');
    return isPlayedCubeAction('Double') && isPlayedCubeAction('Pass');
}

// The single place the block's state is named (ADR-0020 rule 4). Empty means
// "still computing" and nothing else — never a refusal, never a regime that
// will never answer, never a dead cube.
function verdictText(decision, state, t) {
    switch (state) {
        case DECISION_STATE.PENDING:
            return '';
        case DECISION_STATE.NO_DECISION:
            return t('cube.noDecision');
        case DECISION_STATE.REFUSED:
            return t('cube.refused');
        case DECISION_STATE.CUBE_OPPONENT:
            return t('cube.cubeOpponent');
        case DECISION_STATE.CRAWFORD:
            return t('cube.crawford');
        default:
            // A live evaluation names its verdict by key, so it is translated
            // and keeps "too good"; a stored record carries its analysing
            // engine's own words and is reported verbatim.
            return decision?.verdict ? t('cube.verdicts.' + decision.verdict) : (decision?.verdictText ?? '');
    }
}

/**
 * cubeRows lays out a cube Decision (utils/cubeDecision.js) as the three
 * named option rows plus the verdict row.
 *
 * An absent equity or error is an EMPTY cell here, not a dash: in this block
 * an empty cell is a state — pending (rule 4), the best option's "nothing to
 * lose" (rule 2), or "no choice to make" with a dead cube (rule 5).
 *
 * @returns {{ header: string[], rows: {key, label, cells: string[], highlight: boolean, best: boolean}[], verdict: {label, text, unavailable} }}
 */
export function cubeRows(decision, { t, cubeValue = 0, isPlayedCubeAction = () => false, masked = false } = {}) {
    const options = decision?.options ?? CUBE_OPTIONS.map((key) => ({ key, equity: null, error: null }));
    const state = decision?.state ?? DECISION_STATE.PENDING;
    // The best-row emphasis is suppressed under the mask — it is the verdict's
    // only other carrier (ADR-0020 rule 7).
    const best = masked ? null : decision?.best;
    const cell = (v) => (masked ? HIDDEN : (formatEquity(v) ?? ''));
    return {
        header: [t('analysis.decision'), t('analysis.equity'), t('analysis.error')],
        rows: options.map((option) => ({
            key: option.key,
            label: t(OPTION_LABEL_KEYS[option.key][cubeValue >= 1 ? 1 : 0]),
            cells: [cell(option.equity), cell(option.error)],
            highlight: !masked && isPlayedOption(option.key, isPlayedCubeAction),
            best: option.key === best
        })),
        verdict: {
            label: t('analysis.bestAction'),
            text: masked ? HIDDEN : verdictText(decision, state, t),
            unavailable: state !== DECISION_STATE.VERDICT && state !== DECISION_STATE.PENDING
        }
    };
}

// The provenance footer of a stored record: depth and engine, the engine
// falling back to the record-wide version when the cube analysis has none.
export function cubeInfoRows(cubeAnalysis, { t, engineFallback = '' } = {}) {
    return [
        { label: t('analysis.analysisDepth'), cells: [cubeAnalysis?.analysisDepth ?? ''] },
        { label: t('analysis.engine'), cells: [cubeAnalysis?.analysisEngine || engineFallback || ''] }
    ];
}

// The position facts of a stored cube record, in the compact P/O grid the
// copied image paints beside the decision. The DOM shows these in
// PositionFactsTable, on its own axis (ADR-0018); the image keeps the
// compact grid, but the figures go through the same two rules as everywhere.
export function cubeFactRows(cube) {
    const chance = (v) => formatChance(v) ?? DASH;
    const eq = (v) => formatEquity(v) ?? DASH;
    return {
        header: ['', 'P', 'O'],
        rows: [
            { label: 'W', cells: [chance(cube?.playerWinChances), chance(cube?.opponentWinChances)] },
            { label: 'G', cells: [chance(cube?.playerGammonChances), chance(cube?.opponentGammonChances)] },
            { label: 'B', cells: [chance(cube?.playerBackgammonChances), chance(cube?.opponentBackgammonChances)] },
            { label: 'ND Eq', cells: [eq(cube?.cubelessNoDoubleEquity)] },
            { label: 'D Eq', cells: [eq(cube?.cubelessDoubleEquity)] }
        ]
    };
}

// ---------------------------------------------------------------------------
// Checker candidate list (ADR-0018)
// ---------------------------------------------------------------------------

// Column ids, in display order; the two provenance columns come last so a
// caller that hides them (EPCPanel, ADR-0018 rule 4) just truncates.
export const CHECKER_COLUMNS = ['move', 'equity', 'error', 'pw', 'pg', 'pb', 'ow', 'og', 'ob', 'depth', 'engine'];

const CHECKER_HEADER_KEYS = {
    move: 'analysis.move',
    equity: 'analysis.equity',
    error: 'analysis.error',
    pw: 'analysis.playerWin',
    pg: 'analysis.playerGammon',
    pb: 'analysis.playerBackgammon',
    ow: 'analysis.opponentWin',
    og: 'analysis.opponentGammon',
    ob: 'analysis.opponentBackgammon',
    depth: 'analysis.depth',
    engine: 'analysis.engine'
};

function chanceCells(vector) {
    return [vector?.playerWinChance, vector?.playerGammonChance, vector?.playerBackgammonChance, vector?.opponentWinChance, vector?.opponentGammonChance, vector?.opponentBackgammonChance].map(
        (v) => formatChance(v) ?? DASH
    );
}

/**
 * checkerRows lays out a ranked candidate list. Sorting, truncation and
 * selection stay with the caller; the rows come back in the order given.
 *
 * An absent value is a dash: the best move of a stored record carries no
 * error at all (domain.CheckerMove.EquityError is nil there, by design), and
 * a dash says "nothing to lose here" where +0.000 would claim a measurement.
 *
 * @returns {{ columns: string[], header: string[], baseline: object|null, rows: {key, move, label, cells: string[], highlight: boolean}[] }}
 */
export function checkerRows(moves, { t, isPlayedMove = () => false, showProvenance = true, baseline = null } = {}) {
    const columns = showProvenance ? CHECKER_COLUMNS : CHECKER_COLUMNS.slice(0, 9);
    const provenance = (cells) => (showProvenance ? cells : []);
    return {
        columns,
        header: columns.map((c) => t(CHECKER_HEADER_KEYS[c])),
        // The pre-roll vector (ADR-0018 rule 2): no error figure — the gap to
        // it is the luck of the roll, never the merit of a play (rule 3).
        baseline: baseline
            ? {
                  label: t('eval.baseline'),
                  cells: [formatEquity(baseline.cubelessEquity) ?? DASH, '', ...chanceCells(baseline), ...provenance(['', ''])]
              }
            : null,
        rows: (moves ?? []).map((move) => ({
            key: move.index ?? move.move,
            move,
            label: move.move ?? '',
            cells: [formatEquity(move.equity) ?? DASH, formatEquity(move.equityError) ?? DASH, ...chanceCells(move), ...provenance([move.analysisDepth ?? '', move.analysisEngine ?? ''])],
            highlight: isPlayedMove(move)
        }))
    };
}

// ---------------------------------------------------------------------------
// "Was this played?" — the predicates AnalysisPanel applies to a stored
// record, as pure functions so the copied image highlights exactly the rows
// the panel does. In match mode only the current match's own action counts;
// browsing positions, every recorded play does.
// ---------------------------------------------------------------------------

function normalizeMove(move) {
    return move ? move.split(' ').sort().join(' ') : '';
}

export function playedMovePredicate(analysis, isMatchMode) {
    return (move) => {
        if (!move?.move) return false;
        const target = normalizeMove(move.move);
        if (isMatchMode) return analysis.playedMove ? normalizeMove(analysis.playedMove) === target : false;
        if (analysis.playedMoves?.some((pm) => normalizeMove(pm) === target)) return true;
        return analysis.playedMove ? normalizeMove(analysis.playedMove) === target : false;
    };
}

export function playedCubePredicate(analysis, isMatchMode) {
    return (action) => {
        const parts = normalizeCubeAction(action);
        if (isMatchMode) {
            if (!analysis.playedCubeAction) return false;
            const played = normalizeCubeAction(analysis.playedCubeAction);
            return parts.every((p) => played.includes(p));
        }
        const played = new Set((analysis.playedCubeActions ?? []).flatMap(normalizeCubeAction));
        if (played.size === 0 && analysis.playedCubeAction) {
            for (const p of normalizeCubeAction(analysis.playedCubeAction)) played.add(p);
        }
        return played.size > 0 && parts.every((p) => played.has(p));
    };
}
