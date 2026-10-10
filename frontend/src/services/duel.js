// duel.js — the pure part of the Duel tab (ADR-0072, ADR-0073): the creation form, the clocks
// read from the Arbiter's state, and what changed between two states. No store, no binding.

/** The Bot's levels, the analysis's own (ADR-0072 rule 7). */
export const BOT_LEVELS = ['instant', 'normal', 'thorough'];

/** The longest match the equity table covers (ADR-0072 rule 12). */
export const MAX_MATCH_LENGTH = 25;

/** Where the first game starts: the opening position, or the Position on the board. */
export const START = Object.freeze({ OPENING: 'opening', BOARD: 'board' });

/** Away scores' sentinels (CONTEXT.md « Away score »): one point away, in or after the Crawford game. */
export const CRAWFORD = 1;
export const POST_CRAWFORD = 0;

/**
 * The time controls offered before any is saved: minutes of reserve per point of the match and
 * the delay per move, in seconds. The Arbiter's offer, when it names the same, wins.
 */
export const PRESET_CADENCES = Object.freeze([Object.freeze({ name: 'standard', minutesPerPoint: 2, delay: 12 }), Object.freeze({ name: 'speed', minutesPerPoint: 0.4, delay: 10 })]);

/**
 * The form as it is first shown, and as a remembered one is completed.
 * `cadence` is the name of a time control, or '' for none; `cadences` the player's own.
 */
export const DEFAULT_FORM = Object.freeze({
    money: false,
    matchLength: 7,
    jacoby: true,
    start: START.OPENING,
    away: [7, 7],
    reroll: false,
    afterCube: false,
    singleGame: false,
    side: 0,
    level: 'normal',
    cadence: '',
    minutesPerPoint: 2,
    delay: 12,
    cadences: /** @type {{name: string, minutesPerPoint: number, delay: number}[]} */ ([]),
    timeOut: 'continue',
    player: '',
    record: true,
    pipcount: true
});

/**
 * @param {any} n
 * @param {number} lo
 * @param {number} hi
 * @param {number} d
 */
function clamp(n, lo, hi, d) {
    const v = Math.trunc(Number(n));
    return Number.isFinite(v) ? Math.min(hi, Math.max(lo, v)) : d;
}

/**
 * Minutes per point, to the tenth: the Arbiter counts whole seconds.
 * @param {any} v
 * @param {number} d
 */
function minutes(v, d) {
    const n = Number(v);
    return Number.isFinite(n) && n > 0 ? Math.min(60, Math.round(n * 600) / 600) : d;
}

/**
 * A saved time control, or null when it is not one.
 * @param {any} c
 */
function savedCadence(c) {
    if (!c || typeof c !== 'object' || typeof c.name !== 'string' || !c.name.trim()) return null;
    return { name: c.name.trim(), minutesPerPoint: minutes(c.minutesPerPoint, 2), delay: clamp(c.delay, 0, 600, 0) };
}

/**
 * A remembered form, completed and clamped: what an older version stored, or a hand-edited
 * value, never reaches the Arbiter as is. The choices about the board's roll and cube belong
 * to the position, and start again at their defaults.
 * @param {any} saved
 */
export function normalizeForm(saved) {
    const f = { ...DEFAULT_FORM, ...(saved && typeof saved === 'object' ? saved : {}) };
    f.money = !!f.money;
    f.matchLength = clamp(f.matchLength, 1, MAX_MATCH_LENGTH, DEFAULT_FORM.matchLength);
    f.jacoby = !!f.jacoby;
    if (!Object.values(START).includes(f.start)) f.start = START.OPENING;
    const away = Array.isArray(f.away) ? f.away : [f.matchLength, f.matchLength];
    f.away = [clamp(away[0], 0, f.matchLength, f.matchLength), clamp(away[1], 0, f.matchLength, f.matchLength)];
    f.reroll = false;
    f.afterCube = false;
    f.singleGame = !!f.singleGame;
    f.side = f.side === 1 ? 1 : 0;
    if (!BOT_LEVELS.includes(f.level)) f.level = DEFAULT_FORM.level;
    f.cadences = (Array.isArray(f.cadences) ? f.cadences : []).map(savedCadence).filter(Boolean);
    f.cadence = typeof f.cadence === 'string' ? f.cadence : '';
    // The tournament preset became the standard one; the fixed reserves are offered no longer.
    if (f.cadence === 'tournament') f.cadence = 'standard';
    if (f.cadence && !PRESET_CADENCES.some((p) => p.name === f.cadence) && !f.cadences.some((/** @type {any} */ c) => c.name === f.cadence)) f.cadence = '';
    f.minutesPerPoint = minutes(f.minutesPerPoint, DEFAULT_FORM.minutesPerPoint);
    f.delay = clamp(f.delay, 0, 600, DEFAULT_FORM.delay);
    f.timeOut = f.timeOut === 'lose_match' ? 'lose_match' : 'continue';
    f.player = typeof f.player === 'string' ? f.player : '';
    f.record = f.record !== false;
    f.pipcount = f.pipcount !== false;
    return f;
}

