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

// The equity column's header key (ADR-0016 point 6, #190/C.3): the figures
// beneath it are on two different scales depending on the position's own
// referential (ADR-0019) — money points at money play, normalised match
// equity at a score — and before this the header just said "Équité" either
// way, the one thing that could have told the reader the scale had changed.
// `isMoney` is undefined at call sites with no position to read a
// referential from (a bare formatting test, the search filter's own
// "Équité (millièmes)" label, which names a stored column across the whole
// database rather than one position's referential and is out of this
// fiche's scope) — those keep the plain, referential-silent label.
function equityHeaderKey(isMoney) {
    if (isMoney === true) return 'analysis.equityMoney';
    if (isMoney === false) return 'analysis.equityMatch';
    return 'analysis.equity';
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
export function cubeRows(decision, { t, cubeValue = 0, isPlayedCubeAction = () => false, masked = false, isMoney } = {}) {
    const options = decision?.options ?? CUBE_OPTIONS.map((key) => ({ key, equity: null, error: null }));
    const state = decision?.state ?? DECISION_STATE.PENDING;
    // The best-row emphasis is suppressed under the mask — it is the verdict's
    // only other carrier (ADR-0020 rule 7).
    const best = masked ? null : decision?.best;
    const cell = (v) => (masked ? HIDDEN : (formatEquity(v) ?? ''));
    return {
        header: [t('analysis.decision'), t(equityHeaderKey(isMoney)), t('analysis.error')],
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

// The reading order of a play: the least advanced checker moves first.
//
// A play is a set of steps, and the order its tokens are written in carries no
// meaning to the engine — which is why they arrive in whatever order their
// producer chose: gammonNet sorts them as strings ("18/14 24/18"), and an
// imported analysis keeps XG's or gnubg's own order. A reader, though, replays
// them one after the other on the board, so a play reads naturally only when it
// starts from the back: "24/18 18/14", never "18/14 24/18".
//
// The rank of a token is its ORIGIN point, in the mover-relative numbering the
// notation already uses (24 = the mover's own back checkers, "bar" further back
// still, 25 here). Ties keep their original order — sort is stable — so
// "8/5 8/3" is left exactly as its producer wrote it.
//
// Anything that does not parse as this notation ("Cannot move", a free-text
// cell) is returned untouched: the rule reorders a play, it never rewrites a
// string it does not understand.
function tokenOrigin(token) {
    const from = token.split('/')[0];
    if (from.toLowerCase() === 'bar') return 25;
    const point = Number.parseInt(from, 10);
    return Number.isNaN(point) || point < 1 || point > 24 ? null : point;
}

export function orderMoveTokens(move) {
    if (!move) return '';
    const tokens = move.trim().split(/\s+/).filter(Boolean);
    const ranked = tokens.map((token) => ({ token, origin: tokenOrigin(token) }));
    if (ranked.some((r) => r.origin === null)) return move;
    return ranked
        .sort((a, b) => b.origin - a.origin)
        .map((r) => r.token)
        .join(' ');
}

// One checker, one displacement: a play the SAME checker made in several steps
// reads as the single move it is — "24/18 18/14" is "24/14".
//
// The step-by-step form carries something the condensed one cannot only when
// the checker HIT on its way through: "24/18* 18/14" says a blot was picked up
// on 18, and writing "24/14" would erase it. So a staging point survives
// exactly when the checker hit on landing there, and disappears otherwise — a
// hit on the FINAL point travels with the condensed token ("24/18 18/14*"
// becomes "24/14*"), since nothing about it is lost.
//
// The rewrite only ever REMOVES a staging point; the surviving ones keep the
// separator their producer wrote them with. gnubg's and XG's own chained form
// stays chained ("24/18*/14"), a play written as separate tokens stays in
// separate tokens, and no play is ever re-spelled just to look uniform.
//
// Two tokens are joined only when they carry the same multiplicity:
// "24/18(2) 18/14(2)" is two checkers that both went the whole way, so it is
// "24/14(2)"; "24/18(2) 18/14" is two checkers whose paths diverged, and the
// notation is already the shortest true statement about them.

const POINT = /^(?:bar|off|[1-9]|1\d|2[0-4])$/i;

// A token is a chain of points ("24/18/14"), each LANDING point optionally
// starred, with an optional multiplicity: "13/7*(2)". Anything else → null,
// and the play is then left exactly as it arrived.
function parseToken(token) {
    let body = token;
    let count = 1;
    const times = /\((\d+)\)$/.exec(body);
    if (times) {
        count = Number.parseInt(times[1], 10);
        body = body.slice(0, times.index);
    }
    const parts = body.split('/');
    if (parts.length < 2) return null;
    const points = [];
    const hits = [];
    for (const [i, part] of parts.entries()) {
        const hit = part.endsWith('*');
        const point = hit ? part.slice(0, -1) : part;
        // A starting point is never hit — only a landing is.
        if (!POINT.test(point) || (i === 0 && hit)) return null;
        points.push(point);
        if (i > 0) hits.push(hit);
    }
    return { points, hits, count };
}

function renderToken({ points, hits, count }) {
    let text = points[0];
    for (let i = 1; i < points.length; i++) text += '/' + points[i] + (hits[i - 1] ? '*' : '');
    return count > 1 ? `${text}(${count})` : text;
}

// hits[i - 1] is the landing on points[i]: keep that point when the checker hit
// there, and always keep the point the play ends on.
function dropIdleStages({ points, hits, count }) {
    const kept = [points[0]];
    const keptHits = [];
    for (let i = 1; i < points.length; i++) {
        if (hits[i - 1] || i === points.length - 1) {
            kept.push(points[i]);
            keptHits.push(hits[i - 1]);
        }
    }
    return { points: kept, hits: keptHits, count };
}

const landsOn = (play) => play.points[play.points.length - 1];
const hitOnLanding = (play) => play.hits[play.hits.length - 1];

// The second play continues the first when it starts where the first ended,
// the same number of checkers made both, and no blot was picked up in between.
function continues(first, second) {
    return first.count === second.count && !hitOnLanding(first) && landsOn(first) === second.points[0];
}

function chain(first, second) {
    return {
        points: [...first.points.slice(0, -1), ...second.points.slice(1)],
        hits: [...first.hits.slice(0, -1), ...second.hits],
        count: first.count
    };
}

export function condenseMoveTokens(move) {
    if (!move) return '';
    const parsed = move.trim().split(/\s+/).filter(Boolean).map(parseToken);
    if (parsed.length === 0 || parsed.some((play) => play === null)) return move;
    const plays = parsed.map(dropIdleStages);
    // The two halves of one checker's journey need not be neighbours in the
    // producer's order ("13/11 12/10 11/9"), so every pair is a candidate; a
    // join shortens the list, and the result may itself continue further, hence
    // the restart. At most four steps in a play — the cost is nothing.
    for (let i = 0; i < plays.length; i++) {
        const next = plays.findIndex((play, j) => j !== i && continues(plays[i], play));
        if (next === -1) continue;
        plays[i] = chain(plays[i], plays[next]);
        plays.splice(next, 1);
        i = -1;
    }
    return plays.map(renderToken).join(' ');
}

// The move cell of a candidate list: condensed first, then read from the back.
export function moveLabel(move) {
    return orderMoveTokens(condenseMoveTokens(move));
}

function chanceCells(vector) {
    return [vector?.playerWinChance, vector?.playerGammonChance, vector?.playerBackgammonChance, vector?.opponentWinChance, vector?.opponentGammonChance, vector?.opponentBackgammonChance].map(
        (v) => formatChance(v) ?? DASH
    );
}

/**
 * checkerRows lays out a ranked candidate list. Sorting, truncation and
 * selection stay with the caller; the rows come back in the order given.
 *
 * The move cell is rewritten by moveLabel above: one checker's several steps
 * become the single displacement they are (hits excepted), and the play is
 * then read from the back — the least advanced checker moves first, whatever
 * order the producer wrote.
 *
 * An absent value is a dash — the project's mark for a value never measured,
 * never for a zero. The one exception is the error column: the best move of a
 * stored record carries no error at all (domain.CheckerMove.EquityError is nil
 * there, by construction — it IS the reference), so its absence is a zero by
 * definition and is written +0.000, as it always was.
 *
 * @returns {{ columns: string[], header: string[], baseline: object|null, rows: {key, move, label, cells: string[], highlight: boolean}[] }}
 */
export function checkerRows(moves, { t, isPlayedMove = () => false, showProvenance = true, baseline = null, isMoney } = {}) {
    const columns = showProvenance ? CHECKER_COLUMNS : CHECKER_COLUMNS.slice(0, 9);
    const provenance = (cells) => (showProvenance ? cells : []);
    return {
        columns,
        header: columns.map((c) => (c === 'equity' ? t(equityHeaderKey(isMoney)) : t(CHECKER_HEADER_KEYS[c]))),
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
            label: moveLabel(move.move),
            cells: [formatEquity(move.equity) ?? DASH, formatEquity(move.equityError ?? 0), ...chanceCells(move), ...provenance([move.analysisDepth ?? '', move.analysisEngine ?? ''])],
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
