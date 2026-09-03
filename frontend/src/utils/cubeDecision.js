// The one shape a cube Decision takes, whatever regime produced its equities
// (ADR-0020). Two sources reach the panel — race.Money, carried by the race
// regimes, and domain.DoublingCubeAnalysis, produced live by gammonNet or read
// from a stored record — and this is where they become one object.
//
// It lives in the frontend, and in one place, for the reason gammonnet_eval.go
// already states about the exact/evaluated merge: "this function itself does
// not know how the two are combined, that merge lives in the frontend's
// displayRace". race.Money keeps its regime and its deliberately
// money-referential exact case (ADR-0017 rule 4); folding it into
// DoublingCubeAnalysis, a type with neither regime nor Referential, would push
// a display distinction into storage.
//
// A pure module on purpose: like moverFactsToSides, it is the part worth
// testing without a DOM.

import { normalizeCubeAction } from './cubeAction.js';

// The three options, in canonical order, never sorted (ADR-0020 rule 1). They
// are NAMED, so unlike a ranked play list their order carries no information —
// and sorting them by equity would permute the rows under the eye across the
// 0-ply → display-depth escalation, on exactly the close decisions a user is
// studying.
export const CUBE_OPTIONS = ['no_double', 'double_take', 'double_pass'];

// What the block's verdict cell is saying. Exactly one of these at a time, and
// it is the single place the block's state is named (ADR-0020 rule 4) — an
// empty cell means "still computing" and nothing else.
export const DECISION_STATE = {
    PENDING: 'pending', // a search is genuinely in flight
    VERDICT: 'verdict', // there is an answer
    NO_DECISION: 'no_decision', // the regime is not entitled to one (estimated, ADR-0009)
    REFUSED: 'refused', // the engine declined the position (beyond the MET's horizon)
    CUBE_OPPONENT: 'cube_opponent', // the opponent owns the cube: nothing to turn
    CRAWFORD: 'crawford' // the Crawford game: no cube in play, by rule
};

// isMoneyPosition is THE money/match predicate on the frontend's own position
// shape — the JS twin of gammonnet.IsMoneyPosition (#190/C.3 point 2). Before
// it existed, cubeTurnability below and EPCPanel's own hasScore each wrote
// this test independently (`score[0] < 0 && score[1] < 0` here,
// `score[0] !== -1 || score[1] !== -1` there): equivalent on a well-formed
// score, and silently NOT equivalent on the malformed one — exactly one side
// carrying the money sentinel — which is the same divergence the Go side had
// between gammonnet_eval.go and domaineval.go before this fiche.
export function isMoneyPosition(position) {
    const score = position?.score ?? [-1, -1];
    return score[0] < 0 && score[1] < 0;
}

// cubeTurnability reports whether the player on roll can turn the cube at all.
// This is a rule of the game read off the board, never an engine output —
// which is why it is computed here and not carried on the wire. gammonNet says
// as much itself (cube.go): "A cube the opponent owns cannot be turned by the
// player on roll: the verdict table presupposes doubling is an option, so
// outside that precondition there is nothing to weigh."
//
// Crawford is read from the away-score sentinel, the same rule
// MatchStateFromPosition uses: either side raw-1-away means this game is the
// Crawford game (0 is the "1-away, post-Crawford" sentinel, see CONTEXT.md).
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

// fromRaceMoney maps a race.Money (`{cubeless, no_double, double_take,
// double_pass, verdict}`) onto the common shape. Cubeless is deliberately
// dropped: ADR-0017 rule 1 makes it a position fact, and the facts table has
// carried it since — the copy in the race decision was the last place that
// said otherwise.
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
        // A stored record's own errors, kept for the `stored` content rule
        // below — never recomputed.
        errors: {
            no_double: cube.cubefulNoDoubleError ?? null,
            double_take: cube.cubefulDoubleTakeError ?? null,
            double_pass: cube.cubefulDoublePassError ?? null
        },
        verdict: verdictKey || null,
        verdictText: cube.bestCubeAction || ''
    };
}

