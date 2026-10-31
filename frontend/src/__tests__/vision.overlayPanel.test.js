/**
 * vision.overlayPanel.test.js — the overlay drift indicator, lit from the zone the user traces
 * to the badge: the zone drawn on the calibration reaches the core, a lasting change inside it
 * (and only inside it) becomes an `overlay.changed` event, and the panel's indicator shows it
 * for as long as ADR-0109 rule 7 says. The core runs for real; only the worker is replaced.
 */
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { must } from './helpers/must.js';
import VideoCalibration from '../components/VideoCalibration.svelte';
import { createVisionSession } from '../services/visionSession.js';
import { recentOverlay } from '../services/visionTimeline.js';
import { createWorkerCore } from '../vision/workerCore.js';
import { createMemoryJournal } from '../vision/journal.js';
import { EVENT_TYPES } from '../vision/contract.js';
import { makeFrame, texture, inside } from './fixtures/vision/frames.js';

// The pane is 400×200; the picture fills it.
const PANE = { w: 400, h: 200 };
const CORNERS = [
    [0.1, 0.75],
    [0.9, 0.75],
    [0.8, 0.15],
    [0.2, 0.15]
];
// Traced on the picture, in the 320×180 frame: x 80–240, y 90–135.
const ZONE_CLICKS = [
    [0.25, 0.5],
    [0.75, 0.75]
];
const ZONE_PIXELS = { x: 80, y: 90, w: 160, h: 45 };
// The badge stays up for 30 s after a change, on the stream's clock.
const SHOWN = 30;

beforeEach(() => {
    localStorage.clear();
    vi.spyOn(HTMLElement.prototype, 'offsetWidth', 'get').mockReturnValue(PANE.w);
    vi.spyOn(HTMLElement.prototype, 'offsetHeight', 'get').mockReturnValue(PANE.h);
    vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockReturnValue(
        /** @type {DOMRect} */ ({ left: 0, top: 0, width: PANE.w, height: PANE.h, right: PANE.w, bottom: PANE.h, x: 0, y: 0, toJSON() {} })
    );
});
afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
});

/** @param {any} el @param {number[]} f */
async function click(el, f) {
    const p = [f[0] * PANE.w, f[1] * PANE.h];
    await fireEvent.pointerMove(el, { clientX: p[0], clientY: p[1] });
    await fireEvent.pointerDown(el, { clientX: p[0], clientY: p[1], button: 0 });
}

/** The vision facade with the real worker core behind it, no Worker. */
function inProcessVision() {
    /** @type {Set<(e: any) => void>} */
    const listeners = new Set();
    const core = createWorkerCore({
        post: (m) => {
            if (m.kind === 'event') for (const l of listeners) l(m.event);
        },
        openJournal: async (o) => createMemoryJournal(o.session)
    });
    return {
        core,
        calibrate: vi.fn((calibration) => core.handle({ kind: 'calibrate', calibration })),
        start: vi.fn(({ session }) => core.handle({ kind: 'start', session })),
        stop: vi.fn(),
        clearJournal: vi.fn(),
        dispose: vi.fn(),
        subscribe: (/** @type {(e: any) => void} */ l) => (listeners.add(l), () => void listeners.delete(l))
    };
}

/** 2 frames a second; `region` is retextured from `changeAt` on. */
async function play(vision, region, changeAt, seconds) {
    for (let i = 0; i <= seconds * 2; i++) {
        const t = i / 2;
        const data = makeFrame((x, y) => (t >= changeAt && inside(region, x, y) ? texture(y, x, 1) : texture(x, y)));
        await vision.core.handle({ kind: 'frame', mediaTime: t, width: 320, height: 180, buffer: data.buffer });
    }
}

async function setup() {
    const video = Object.assign(new EventTarget(), { videoWidth: 400, videoHeight: 200, readyState: 4, seeking: false, currentTime: 0 });
    const vision = inProcessVision();
    /** @type {any[]} */
    const events = [];
    const frame = { width: 320, height: 180, data: new Uint8ClampedArray(320 * 180 * 4).fill(128) };
    const session = createVisionSession({
        video: /** @type {any} */ (video),
        key: 'd1',
        push: (e) => events.push(e),
        vision: /** @type {any} */ (vision),
        createGrabber: () => ({ grab: () => frame })
    });
    const r = render(VideoCalibration, { props: { calibrationKey: 'd1', video: /** @type {any} */ (video), oncalibration: (/** @type {any} */ c) => session.calibrate(c) } });
    await fireEvent.click(r.getByTestId('calibration-start'));
    for (const p of CORNERS) await click(r.getByTestId('calibration-capture'), p);
    return { r, vision, events, session };
}

describe('overlay drift, from the traced zone to the indicator', () => {
    test('a lasting change inside the traced zone lights the indicator for SHOWN seconds', async () => {
        const { r, vision, events } = await setup();
        await fireEvent.click(r.getByTestId('calibration-overlay'));
        for (const p of ZONE_CLICKS) await click(r.getByTestId('calibration-capture'), p);
        expect(must(vision.calibrate.mock.lastCall)[0].overlay).toEqual(ZONE_PIXELS);

        await play(vision, ZONE_PIXELS, 10, 30);
        const drift = events.filter((e) => e.type === EVENT_TYPES.OVERLAY_CHANGED);
        expect(drift).toHaveLength(1);
        expect(drift[0].mediaTime).toBe(10);
        expect(recentOverlay(events, 9.5)).toBeNull();
        expect(recentOverlay(events, 10)).not.toBeNull();
        expect(recentOverlay(events, 10 + SHOWN)).not.toBeNull();
        expect(recentOverlay(events, 10 + SHOWN + 0.5)).toBeNull();
    });

    test('a change outside the zone, or with no zone traced, lights nothing', async () => {
        const { r, vision, events } = await setup();
        await fireEvent.click(r.getByTestId('calibration-overlay'));
        for (const p of ZONE_CLICKS) await click(r.getByTestId('calibration-capture'), p);
        await play(vision, { x: 250, y: 140, w: 60, h: 30 }, 10, 30);
        expect(events.some((e) => e.type === EVENT_TYPES.OVERLAY_CHANGED)).toBe(false);

        await fireEvent.click(r.getByTestId('calibration-overlay-clear'));
        expect(must(vision.calibrate.mock.lastCall)[0]).not.toHaveProperty('overlay');
        await play(vision, ZONE_PIXELS, 10, 30);
        expect(events.some((e) => e.type === EVENT_TYPES.OVERLAY_CHANGED)).toBe(false);
    });
});
