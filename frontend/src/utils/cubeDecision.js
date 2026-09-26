// The one shape a cube Decision takes, whatever regime produced its equities (ADR-0020): race.Money
// (race regimes) and domain.DoublingCubeAnalysis (live gammonNet or a stored record) become one
// object here. It lives in the frontend because the exact/evaluated merge already does (displayRace):
// folding race.Money, with its regime and money-referential exact case (ADR-0017 rule 4), into
// DoublingCubeAnalysis would push a display distinction into storage. Pure, testable without a DOM.

import { normalizeCubeAction } from './cubeAction.js';

// The three options, in canonical order, never sorted (ADR-0020 rule 1): they are named, and
// sorting by equity would permute rows across the 0-ply → display-depth escalation.
export const CUBE_OPTIONS = ['no_double', 'double_take', 'double_pass'];

// The block's state, exactly one at a time (ADR-0020 rule 4); an empty cell means "still
// computing" and nothing else.
export const DECISION_STATE = {
    PENDING: 'pending', // a search is genuinely in flight
    VERDICT: 'verdict', // there is an answer
    NO_DECISION: 'no_decision', // the regime is not entitled to one (estimated, ADR-0009)
    REFUSED: 'refused', // the engine declined the position (beyond the MET's horizon)
    CUBE_OPPONENT: 'cube_opponent', // the opponent owns the cube: nothing to turn
    CRAWFORD: 'crawford' // the Crawford game: no cube in play, by rule
};

// isMoneyPosition is THE money/match predicate on the frontend (twin of gammonnet.IsMoneyPosition);
// callers must not re-derive it: `score[0] < 0 && score[1] < 0` and `score[0] !== -1 || …`
// diverge when only one side carries the money sentinel.
export function isMoneyPosition(position) {
    const score = position?.score ?? [-1, -1];
    return score[0] < 0 && score[1] < 0;
}

// cubeTurnability reports whether the player on roll can turn the cube at all — a rule of the
// game read off the board, not an engine output. Crawford is read from the away-score sentinel
// like MatchStateFromPosition: either side raw 1 = Crawford game (0 = 1-away post-Crawford, CONTEXT.md).
export function cubeTurnability(position) {
    if (!position) return null;
    const score = position.score ?? [-1, -1];
    const isMoney = isMoneyPosition(position);
    if (!isMoney && (score[0] === 1 || score[1] === 1)) return DECISION_STATE.CRAWFORD;

    const owner = position.cube?.owner ?? -1;
    const onRoll = position.player_on_roll ?? 0;
    if (owner !== -1 && owner !== onRoll) return DECISION_STATE.CUBE_OPPONENT;
    return null;
}

// currentScoreCell names the cube-matrix cell the position stands on ("you are here"). The axes
// are AWAY scores and the verdict does not depend on match length, so the cell is a recomputation
// of the very same decision. Null for money, the Crawford game (raw 1; the grid is post-Crawford,
// raw 0 is its row 1) and an away beyond the grid. Returns `{ awayOnRoll, awayOpponent }`: the
// player on roll decides, so it holds the rows.
export function currentScoreCell(position, gridLength) {
    if (!position || !(gridLength > 0)) return null;
    const score = position.score ?? [-1, -1];
    if (isMoneyPosition(position)) return null;
    if (score[0] === 1 || score[1] === 1) return null; // Crawford: no cube in play
    const away = (raw) => (raw === 0 ? 1 : raw); // post-Crawford 1-away sentinel
    const onRoll = position.player_on_roll === 1 ? 1 : 0;
    const mine = away(score[onRoll]);
    const theirs = away(score[1 - onRoll]);
    if (!(mine >= 1) || !(theirs >= 1)) return null; // a half-money, malformed score
    if (mine > gridLength || theirs > gridLength) return null;
    return { awayOnRoll: mine, awayOpponent: theirs };
}

// fromRaceMoney maps a race.Money onto the common shape. Cubeless is dropped: it is a position
// fact (ADR-0017 rule 1).
function fromRaceMoney(money) {
    return {
        equities: {
            no_double: money.no_double,
            double_take: money.double_take,
            double_pass: money.double_pass
        },
        verdict: money.verdict || null
    };
}

function fromCubeAnalysis(cube, verdictKey) {
    return {
        equities: {
            no_double: cube.cubefulNoDoubleEquity ?? null,
            double_take: cube.cubefulDoubleTakeEquity ?? null,
            double_pass: cube.cubefulDoublePassEquity ?? null
        },
        // A stored record's own errors, never recomputed (see `stored` below).
        errors: {
            no_double: cube.cubefulNoDoubleError ?? null,
            double_take: cube.cubefulDoubleTakeError ?? null,
            double_pass: cube.cubefulDoublePassError ?? null
        },
        verdict: verdictKey || null,
        verdictText: cube.bestCubeAction || ''
    };
}

