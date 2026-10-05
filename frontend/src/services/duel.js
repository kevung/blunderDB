// duel.js — the pure part of the Duel tab (ADR-0072, ADR-0073): the creation form, the clocks
// read from the Arbiter's state, and what changed between two states. No store, no binding.

/** The Bot's levels, the analysis's own (ADR-0072 rule 7). */
export const BOT_LEVELS = ['instant', 'normal', 'thorough'];

/** The longest match the equity table covers (ADR-0072 rule 12). */
export const MAX_MATCH_LENGTH = 25;

/** Where the first game starts. */
export const START = Object.freeze({ OPENING: 'opening', BOARD: 'board', SCORE: 'score' });

/**
 * The form as it is first shown, and as a remembered one is completed.
 * `cadence` is a named Cadence, or '' for none.
 */
export const DEFAULT_FORM = Object.freeze({
    money: false,
    matchLength: 7,
    jacoby: true,
    start: START.OPENING,
    away: [7, 7],
    side: 0,
    level: 'normal',
    cadence: '',
    timeOut: 'continue',
    player: '',
    record: true
});

/**
 * A remembered form, completed and clamped: what an older version stored, or a hand-edited
 * value, never reaches the Arbiter as is.
 * @param {any} saved
 */
export function normalizeForm(saved) {
    const f = { ...DEFAULT_FORM, ...(saved && typeof saved === 'object' ? saved : {}) };
    const clamp = (/** @type {any} */ n, /** @type {number} */ lo, /** @type {number} */ hi, /** @type {number} */ d) => {
        const v = Math.trunc(Number(n));
        return Number.isFinite(v) ? Math.min(hi, Math.max(lo, v)) : d;
    };
    f.money = !!f.money;
    f.matchLength = clamp(f.matchLength, 1, MAX_MATCH_LENGTH, DEFAULT_FORM.matchLength);
    f.jacoby = !!f.jacoby;
    if (!Object.values(START).includes(f.start)) f.start = START.OPENING;
    const away = Array.isArray(f.away) ? f.away : DEFAULT_FORM.away;
    f.away = [clamp(away[0], 1, f.matchLength, f.matchLength), clamp(away[1], 1, f.matchLength, f.matchLength)];
    f.side = f.side === 1 ? 1 : 0;
    if (!BOT_LEVELS.includes(f.level)) f.level = DEFAULT_FORM.level;
    f.cadence = typeof f.cadence === 'string' ? f.cadence : '';
    f.timeOut = f.timeOut === 'lose_match' ? 'lose_match' : 'continue';
    f.player = typeof f.player === 'string' ? f.player : '';
    f.record = f.record !== false;
    return f;
}

/**
 * The opening position at a chosen score, no roll on it: `away` is what each player still
 * needs, 1 being the Crawford game (CONTEXT.md « Away score »). The Arbiter draws the opening
 * roll.
 * @param {number} matchLength
 * @param {number[]} away
 */
export function scoreStart(matchLength, away) {
    return {
        board: openingBoard(),
        cube: { owner: -1, value: 0 },
        dice: [0, 0],
        score: [away[0], away[1]],
        player_on_roll: 0,
        decision_type: 0,
        has_jacoby: 0,
        has_beaver: 0
    };
}

/** The opening board, in `domain.Board`'s shape (point 0 and 25 are the bars). */
function openingBoard() {
    const points = Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 }));
    /** @type {[number, number, number][]} */
    const layout = [
        [24, 2, 0],
        [13, 5, 0],
        [8, 3, 0],
        [6, 5, 0],
        [1, 2, 1],
        [12, 5, 1],
        [17, 3, 1],
        [19, 5, 1]
    ];
    for (const [point, checkers, color] of layout) points[point] = { checkers, color };
    return { points, bearoff: [0, 0] };
}

