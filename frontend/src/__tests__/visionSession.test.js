/**
 * visionSession.test.js — the panel's side of the vision core: the calibration in force
 * becomes the core's ROIs, the board's presence is judged at 2 Hz against a thumbnail taken at
 * calibration, and the analysis keeps silent while the board is out of the picture
 * (ADR-0109 rule 3).
 */
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { calibrateBoard } from '../vision/geometry/index.js';
import { OBLIQUE_CLICKS, POSITIONS, renderBoard, stackCheckers } from '../vision/geometry/synthetic.fixture.js';
import { EVENT_TYPES as T } from '../vision/contract.js';
import { createVisionSession } from '../services/visionSession.js';
import { viewLost } from '../services/visionTimeline.js';

// Twice a second, stated here in the literal.
const PRESENCE_PERIOD_MS = 500;
// The view is lost after four seconds of low scores: eight readings.
const HOLD = 8 * PRESENCE_PERIOD_MS;

const CLICKS = /** @type {[number, number][]} */ ([
    [0.9, 0.9],
    [0.1, 0.9],
    [0.1, 0.1],
    [0.9, 0.1]
]);
const cal = () => {
    const r = /** @type {any} */ (calibrateBoard(CLICKS));
    return { name: 'plein écran', clicks: CLICKS, modelToImage: r.modelToImage };
};

/**
 * A frame of noise: two seeds are two unrelated boards.
 * @param {number} seed
 */
function noise(seed) {
    const data = new Uint8ClampedArray(320 * 180 * 4);
    let s = seed;
    for (let i = 0; i < data.length; i += 4) {
        s = (s * 1664525 + 1013904223) >>> 0;
        data[i] = data[i + 1] = data[i + 2] = s >>> 24;
        data[i + 3] = 255;
    }
    return { width: 320, height: 180, data };
}

function setup({ key = 'd1', videoWidth = 1920 } = {}) {
    const video = Object.assign(new EventTarget(), { videoWidth, videoHeight: 1080, readyState: 4, seeking: false, currentTime: 12 });
    /** @type {Array<(e: any) => void>} */
    const listeners = [];
    const vision = {
        calibrate: vi.fn(),
        start: vi.fn(),
        stop: vi.fn(),
        clearJournal: vi.fn(),
        dispose: vi.fn(),
        subscribe: vi.fn((l) => (listeners.push(l), () => {}))
    };
    const frame = { current: noise(1) };
    const grab = vi.fn(() => frame.current);
    /** @type {any[]} */
    const pushed = [];
    const session = createVisionSession({ video: /** @type {any} */ (video), key, push: (e) => pushed.push(e), vision: /** @type {any} */ (vision), createGrabber: () => ({ grab }) });
    return { video, vision, frame, grab, pushed, session, listeners };
}

beforeEach(() => {
    vi.useFakeTimers();
    localStorage.clear();
});
afterEach(() => vi.useRealTimers());

describe('calibration to the core', () => {
    test('the calibration in force becomes left and right ROIs and a witness, and the core starts on a session named after the draft and the calibration', () => {
        const { session, vision } = setup();
        session.calibrate(cal());
        const arg = vision.calibrate.mock.calls[0][0];
        expect(arg.rois.map((/** @type {any} */ r) => r.name)).toEqual(['left', 'right']);
        expect(arg.witness.w).toBeGreaterThan(0);
        expect(vision.start).toHaveBeenCalledWith({ session: 'd1:plein écran' });
    });

    test('the overlay zone goes to the core in pixels of the reduced frame; without one, or under the hash grid, none goes', () => {
        const { session, vision } = setup();
        session.calibrate({ ...cal(), overlay: { x: 0.25, y: 0.5, w: 0.5, h: 0.25 } });
        expect(vision.calibrate.mock.calls[0][0].overlay).toEqual({ x: 80, y: 90, w: 160, h: 45 });
        session.calibrate(cal());
        expect(vision.calibrate.mock.calls[1][0]).not.toHaveProperty('overlay');
        session.calibrate({ ...cal(), overlay: { x: 0.5, y: 0.5, w: 8 / 320, h: 0.25 } });
        expect(vision.calibrate.mock.calls[2][0]).not.toHaveProperty('overlay');
    });

    test('without the size of the picture nothing is handed over until its metadata is known', () => {
        const { session, vision, video } = setup({ videoWidth: 0 });
        session.calibrate(cal());
        expect(vision.calibrate).not.toHaveBeenCalled();
        video.videoWidth = 1920;
        video.dispatchEvent(new Event('loadedmetadata'));
        expect(vision.calibrate).toHaveBeenCalledTimes(1);
        expect(vision.start).toHaveBeenCalledTimes(1);
    });

    test('no calibration stops the core and the presence reading', () => {
        const { session, vision, grab } = setup();
        session.calibrate(cal());
        session.calibrate(null);
        expect(vision.stop).toHaveBeenCalled();
        vi.advanceTimersByTime(5 * PRESENCE_PERIOD_MS);
        expect(grab).not.toHaveBeenCalled();
    });

    test('the core events reach the panel', () => {
        const { listeners, pushed } = setup();
        listeners[0]({ type: T.DICE_REST });
        expect(pushed).toHaveLength(1);
    });
});

