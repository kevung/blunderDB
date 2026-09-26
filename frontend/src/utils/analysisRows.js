// The formatted rows of an analysis, built once for every surface that shows them: the two DOM
// tables (CubeVerdictTable, CandidateMovesTable) and the canvas of "copy board with analysis".
// This is the one place a number becomes a cell; consumers only lay out strings, so figures
// leave exactly as the backend supplied them, formatted, never converted (ADR-0019).
// Pure: no store, no DOM, no i18n import — `t` is a parameter, so the canvas (`translate`) and
// a component (`$t`) get identical labels.

import { CUBE_OPTIONS, DECISION_STATE } from './cubeDecision.js';

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

// The equity column's header key (ADR-0016 point 6): money points at money play, normalised
// match equity at a score (ADR-0019), so the header says which. `isMoney` undefined (no position
// to read a referential from, e.g. the search filter naming a stored column) keeps the plain label.
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

// playedCubeAction speaks the legacy vocabulary ("Double", "Take", …), so canonical keys are
// translated back here rather than changing a contract shared by two panels and the board.
export function isPlayedOption(key, isPlayedCubeAction) {
    if (key === 'no_double') return isPlayedCubeAction('No Double');
    if (key === 'double_take') return isPlayedCubeAction('Double') && isPlayedCubeAction('Take');
    return isPlayedCubeAction('Double') && isPlayedCubeAction('Pass');
}

// The verdict cell's text (ADR-0020 rule 4): empty only while computing.
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
            // A live verdict is a key (translated, keeps "too good"); a stored record's engine
            // words are reported verbatim.
            return decision?.verdict ? t('cube.verdicts.' + decision.verdict) : (decision?.verdictText ?? '');
    }
}

/**
 * cubeRows lays out a cube Decision (utils/cubeDecision.js) as three option rows plus the
 * verdict. An absent equity or error is an EMPTY cell, not a dash: here emptiness is a state —
 * pending (rule 4), the best option's "nothing to lose" (rule 2), a dead cube (rule 5).
 *
 * @returns {{ header: string[], rows: {key, label, cells: string[], highlight: boolean, best: boolean}[], verdict: {label, text, unavailable} }}
 */
export function cubeRows(decision, { t, cubeValue = 0, isPlayedCubeAction = () => false, masked = false, isMoney } = {}) {
    const options = decision?.options ?? CUBE_OPTIONS.map((key) => ({ key, equity: null, error: null }));
    const state = decision?.state ?? DECISION_STATE.PENDING;
    // No best-row emphasis under the mask: the verdict's only other carrier (ADR-0020 rule 7).
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

// The position facts of a stored cube record, in the compact P/O grid the copied image paints
// beside the decision (the DOM uses PositionFactsTable, ADR-0018), through the same two rules.
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
// caller that hides them (EvalPanel, ADR-0018 rule 4) just truncates.
export const CHECKER_COLUMNS = ['move', 'equity', 'error', 'pw', 'pg', 'pb', 'ow', 'og', 'ob', 'depth', 'engine'];

/**
 * The two named projections of a candidate list (ADR-0048 decision 4). `judge` weighs a play:
 * the six probability columns are the matter of it (Eval and Analysis panels). `identify`
 * recognises a play already made (transcription): the probabilities cost 290 px, which breaks
 * the three-column layout at 1024 px; `error` stays, to flag a blunder at validation.
 * Named rather than a free `columns` list, so a third caller has to justify itself here.
 */
export const CHECKER_PROJECTIONS = Object.freeze({
    judge: CHECKER_COLUMNS,
    identify: ['move', 'equity', 'error']
});

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

// The reading order of a play: the least advanced checker moves first ("24/18 18/14", never
// "18/14 24/18"), since producers emit any order (gammonNet sorts as strings, imports keep XG's
// or gnubg's). Rank = ORIGIN point, mover-relative, bar = 25. The sort is stable, so ties keep
// the producer's order. A token that does not parse ("Cannot move") is returned untouched.
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

// One checker, one displacement: "24/18 18/14" reads "24/14". A staging point survives only
// when the checker HIT there ("24/18* 18/14" keeps the pickup on 18); a hit on the final point
// travels with the condensed token ("24/14*"). The rewrite only removes staging points; the
// survivors keep their producer's separator ("24/18*/14" stays chained). Tokens join only with
// equal multiplicity: "24/18(2) 18/14" is two checkers whose paths diverged.

const POINT = /^(?:bar|off|[1-9]|1\d|2[0-4])$/i;

// A token: a chain of points ("24/18/14"), each landing optionally starred, optional
// multiplicity ("13/7*(2)"). Anything else → null, and the play is left as it arrived.
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
    // The halves of one checker's journey need not be neighbours ("13/11 12/10 11/9"), so every
    // pair is a candidate, and a join may continue further, hence the restart (≤ 4 steps).
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
 * checkerRows lays out a ranked candidate list, in the order given (sorting, truncation and
 * selection stay with the caller); move cells go through moveLabel. An absent value is a dash
 * (never measured, never a zero) — except the error column: the best move of a stored record
 * has no error by construction (it IS the reference), written +0.000.
 *
 * @returns {{ columns: string[], header: string[], baseline: object|null, rows: {key, move, label, cells: string[], highlight: boolean}[] }}
 */
export function checkerRows(moves, { t, isPlayedMove = () => false, showProvenance = true, baseline = null, isMoney, projection = 'judge' } = {}) {
    // `judge` keeps the truncation by showProvenance (ADR-0018 rule 4), not a projection.
    const identifying = projection === 'identify';
    const columns = identifying ? CHECKER_PROJECTIONS.identify : showProvenance ? CHECKER_COLUMNS : CHECKER_COLUMNS.slice(0, 9);
    const provenance = (cells) => (showProvenance && !identifying ? cells : []);
    const chances = (row) => (identifying ? [] : chanceCells(row));
    return {
        columns,
        header: columns.map((c) => (c === 'equity' ? t(equityHeaderKey(isMoney)) : t(CHECKER_HEADER_KEYS[c]))),
        // The pre-roll vector (ADR-0018 rule 2): no error figure — the gap to
        // it is the luck of the roll, never the merit of a play (rule 3).
        baseline: baseline
            ? {
                  label: t('eval.baseline'),
                  cells: [formatEquity(baseline.cubelessEquity) ?? DASH, '', ...chances(baseline), ...provenance(['', ''])]
              }
            : null,
        rows: (moves ?? []).map((move) => ({
            key: move.index ?? move.move,
            move,
            label: moveLabel(move.move),
            cells: [formatEquity(move.equity) ?? DASH, formatEquity(move.equityError ?? 0), ...chances(move), ...provenance([move.analysisDepth ?? '', move.analysisEngine ?? ''])],
            highlight: isPlayedMove(move)
        }))
    };
}

// "Was this played?" lives once in utils/playedMarks.js, shared by every surface that highlights.
