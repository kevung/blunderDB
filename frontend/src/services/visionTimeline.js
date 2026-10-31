// What the panel makes of the vision events (ADR-0109 rules 5-7): the markers of the timeline
// and the jump between them, the instant a roll's Repère is snapped to, the verdict "view
// lost", the overlay that changed with no action. Pure functions over the event flow of
// `vision/contract.js`; the panel knows no pixel.

import { EVENT_TYPES, STATES } from '../vision/contract.js';

/** Longest a rest of the dice covers the current instant, in seconds (ADR-0109 rule 5). */
export const REST_WINDOW_MAX = 60;
/** A jump from a marker lands on the next one, not on the marker the playhead sits on, seconds. */
export const JUMP_EPSILON = 0.1;
/** Delay of the stream's overlay behind the board, seconds (ADR-0109 rule 7). */
export const OVERLAY_TOLERANCE = 5;
/** How long an unexpected overlay change stays reported once it can be judged, seconds. */
export const OVERLAY_SHOWN = 30;

/** The events worth a marker: the hands and the clock would drown the line. */
export const MARKER_TYPES = Object.freeze(
    new Set([
        EVENT_TYPES.DICE_THROWN,
        EVENT_TYPES.DICE_REST,
        EVENT_TYPES.DICE_PICKUP,
        EVENT_TYPES.GAME_START,
        EVENT_TYPES.GAME_END,
        EVENT_TYPES.VIEW_LOST,
        EVENT_TYPES.VIEW_FOUND,
        EVENT_TYPES.OVERLAY_CHANGED
    ])
);

/**
 * The shape tells the state apart without the colour: a circle is sure, a diamond to check,
 * a triangle uncertain.
 * @param {string} state
 * @returns {'circle' | 'diamond' | 'triangle'}
 */
export function markerShape(state) {
    if (state === STATES.SURE) return 'circle';
    if (state === STATES.CHECK) return 'diamond';
    return 'triangle';
}

/**
 * @param {readonly import('../vision/contract.js').VisionEvent[]} events
 * @returns {{ type: string, mediaTime: number, state: string, shape: 'circle' | 'diamond' | 'triangle' }[]}
 */
export function timelineMarkers(events) {
    return events
        .filter((e) => MARKER_TYPES.has(e.type))
        .map((e) => ({ type: e.type, mediaTime: e.mediaTime, state: e.state, shape: markerShape(e.state) }))
        .sort((a, b) => a.mediaTime - b.mediaTime);
}

/**
 * The marker a jump from `at` lands on: the first strictly after (forward) or before
 * (backward), every state included. Null when there is none.
 * @param {{ mediaTime: number }[]} markers sorted by time
 * @param {number} at seconds
 * @param {1 | -1} direction
 */
export function jumpTarget(markers, at, direction) {
    if (direction > 0) return markers.find((m) => m.mediaTime > at + JUMP_EPSILON) ?? null;
    for (let i = markers.length - 1; i >= 0; i--) if (markers[i].mediaTime < at - JUMP_EPSILON) return markers[i];
    return null;
}

/**
 * Whether the calibrated board is out of the picture at `at`: the last verdict of the
 * presence, up to then, says so.
 * @param {readonly import('../vision/contract.js').VisionEvent[]} events
 * @param {number} [at] seconds; the whole flow by default
 */
export function viewLost(events, at = Infinity) {
    let lost = false;
    let seen = -Infinity;
    for (const e of events) {
        if (e.mediaTime > at || e.mediaTime < seen) continue;
        if (e.type === EVENT_TYPES.VIEW_LOST) [lost, seen] = [true, e.mediaTime];
        else if (e.type === EVENT_TYPES.VIEW_FOUND) [lost, seen] = [false, e.mediaTime];
    }
    return lost;
}

/**
 * The instant the first die of a new Action stamps as the roll's Repère: the `mediaTime` of
 * the latest rest of the dice that covers `at` when it is sure — from its first image to the
 * next pick-up or throw, at most REST_WINDOW_MAX later. Null leaves the current instant.
 * @param {readonly import('../vision/contract.js').VisionEvent[]} events
 * @param {number} at seconds
 * @returns {number | null} seconds
 */
export function snapRollTime(events, at) {
    if (viewLost(events, at)) return null;
    let rest = null;
    for (const e of events) {
        if (e.type !== EVENT_TYPES.DICE_REST || e.mediaTime > at) continue;
        if (rest && e.mediaTime < rest.mediaTime) continue;
        const end = events.reduce((m, o) => (o.mediaTime > e.mediaTime && (o.type === EVENT_TYPES.DICE_PICKUP || o.type === EVENT_TYPES.DICE_THROWN) ? Math.min(m, o.mediaTime) : m), Infinity);
        if (at < end && at - e.mediaTime <= REST_WINDOW_MAX) rest = e;
    }
    return rest && rest.state === STATES.SURE ? rest.mediaTime : null;
}

/**
 * The overlay change to report: the latest one old enough to be judged, still recent, that no
 * Action explains. An Action explains it when one of its Repères falls within the overlay's
 * delay of the change, on either side.
 * @param {readonly import('../vision/contract.js').VisionEvent[]} events
 * @param {readonly number[]} actionTimes seconds, the Repères of the Actions
 * @param {number} at seconds, the current instant
 */
export function unexpectedOverlay(events, actionTimes, at) {
    let found = null;
    for (const e of events) {
        if (e.type !== EVENT_TYPES.OVERLAY_CHANGED) continue;
        if (at - e.mediaTime < OVERLAY_TOLERANCE || at - e.mediaTime > OVERLAY_SHOWN) continue;
        if (actionTimes.some((t) => Math.abs(t - e.mediaTime) <= OVERLAY_TOLERANCE)) continue;
        if (!found || e.mediaTime > found.mediaTime) found = e;
    }
    return found;
}