// bestOption is the option to mark, derived from the equities rather than
// taken on trust: the doubling branch is worth the CHEAPER of take/pass (the
// opponent picks), and the best is whichever of that and no-double is higher —
// the same rule domaineval.go and ingest/xgmap.go's computeBestCubeAction
// apply. "Too good" is a verdict, not a fourth option: it still means "do not
// double", so it marks the no-double row.
function bestOption(equities) {
    const { no_double: nd, double_take: dt, double_pass: dp } = equities;
    if (nd == null || dt == null || dp == null) return null;
    const doubling = Math.min(dt, dp);
    if (doubling > nd) return dt <= dp ? 'double_take' : 'double_pass';
    return 'no_double';
}

// bestFromLabel maps a stored record's own best-action string onto one of the
// three canonical rows, reusing normalizeCubeAction — the very function
// AnalysisPanel already trusts to highlight the played action. An absent or
// unparseable string marks nothing: the panel falls back to what it showed
// before this decision, never to a row marked at random.
function bestFromLabel(label) {
    const parts = normalizeCubeAction(label);
    if (!parts.length) return null;
    if (parts.includes('take')) return 'double_take';
    if (parts.includes('pass')) return 'double_pass';
    if (parts.includes('nodouble')) return 'no_double';
    return null;
}

/**
 * cubeDecision builds the block's whole content from whichever source the
 * position has. Returns `{ state, options, verdict, best }` where options is
 * always the three canonical rows, in order, with `equity`/`error` null until
 * a value lands — the structure never depends on the state of the calculation
 * (ADR-0017 rule 3).
 *
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
        // A race whose regime is not entitled to a verdict: not pending, and it
        // will never become pending. ADR-0009 — the cube verdict is never
        // estimated.
        //
        // Only once an evaluation has come back for this position, though: the
        // fast synchronous race path (updateEPC) lands well before gammonNet's
        // own answer, and an estimated block with no money block yet is still
        // waiting for the EVALUATED regime that ADR-0012 makes available. Saying
        // "no decision" in that window would flash a settled state at a position
        // still being computed — the same lie in the other direction.
        else if (race) return { state: settled ? DECISION_STATE.NO_DECISION : DECISION_STATE.PENDING, options: empty, verdict: null, best: null };
    } else if (cubeAnalysis) {
        source = fromCubeAnalysis(cubeAnalysis, verdictKey);
    }

    if (!source) return { state: DECISION_STATE.PENDING, options: empty, verdict: null, best: null };

    const best = bestOption(source.equities);

    // Where doubling is not an option, the equities still inform — "if the
    // cube were centred, doubling would be worth this" — but nothing advises:
    // an error is what a CHOICE costs, and there is no choice (ADR-0020 rule
    // 5). That includes the no-double row, which is not a decision either when
    // the cube is dead.
    if (turnability) {
        return {
            state: turnability,
            options: CUBE_OPTIONS.map((key) => ({ key, equity: source.equities[key], error: null })),
            verdict: null,
            best: null
        };
    }

    // Analysis reports, it does not correct (ADR-0020). A stored record's
    // errors are shown as written — including a best action whose error is not
    // zero, which a rounding or an inconsistent source can produce — and the
    // marked row follows the record's OWN declared best action rather than our
    // arithmetic on its equities. Blanking, and deriving the best, are rules of
    // the path where we compute `equity − best` ourselves.
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
            // Blank on the best option rather than +0.000 — the same rule
            // ADR-0018 rule 3 gives the Baseline band, for the same reason:
            // a zero reads as a measured result, an absence reads as "there
            // is nothing to lose here".
            error: key === best || bestEquity == null || source.equities[key] == null ? null : source.equities[key] - bestEquity
        })),
        verdict: source.verdict,
        verdictText: '',
        best
    };
}