describe('presence of the board', () => {
    test('read twice a second: nothing before 500 ms, a first reading then', () => {
        const { session, grab } = setup();
        session.calibrate(cal());
        vi.advanceTimersByTime(PRESENCE_PERIOD_MS - 1);
        expect(grab).not.toHaveBeenCalled();
        vi.advanceTimersByTime(1);
        expect(grab).toHaveBeenCalledTimes(1);
    });

    test('the same board stays present; another picture is "view lost", the core is silenced, and the board back is "view found"', () => {
        const { session, vision, frame, pushed, video } = setup();
        session.calibrate(cal());
        vi.advanceTimersByTime(PRESENCE_PERIOD_MS); // the reference
        vi.advanceTimersByTime(PRESENCE_PERIOD_MS);
        expect(pushed).toEqual([]);
        vision.stop.mockClear();
        frame.current = noise(99);
        video.currentTime = 40;
        vi.advanceTimersByTime(HOLD - PRESENCE_PERIOD_MS);
        expect(pushed).toEqual([]);
        vi.advanceTimersByTime(PRESENCE_PERIOD_MS);
        expect(pushed.map((e) => e.type)).toEqual([T.VIEW_LOST]);
        expect(pushed[0].mediaTime).toBe(40);
        expect(vision.stop).toHaveBeenCalledTimes(1);
        vision.start.mockClear();
        frame.current = noise(1);
        vi.advanceTimersByTime(PRESENCE_PERIOD_MS);
        expect(pushed.map((e) => e.type)).toEqual([T.VIEW_LOST, T.VIEW_FOUND]);
        expect(vision.start).toHaveBeenCalledTimes(1);
    });

    test('a lost view is announced once, not at every reading', () => {
        const { session, frame, pushed } = setup();
        session.calibrate(cal());
        vi.advanceTimersByTime(PRESENCE_PERIOD_MS);
        frame.current = noise(99);
        vi.advanceTimersByTime(3 * HOLD);
        expect(pushed).toHaveLength(1);
    });

    test('no reading while the player seeks or has no picture', () => {
        const { session, video, grab } = setup();
        session.calibrate(cal());
        video.seeking = true;
        vi.advanceTimersByTime(PRESENCE_PERIOD_MS);
        video.seeking = false;
        video.readyState = 1;
        vi.advanceTimersByTime(PRESENCE_PERIOD_MS);
        expect(grab).not.toHaveBeenCalled();
    });

    test('an unreadable frame says nothing', () => {
        const { session, grab, pushed } = setup();
        grab.mockImplementation(() => {
            throw new Error('tainted');
        });
        session.calibrate(cal());
        vi.advanceTimersByTime(3 * PRESENCE_PERIOD_MS);
        expect(pushed).toEqual([]);
    });

    test('the reference thumbnail is kept with the draft: a session opened later judges from its first reading', () => {
        const first = setup();
        first.session.calibrate(cal());
        vi.advanceTimersByTime(PRESENCE_PERIOD_MS);
        first.session.dispose();
        const second = setup();
        second.frame.current = noise(99);
        second.session.calibrate(cal());
        vi.advanceTimersByTime(HOLD);
        expect(second.pushed.map((e) => e.type)).toEqual([T.VIEW_LOST]);
    });

    test('each calibration of a draft keeps its own reference', () => {
        const first = setup();
        first.session.calibrate(cal());
        vi.advanceTimersByTime(PRESENCE_PERIOD_MS);
        const other = setup();
        other.frame.current = noise(99);
        other.session.calibrate({ ...cal(), name: 'incrustation' });
        vi.advanceTimersByTime(HOLD);
        expect(other.pushed).toEqual([]);
        const back = setup();
        back.frame.current = noise(99);
        back.session.calibrate(cal());
        vi.advanceTimersByTime(HOLD);
        expect(back.pushed.map((e) => e.type)).toEqual([T.VIEW_LOST]);
    });

    test('another draft, or other clicks, do not inherit the reference', () => {
        const first = setup();
        first.session.calibrate(cal());
        vi.advanceTimersByTime(PRESENCE_PERIOD_MS);
        const other = setup({ key: 'd2' });
        other.frame.current = noise(99);
        other.session.calibrate(cal());
        vi.advanceTimersByTime(HOLD);
        expect(other.pushed).toEqual([]);
        const moved = setup();
        moved.frame.current = noise(99);
        moved.session.calibrate({ ...cal(), clicks: [[0.8, 0.9], ...CLICKS.slice(1)] });
        vi.advanceTimersByTime(HOLD);
        expect(moved.pushed).toEqual([]);
    });
});