// bestOption is derived from the equities: the doubling branch is worth the CHEAPER of take/pass,
// the best is the higher of that and no-double (as domaineval.go and xgmap.go's
// computeBestCubeAction). "Too good" is a verdict, not an option: it marks the no-double row.
function bestOption(equities) {
    const { no_double: nd, double_take: dt, double_pass: dp } = equities;
    if (nd == null || dt == null || dp == null) return null;
    const doubling = Math.min(dt, dp);
    if (doubling > nd) return dt <= dp ? 'double_take' : 'double_pass';
    return 'no_double';
}

// bestFromLabel maps a stored best-action string onto a canonical row via normalizeCubeAction.
// Unparseable marks nothing, never a row at random.
function bestFromLabel(label) {
    const parts = normalizeCubeAction(label);
    if (!parts.length) return null;
    if (parts.includes('take')) return 'double_take';
    if (parts.includes('pass')) return 'double_pass';
    if (parts.includes('nodouble')) return 'no_double';
    return null;
}

/**
 * cubeDecision builds the block's content from whichever source the position has. Returns
 * `{ state, options, verdict, best }`; options are always the three canonical rows with
 * `equity`/`error` null until a value lands (ADR-0017 rule 3).
 * @param {object}  args
 * @param {object=} args.race         race.Eval currently on display (displayRace), when the position is a race
 * @param {boolean} args.isRace       whether the position is a pure bearoff at all
 * @param {object=} args.cubeAnalysis domain.DoublingCubeAnalysis from the live evaluation
 * @param {string=} args.verdictKey   the live evaluation's typed verdict (ADR-0020 rule 3)
 * @param {boolean=} args.refused     the engine declined this position
 * @param {string=} args.turnability  cubeTurnability(position)
 * @param {boolean=} args.stored      the record is an imported/stored analysis, not our own computation
 * @param {boolean=} args.settled     an evaluation has come back for THIS position
 */
export function cubeDecision({ race = null, isRace = false, cubeAnalysis = null, verdictKey = '', refused = false, turnability = null, stored = false, settled = true } = {}) {
    const empty = CUBE_OPTIONS.map((key) => ({ key, equity: null, error: null }));

    if (refused) return { state: DECISION_STATE.REFUSED, options: empty, verdict: null, best: null };

    let source = null;
    if (isRace) {
        if (race?.money) source = fromRaceMoney(race.money);
        // A race regime not entitled to a verdict (ADR-0009): never pending — but only once an
        // evaluation has come back. The synchronous race path lands before gammonNet's answer
        // (ADR-0012's evaluated regime); "no decision" in that window would flash a false state.
        else if (race) return { state: settled ? DECISION_STATE.NO_DECISION : DECISION_STATE.PENDING, options: empty, verdict: null, best: null };
    } else if (cubeAnalysis) {
        source = fromCubeAnalysis(cubeAnalysis, verdictKey);
    }

    if (!source) return { state: DECISION_STATE.PENDING, options: empty, verdict: null, best: null };

    const best = bestOption(source.equities);

    // Where doubling is not an option the equities still inform, but nothing advises: an error is
    // what a choice costs, and there is none, not even no-double (ADR-0020 rule 5).
    if (turnability) {
        return {
            state: turnability,
            options: CUBE_OPTIONS.map((key) => ({ key, equity: source.equities[key], error: null })),
            verdict: null,
            best: null
        };
    }

    // Analysis reports, it does not correct (ADR-0020): a stored record's errors are shown as
    // written and the marked row is its OWN declared best action. Blanking and deriving the best
    // apply only when we compute `equity − best` ourselves.
    if (stored) {
        const declared = bestFromLabel(source.verdictText);
        return {
            state: DECISION_STATE.VERDICT,
            options: CUBE_OPTIONS.map((key) => ({ key, equity: source.equities[key], error: source.errors?.[key] ?? null })),
            verdict: null,
            verdictText: source.verdictText,
            best: declared
        };
    }

    const bestEquity = best == null ? null : source.equities[best];
    return {
        state: DECISION_STATE.VERDICT,
        options: CUBE_OPTIONS.map((key) => ({
            key,
            equity: source.equities[key],
            // Blank on the best option rather than +0.000 (as ADR-0018 rule 3): a zero reads as
            // measured, an absence as "nothing to lose".
            error: key === best || bestEquity == null || source.equities[key] == null ? null : source.equities[key] - bestEquity
        })),
        verdict: source.verdict,
        verdictText: '',
        best
    };
}