/**
 * The time controls the menu offers: the presets — the Arbiter's when it offers them — then the
 * player's own.
 * @param {any} form
 * @param {any[]} [offered] the Arbiter's named Cadences (`reservePerPoint` in seconds)
 */
export function cadenceChoices(form, offered = []) {
    const presets = PRESET_CADENCES.map((p) => {
        const o = (offered ?? []).find((c) => c?.name === p.name && c.reservePerPoint > 0);
        return o ? { name: p.name, minutesPerPoint: o.reservePerPoint / 60, delay: o.delay ?? 0, preset: true } : { ...p, preset: true };
    });
    const own = (form?.cadences ?? []).filter((/** @type {any} */ c) => !presets.some((p) => p.name === c.name)).map((/** @type {any} */ c) => ({ ...c, preset: false }));
    return [...presets, ...own];
}

/**
 * The form with the time control `name` chosen: its two numbers fill the fields.
 * @param {any} form
 * @param {string} name '' for none
 * @param {any[]} [offered]
 */
export function chooseCadence(form, name, offered = []) {
    const c = cadenceChoices(form, offered).find((x) => x.name === name);
    if (!c) return { ...form, cadence: '' };
    return { ...form, cadence: c.name, minutesPerPoint: c.minutesPerPoint, delay: c.delay };
}

/**
 * The form with the fields' two numbers saved as the player's time control `name`, and chosen;
 * a preset's name is not taken over.
 * @param {any} form
 * @param {string} name
 */
export function saveCadence(form, name) {
    const n = (name ?? '').trim();
    if (!n || PRESET_CADENCES.some((p) => p.name === n)) return form;
    const c = savedCadence({ name: n, minutesPerPoint: form.minutesPerPoint, delay: form.delay });
    if (!c) return form;
    const cadences = [...(form.cadences ?? []).filter((/** @type {any} */ x) => x.name !== n), c];
    return { ...form, cadences, cadence: n };
}

/**
 * The form without the player's time control `name`; a preset stays.
 * @param {any} form
 * @param {string} name
 */
export function deleteCadence(form, name) {
    const cadences = (form.cadences ?? []).filter((/** @type {any} */ x) => x.name !== name);
    return { ...form, cadences, cadence: form.cadence === name ? '' : form.cadence };
}

/**
 * The Away scores a match of `length` offers each player, highest first: the two sentinels of
 * one point away close the list (a 1-point match is the Crawford game only).
 * @param {number} length
 */
export function awayChoices(length) {
    if (length <= 1) return [CRAWFORD];
    return [...Array.from({ length: length - 1 }, (_, i) => length - i), CRAWFORD, POST_CRAWFORD];
}

/**
 * The form as the board's Position states it: a money position makes a money session, a score
 * the Away scores, the length stretched to hold them.
 * @param {any} form
 * @param {any} position
 */
export function formFromBoard(form, position) {
    const score = position?.score;
    if (!Array.isArray(score)) return form;
    if (score[0] < 0 || score[1] < 0) return form.money ? form : { ...form, money: true };
    const away = [score[0], score[1]];
    const matchLength = Math.min(MAX_MATCH_LENGTH, Math.max(form.matchLength, away[0], away[1], 1));
    if (!form.money && form.matchLength === matchLength && form.away[0] === away[0] && form.away[1] === away[1]) return form;
    return { ...form, money: false, matchLength, away };
}