describe('presence on a board in play', () => {
    // The synthetic board seen at an angle, its points close in tone as on felt.
    const FELT = /** @type {[number, number]} */ ([100, 130]);
    const OBLIQUE = /** @type {[number, number][]} */ (OBLIQUE_CLICKS.map(([x, y]) => [x / 320, y / 180]));
    const oblique = () => ({ name: 'plein écran', clicks: OBLIQUE, modelToImage: /** @type {any} */ (calibrateBoard(OBLIQUE)).modelToImage });
    /** @param {object} [o] */
    const board = (o = {}) =>
        /** @type {any} */ (renderBoard({ width: 320, height: 180, clicks: OBLIQUE_CLICKS, points: FELT, background: 150, rgba: true, checkers: stackCheckers(POSITIONS.start), ...o }));

    test('checkers moving from the start to a bear-off never lose the view', () => {
        const { session, frame, pushed, vision } = setup();
        frame.current = board();
        session.calibrate(oblique());
        vi.advanceTimersByTime(PRESENCE_PERIOD_MS); // the reference, at the start
        for (const p of /** @type {const} */ (['start', 'middle', 'bearoff'])) {
            frame.current = board({ checkers: stackCheckers(POSITIONS[p]) });
            vi.advanceTimersByTime(2 * HOLD);
        }
        expect(pushed).toEqual([]);
        expect(vision.stop).not.toHaveBeenCalled();
    });

    test('an arm across the board for two seconds does not lose the view', () => {
        const { session, frame, pushed, vision } = setup();
        frame.current = board();
        session.calibrate(oblique());
        vi.advanceTimersByTime(PRESENCE_PERIOD_MS);
        frame.current = board({ occluder: { x: 40, y: 0, w: 240, h: 180, luma: 160 } });
        vi.advanceTimersByTime(4 * PRESENCE_PERIOD_MS);
        frame.current = board({ checkers: stackCheckers(POSITIONS.middle) });
        vi.advanceTimersByTime(HOLD);
        expect(pushed).toEqual([]);
        expect(vision.stop).not.toHaveBeenCalled();
    });

    test('another shot, or a camera moved by a few pixels, loses the view', () => {
        /** @type {[number, number][][]} */
        const shots = [
            OBLIQUE_CLICKS.map(([x, y]) => [x + 10, y - 5]),
            [
                [300, 175],
                [-200, 190],
                [-120, -150],
                [280, -160]
            ]
        ];
        for (const clicks of shots) {
            const { session, frame, pushed, vision } = setup();
            frame.current = board();
            session.calibrate(oblique());
            vi.advanceTimersByTime(PRESENCE_PERIOD_MS);
            frame.current = board({ clicks });
            vi.advanceTimersByTime(HOLD);
            expect(pushed.map((e) => e.type)).toEqual([T.VIEW_LOST]);
            expect(vision.stop).toHaveBeenCalledTimes(1);
            session.dispose();
        }
    });

    test('a view lost, then a new calibration: the view is found again and the analysis restarts', () => {
        const { session, frame, pushed, vision, video } = setup();
        frame.current = board();
        session.calibrate(oblique());
        vi.advanceTimersByTime(PRESENCE_PERIOD_MS);
        const moved = /** @type {[number, number][]} */ (OBLIQUE_CLICKS.map(([x, y]) => [x + 10, y - 5]));
        frame.current = board({ clicks: moved });
        vi.advanceTimersByTime(HOLD);
        expect(pushed.map((e) => e.type)).toEqual([T.VIEW_LOST]);
        vision.start.mockClear();
        video.currentTime = 60;
        const clicks = /** @type {[number, number][]} */ (moved.map(([x, y]) => [x / 320, y / 180]));
        session.calibrate({ name: 'déplacée', clicks, modelToImage: /** @type {any} */ (calibrateBoard(clicks)).modelToImage });
        expect(pushed.map((e) => e.type)).toEqual([T.VIEW_LOST, T.VIEW_FOUND]);
        expect(pushed[1].mediaTime).toBe(60);
        expect(viewLost(pushed)).toBe(false);
        expect(vision.start).toHaveBeenCalledWith({ session: 'd1:déplacée' });
        vi.advanceTimersByTime(3 * HOLD);
        expect(pushed).toHaveLength(2);
    });
});