/**
 * `duel.Settings` from the form. `boardPosition` is the Position on the board, for a Start
 * there; a session the Arbiter refuses is refused there, with its reason.
 * @param {any} form
 * @param {any} cadences the named Cadences the Arbiter offers
 * @param {any} [boardPosition]
 */
export function settingsFromForm(form, cadences, boardPosition = null) {
    const f = normalizeForm(form);
    const human = { kind: 'external', name: f.player.trim() };
    const bot = { kind: 'bot', level: f.level };
    /** @type {any} */
    const set = {
        matchLength: f.money ? 0 : f.matchLength,
        jacoby: f.money && f.jacoby,
        sides: f.side === 0 ? [human, bot] : [bot, human],
        discardAtEnd: !f.record
    };
    if (f.start === START.BOARD && boardPosition) {
        const position = { ...boardPosition };
        delete position.id;
        set.start = position;
    } else if (f.start === START.SCORE && !f.money) {
        set.start = scoreStart(f.matchLength, f.away);
    }
    const cadence = (cadences ?? []).find((/** @type {any} */ c) => c.name === f.cadence);
    if (cadence) set.cadence = { ...cadence, timeOut: f.timeOut };
    return set;
}

/**
 * The player's Side: the external one. A desktop Duel has exactly one.
 * @param {any} state a `duel.State`
 */
export function humanSide(state) {
    const sides = state?.sides ?? [];
    return sides[1]?.kind === 'external' && sides[0]?.kind !== 'external' ? 1 : 0;
}

/**
 * Each Side's reserve left, in ms, at `now`, and the delay left of the running turn; `null`
 * without a Cadence. The running Side is the awaited one; the other's reserve stands still
 * (ADR-0073: a reserve and a delay, no increment).
 * @param {any} state a `duel.State`
 * @param {number} now epoch ms
 */
export function clockView(state, now) {
    const clock = state?.clock;
    if (!clock) return null;
    const reserve = [clock.reserve?.[0] ?? 0, clock.reserve?.[1] ?? 0];
    const delay = (clock.cadence?.delay ?? 0) * 1000;
    const awaiting = state.awaiting;
    const running = awaiting && !state.ended ? awaiting.side : -1;
    let delayLeft = 0;
    if (running >= 0) {
        const since = Date.parse(awaiting.since ?? '');
        const elapsed = (clock.spent ?? 0) + (Number.isFinite(since) && since > 0 ? Math.max(now - since, 0) : 0);
        const turn = clock.turn ?? 0;
        const charge = Math.max(turn + elapsed - delay, 0) - Math.max(turn - delay, 0);
        reserve[running] -= charge;
        delayLeft = Math.max(delay - turn - elapsed, 0);
    }
    return { reserve, running, delayLeft, overTime: clock.overTime ?? 0 };
}

/**
 * `m:ss`, a negative reserve shown as its overrun.
 * @param {number} ms
 */
export function formatClock(ms) {
    const sign = ms < 0 ? '−' : '';
    const s = Math.ceil(Math.abs(ms) / 1000);
    return `${sign}${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`;
}

/**
 * The Actions the Arbiter and the Bot played since `before`, the player's own excepted: what the
 * board replays, slowly, before handing the decision back (ADR-0072 rule 11, the "natural" delay
 * is an animation).
 * @param {any} before a match sheet (`transcript.Annotated`), or null
 * @param {any} after the next one
 * @param {number} human the player's Side
 */
export function framesBetween(before, after, human) {
    const from = before?.actions?.length ?? 0;
    const infos = after?.actions ?? [];
    return infos.slice(from).filter((info) => info.side !== human && info.has_position && info.kind === 'checker');
}

/**
 * The scoreline a Duel shows: points, and the length or the money session.
 * @param {any} state
 */
export function scoreline(state) {
    const score = state?.score ?? [0, 0];
    const length = state?.header?.match_length ?? 0;
    return { score: [score[0] ?? 0, score[1] ?? 0], length };
}
