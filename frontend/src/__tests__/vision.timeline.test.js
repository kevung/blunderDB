/**
 * vision.timeline.test.js — what the panel makes of the vision events (ADR-0109 rules 5-7):
 * the markers and the jumps, the snap of a roll's Repère, the view verdict, the overlay drift.
 * Each threshold and each convention has a test that fails when it moves.
 */
import { describe, test, expect } from 'vitest';
import { EVENT_TYPES as T, STATES as S, makeEvent } from '../vision/contract.js';
import { markerShape, timelineMarkers, jumpTarget, viewLost, snapRollTime, recentOverlay } from '../services/visionTimeline.js';

const ev = (/** @type {string} */ type, /** @type {number} */ t, state = S.SURE) => makeEvent(type, t, state);

describe('markers', () => {
    test('each state has its own shape, not only a colour', () => {
        expect([markerShape(S.SURE), markerShape(S.CHECK), markerShape(S.UNCERTAIN)]).toEqual(['circle', 'diamond', 'triangle']);
    });

    test('hands, clock and errors make no marker; the others do, in time order', () => {
        const events = [
            ev(T.DICE_REST, 9),
            ev(T.HANDS_IN, 1),
            ev(T.CLOCK, 2),
            ev(T.ERROR, 3),
            ev(T.FRAME_UNCERTAIN, 4),
            ev(T.DICE_THROWN, 5),
            ev(T.GAME_START, 0.5),
            ev(T.OVERLAY_CHANGED, 7),
            ev(T.VIEW_LOST, 8)
        ];
        expect(timelineMarkers(events).map((m) => m.type)).toEqual([T.GAME_START, T.DICE_THROWN, T.OVERLAY_CHANGED, T.VIEW_LOST, T.DICE_REST]);
    });
});

describe('jump', () => {
    const markers = timelineMarkers([ev(T.DICE_THROWN, 10), ev(T.DICE_REST, 12, S.CHECK), ev(T.DICE_PICKUP, 20, S.UNCERTAIN)]);

    test('forward and backward land on the neighbour, a marker to check or uncertain being a stop', () => {
        expect(jumpTarget(markers, 10.5, 1)?.mediaTime).toBe(12);
        expect(jumpTarget(markers, 12, 1)?.mediaTime).toBe(20);
        expect(jumpTarget(markers, 15, -1)?.mediaTime).toBe(12);
        expect(jumpTarget(markers, 12, -1)?.mediaTime).toBe(10);
    });

    test('the playhead sitting on a marker within an epsilon does not stop the jump on it', () => {
        expect(jumpTarget(markers, 12.05, -1)?.mediaTime).toBe(10);
        expect(jumpTarget(markers, 11.95, 1)?.mediaTime).toBe(20);
        expect(jumpTarget(markers, 12.2, -1)?.mediaTime).toBe(12);
        expect(jumpTarget(markers, 11.8, 1)?.mediaTime).toBe(12);
    });

    test('nothing beyond the ends', () => {
        expect(jumpTarget(markers, 20, 1)).toBeNull();
        expect(jumpTarget(markers, 10, -1)).toBeNull();
    });
});

describe('view verdict', () => {
    test('lost until found, judged at the instant asked', () => {
        const events = [ev(T.VIEW_LOST, 10), ev(T.VIEW_FOUND, 20)];
        expect([viewLost(events, 5), viewLost(events, 15), viewLost(events, 25), viewLost(events)]).toEqual([false, true, false, false]);
    });
    test('events arriving out of order do not undo a later verdict', () => {
        expect(viewLost([ev(T.VIEW_FOUND, 20), ev(T.VIEW_LOST, 10)])).toBe(false);
    });
});

describe('snap of the roll Repère', () => {
    test('a sure rest covering the instant gives its own mediaTime', () => {
        expect(snapRollTime([ev(T.DICE_REST, 30)], 33)).toBe(30);
    });
    test('the latest rest wins', () => {
        expect(snapRollTime([ev(T.DICE_REST, 30), ev(T.DICE_PICKUP, 35), ev(T.DICE_REST, 40)], 42)).toBe(40);
    });
    test('a rest to check or uncertain, or the latest one being so, leaves the current instant', () => {
        expect(snapRollTime([ev(T.DICE_REST, 30, S.CHECK)], 33)).toBeNull();
        expect(snapRollTime([ev(T.DICE_REST, 30, S.UNCERTAIN)], 33)).toBeNull();
        expect(snapRollTime([ev(T.DICE_REST, 30), ev(T.DICE_REST, 32, S.CHECK)], 33)).toBeNull();
    });
    test('a pick-up or a throw ends the window, the instant of the event excluded', () => {
        expect(snapRollTime([ev(T.DICE_REST, 30), ev(T.DICE_PICKUP, 35)], 34.9)).toBe(30);
        expect(snapRollTime([ev(T.DICE_REST, 30), ev(T.DICE_PICKUP, 35)], 35)).toBeNull();
        expect(snapRollTime([ev(T.DICE_REST, 30), ev(T.DICE_THROWN, 35)], 36)).toBeNull();
    });
    test('an event before the rest does not end its window', () => {
        expect(snapRollTime([ev(T.DICE_PICKUP, 25), ev(T.DICE_REST, 30)], 33)).toBe(30);
    });
    test('at most 60 s after the rest, the sixtieth second included', () => {
        expect(snapRollTime([ev(T.DICE_REST, 30)], 90)).toBe(30);
        expect(snapRollTime([ev(T.DICE_REST, 30)], 90.5)).toBeNull();
    });
    test('a rest ahead of the instant does not count; the rest instant itself does', () => {
        expect(snapRollTime([ev(T.DICE_REST, 30)], 29.9)).toBeNull();
        expect(snapRollTime([ev(T.DICE_REST, 30)], 30)).toBe(30);
    });
    test('a lost view leaves the current instant', () => {
        expect(snapRollTime([ev(T.DICE_REST, 30), ev(T.VIEW_LOST, 31)], 33)).toBeNull();
        expect(snapRollTime([ev(T.DICE_REST, 30), ev(T.VIEW_LOST, 31), ev(T.VIEW_FOUND, 32)], 33)).toBe(30);
    });
    test('no event, no snap', () => {
        expect(snapRollTime([], 33)).toBeNull();
    });
});

describe('overlay drift', () => {
    const change = [ev(T.OVERLAY_CHANGED, 100, S.CHECK)];
    test('reported from the instant of the change, whatever the Actions', () => {
        expect(recentOverlay(change, 99.9)).toBeNull();
        expect(recentOverlay(change, 100)?.mediaTime).toBe(100);
    });
    test('forgotten after 30 s', () => {
        expect(recentOverlay(change, 130)?.mediaTime).toBe(100);
        expect(recentOverlay(change, 130.1)).toBeNull();
    });
    test('the latest change is the one reported', () => {
        expect(recentOverlay([...change, ev(T.OVERLAY_CHANGED, 110)], 115)?.mediaTime).toBe(110);
    });
    test('other events say nothing', () => {
        expect(recentOverlay([ev(T.DICE_REST, 100)], 110)).toBeNull();
    });
});