/**
 * The board's Position with the form's score: the Away scores, or money.
 * @param {any} position
 * @param {any} form
 */
export function boardFromForm(position, form) {
    const score = form.money ? [-1, -1] : [Math.min(form.away[0], form.matchLength), Math.min(form.away[1], form.matchLength)];
    if (!position || (position.score?.[0] === score[0] && position.score?.[1] === score[1])) return position;
    return { ...position, score };
}

/**
 * Whether the board's Position carries a roll to play.
 * @param {any} position
 */
export function boardHasDice(position) {
    return !!position && (position.dice?.[0] ?? 0) > 0 && (position.dice?.[1] ?? 0) > 0;
}

/**
 * Whether a Duel from the board begins at a cube decision the form may place itself before or
 * after: no roll to play, and either a double shown offered or a cube the side on roll may turn.
 * The Arbiter keeps the last word; this only decides whether the choice is shown.
 * @param {any} position
 * @param {any} form
 */
export function boardCubeDecision(position, form) {
    if (!position || form.start !== START.BOARD) return false;
    if (boardHasDice(position) && !form.reroll) return false;
    const onRoll = position.player_on_roll;
    const cube = position.cube ?? { owner: -1, value: 0 };
    if (position.decision_type === 1 && ((cube.owner === -1 && cube.value > 0) || cube.owner === 1 - onRoll)) return true;
    if (!form.money && form.matchLength > 1 && (form.away[0] === CRAWFORD || form.away[1] === CRAWFORD)) return false;
    return cube.owner === -1 || cube.owner === onRoll;
}

/**
 * `duel.Settings` from the form. `boardPosition` is the Position on the board, for a Start
 * there; a session the Arbiter refuses is refused there, with its reason.
 * @param {any} form
 * @param {any} cadences the named Cadences the Arbiter offers
 * @param {any} [boardPosition]
 */
export function settingsFromForm(form, cadences, boardPosition = null) {
    const f = normalizeForm({ ...form, reroll: false, afterCube: false });
    const human = { kind: 'external', name: f.player.trim() };
    const bot = { kind: 'bot', level: f.level };
    /** @type {any} */
    const set = {
        matchLength: f.money ? 0 : f.matchLength,
        jacoby: f.money && f.jacoby,
        sides: f.side === 0 ? [human, bot] : [bot, human],
        discardAtEnd: !f.record
    };
    if (f.singleGame) set.singleGame = true;
    if (f.start === START.BOARD && boardPosition) {
        const position = { ...boardFromForm(boardPosition, f) };
        delete position.id;
        set.start = position;
        if (form?.reroll && boardHasDice(boardPosition)) set.reroll = true;
        if (form?.afterCube) set.afterCube = true;
    } else if (!f.money && (f.away[0] !== f.matchLength || f.away[1] !== f.matchLength)) {
        set.away = [f.away[0], f.away[1]];
    }
    // A reserve per point needs a length: a money session plays without a clock.
    if (f.cadence && !f.money) {
        const c = cadenceChoices(f, cadences).find((x) => x.name === f.cadence);
        const same = c && c.minutesPerPoint === f.minutesPerPoint && c.delay === f.delay;
        set.cadence = { name: same ? c.name : '', reservePerPoint: Math.round(f.minutesPerPoint * 60), delay: f.delay, timeOut: f.timeOut };
    }
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
    /** @type {any[]} */
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

/**
 * The key and values of a level's label in the selector: its depth comes from
 * the offer (the Go side reads gammonNet's table), never from a copy here. A
 * level the offer does not describe keeps its bare name.
 *
 * @param {string} name
 * @param {readonly { name: string, ply: number, pruneK: number }[] | null | undefined} levels
 */
export function levelLabelParts(name, levels) {
    const info = (levels ?? []).find((l) => l.name === name);
    if (!info) return { key: null, params: { name } };
    return { key: info.pruneK > 0 ? 'duel.levelPruned' : 'duel.levelPly', params: { name, ply: info.ply } };
}
