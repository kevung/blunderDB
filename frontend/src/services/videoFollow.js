// The Cursor follows the video through the part of a Transcription already timed
// (ADR-0082 rule 3). Pure bookkeeping: the panel reads the clock, asks `step` where
// the Cursor should be, and moves it by a gesture that writes nothing.

// Kinds of Action that start with a roll, and so carry the roll's Repère.
const ROLLING_KINDS = new Set(['checker', 'dance', 'unrecorded']);

// A placement by hand seeks the video a second before the Action's start
// (seekToCursor): the hold must survive that lead.
const HOLD_LEAD_MS = 1500;
// Past its end, the Action placed by hand stays a moment: `v` pressed a little late
// still times it.
const HOLD_GRACE_MS = 2000;
// How far from the instant the panel placed the video its first reading may fall.
const PLACED_SLACK_MS = 1000;

/**
 * The stretch of video each timed Action covers, in document order: from its roll's
 * Repère — else the previous Action's, where a cube decision starts, else its own —
 * up to the start of the next timed Action. The last one ends at its action's Repère,
 * or covers nothing when only its roll is timed: past the last Repère lies the part
 * not yet transcribed. An Action without a Repère covers nothing.
 *
 * @param {any[]} actions - `transcript.ActionInfo`s
 * @returns {{index: number, start: number, end: number, closed: boolean}[]}
 *   `closed`: the Action is timed up to the next one, or up to its own action's Repère.
 */
export function actionSpans(actions) {
    /** @type {{index: number, start: number, end: number, closed: boolean}[]} */
    const spans = [];
    /** @type {number | null} */
    let prevTick = null;
    (actions ?? []).forEach((info, index) => {
        const roll = ROLLING_KINDS.has(info?.kind) && typeof info?.roll_tick_ms === 'number' ? info.roll_tick_ms : null;
        const tick = typeof info?.tick_ms === 'number' ? info.tick_ms : null;
        const start = roll ?? (tick !== null ? (prevTick ?? tick) : null);
        if (start !== null) spans.push({ index, start, end: Math.max(start, tick ?? start), closed: tick !== null });
        if (tick !== null) prevTick = tick;
    });
    for (let i = 0; i < spans.length - 1; i++) {
        spans[i].end = spans[i + 1].start;
        spans[i].closed = true;
    }
    return spans;
}

/**
 * Where the instant `ms` falls: the index of the Action covering it, `'end'` past the
 * last Repère, `null` before the first one or in a document with none.
 *
 * @param {ReturnType<typeof actionSpans>} spans
 * @param {number} ms
 * @returns {number | 'end' | null}
 */
export function coveringAt(spans, ms) {
    if (!spans.length) return null;
    if (ms >= spans[spans.length - 1].end) return 'end';
    const found = spans.find((s) => ms >= s.start && ms < s.end);
    return found ? found.index : null;
}

/**
 * The follower of one panel. `step` is called on every tick of the clock and returns
 * the Cursor stop to move to, or null. It never moves the Cursor while the clock
 * stands still — a paused video is not a reason to take the Cursor from the user —,
 * moves it once when the playback enters an Action's stretch or the part not yet
 * transcribed, and after a placement by hand holds still until the playback has left
 * that Action's stretch: `v` then times the Action the user chose.
 */
/**
 * @param {{ startMs?: number }} [options] - `startMs`: the instant the panel itself put
 *   the video at (a draft reopened where it was left). A first instant read there is no
 *   transition: the Cursor stays where the user types until the playback enters another
 *   stretch.
 */
export function createCursorFollower({ startMs = 0 } = {}) {
    /** @type {number | null | undefined} */
    let lastMs;
    /** @type {number | 'end' | null | undefined} */
    let lastKey;
    /** @type {number | undefined} */
    let lastCursor;
    /** @type {{cursor: number, key: number | 'end' | null} | null} */
    let held = null;

    return {
        /**
         * @param {number | null} ms - the video's instant, null without a ready player
         * @param {any} annotated - `transcript.Annotated`
         * @param {boolean} quiet - the user is typing: watch, never move
         * @returns {number | null} the Cursor stop to move to
         */
        step(ms, annotated, quiet) {
            if (ms === null || !annotated) return null;
            const actions = annotated.actions ?? [];
            const cursor = annotated.cursor ?? actions.length;
            const spans = actionSpans(actions);
            const key = coveringAt(spans, ms);
            const moved = lastMs !== undefined && ms !== lastMs;
            const first = lastMs === undefined;
            lastMs = ms;
            if (first && startMs > 0 && Math.abs(ms - startMs) <= PLACED_SLACK_MS) lastKey = key;

            // A Cursor the follower did not put there was placed by hand.
            if (lastCursor !== undefined && cursor !== lastCursor) held = { cursor, key };
            lastCursor = cursor;

            // While the user types, transitions pass by unheeded. A still clock is no
            // transition either: it is weighed on the next instant that moves, and the
            // first instant read counts as a transition then.
            if (quiet) lastKey = key;
            if (quiet || first || !moved) return null;
            const changed = key !== lastKey;
            lastKey = key;

            if (held) {
                if (!released(held, spans, key, ms)) return null;
                held = null;
            } else if (!changed) {
                return null;
            }

            if (key === null) return null;
            const target = key === 'end' ? actions.length : key;
            if (target === cursor) return null;
            lastCursor = target;
            return target;
        },

        /** The move `step` asked for did not happen: watch the Cursor as it is. */
        forget() {
            lastCursor = undefined;
        }
    };
}

/**
 * Whether the playback has left the stretch of the Action placed by hand.
 *
 * @param {{cursor: number, key: number | 'end' | null}} held
 * @param {ReturnType<typeof actionSpans>} spans
 * @param {number | 'end' | null} key
 * @param {number} ms
 */
function released(held, spans, key, ms) {
    const span = spans.find((s) => s.index === held.cursor);
    if (!span) return key !== held.key;
    if (ms < span.start - HOLD_LEAD_MS) return true;
    return span.closed && ms >= span.end + HOLD_GRACE_MS;
}

/**
 * Whether the user is typing an Action: a die or a play picked, or an entry that no
 * longer says what the Action under it says. The Cursor never leaves it then.
 *
 * @param {any} annotated - `transcript.Annotated`
 * @param {{phase?: string} | null | undefined} keys - the panel's key state
 */
export function isTyping(annotated, keys) {
    if (keys?.phase === 'die1' || keys?.phase === 'candidate') return true;
    const e = annotated?.entry;
    if (!e) return false;
    const dice = e.dice ?? [0, 0];
    if (!e.replacing) return dice[0] > 0 || dice[1] > 0 || !!e.notation;
    const info = annotated.actions?.[e.at];
    if (!info || !ROLLING_KINDS.has(info.kind)) return false;
    // Order-blind: an opening roll is typed player 1's die first, whichever is higher.
    const was = info.before?.dice ?? [0, 0];
    const sameRoll = Math.min(...dice) === Math.min(...was) && Math.max(...dice) === Math.max(...was);
    return !sameRoll || (!!e.notation && e.notation !== info.notation);
}
