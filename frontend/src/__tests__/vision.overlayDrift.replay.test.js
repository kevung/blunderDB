import { describe, expect, it } from 'vitest';
import { createDescriber, createDetector, replay } from '../vision/descriptors.js';
import { createMemoryJournal } from '../vision/journal.js';
import { createWorkerCore } from '../vision/workerCore.js';
import { normalizeCalibration, EVENT_TYPES, STATES, makeEvent } from '../vision/contract.js';
import { createOverlayDrift, hashToHex, hexToHash } from '../vision/overlay.js';
import { CALIBRATION, scene } from './fixtures/vision/frames.js';

const OVERLAY = { x: 100, y: 140, w: 120, h: 30 };
const CAL = { ...CALIBRATION, overlay: OVERLAY };

// The conventions of ADR-0109 rule 7, written out: a test that imported them would follow any change.
const SAMPLE_PERIOD = 0.5;
const LATE_SECONDS = 4;
const EARLY_SECONDS = 1;
const DECISION_SECONDS = 10;

const A = 'ff00ff00ff00ff00';
const B = '00ff00ff00ff00ff';

/** Feeds `seconds` of 2 img/s samples, the overlay going from A to B at `changeAt`; `events` by instant. */
function run(changeAt, events = {}, seconds = 30) {
    const drift = createOverlayDrift();
    const out = [];
    for (let i = 0; i <= seconds * 2; i++) {
        const t = i / 2;
        out.push(...drift.push(t, t >= changeAt ? B : A, events[t] ?? []));
    }
    return out;
}
const ev = (type, t) => makeEvent(type, t, STATES.CHECK);

describe('overlay sampling by the describer', () => {
    it('adds no hash without an overlay in the calibration', () => {
        const d = createDescriber(CALIBRATION);
        expect(d.describe({ mediaTime: 0, data: scene() }).o).toBeUndefined();
    });

    it('hashes at most every SAMPLE_PERIOD seconds and samples again after a seek back', () => {
        const d = createDescriber(CAL);
        const has = (t) => d.describe({ mediaTime: t, data: scene() }).o !== undefined;
        expect(has(0)).toBe(true);
        expect(has(SAMPLE_PERIOD - 0.01)).toBe(false);
        expect(has(SAMPLE_PERIOD)).toBe(true);
        expect(has(SAMPLE_PERIOD + 0.1)).toBe(false);
        expect(has(0)).toBe(true);
    });

    it('does not hash a frame whose picture jumped', () => {
        const d = createDescriber(CAL);
        d.describe({ mediaTime: 0, data: scene() });
        const r = d.describe({ mediaTime: 1, data: scene([CALIBRATION.witness], 5) });
        expect(r.u).toBe(1);
        expect(r.o).toBeUndefined();
    });

    it('rejects an overlay under the 9×8 grid', () => {
        expect(() => normalizeCalibration({ ...CALIBRATION, overlay: { x: 100, y: 100, w: 8, h: 20 } })).toThrow(RangeError);
        expect(() => normalizeCalibration({ ...CALIBRATION, overlay: { x: 100, y: 100, w: 20, h: 7 } })).toThrow(RangeError);
        expect(normalizeCalibration({ ...CALIBRATION, overlay: { x: 100, y: 100, w: 9, h: 8 } }).overlay).toMatchObject({ w: 9, h: 8 });
    });

    it('round-trips the hash through hexadecimal', () => {
        expect(hashToHex(new Uint8Array([1, 0xab, 0]))).toBe('01ab00');
        expect(hexToHash('01ab00')).toEqual(new Uint8Array([1, 0xab, 0]));
        expect(() => hexToHash('zz')).toThrow(TypeError);
    });
});

describe('unexpected / expected overlay change', () => {
    it('reports an unexpected change once, dated at its start, after the decision delay', () => {
        const out = run(5);
        expect(out).toHaveLength(1);
        expect(out[0]).toMatchObject({ type: EVENT_TYPES.OVERLAY_CHANGED, mediaTime: 5, payload: { distance: 64 } });
    });

    it('holds the verdict until DECISION_SECONDS after the start of the change', () => {
        const drift = createOverlayDrift();
        let first = null;
        for (let i = 0; i <= 60; i++) {
            const t = i / 2;
            if (drift.push(t, t >= 5 ? B : A, []).length) {
                first = t;
                break;
            }
        }
        expect(first).toBe(5 + DECISION_SECONDS);
    });

    it('says nothing when the change follows an expected action within LATE_SECONDS', () => {
        for (const type of [EVENT_TYPES.GAME_END, EVENT_TYPES.GAME_START, EVENT_TYPES.DICE_PICKUP]) {
            expect(run(5, { 5: [ev(type, 5 - LATE_SECONDS)] })).toEqual([]);
        }
    });

    it('reports a change later than LATE_SECONDS after the action', () => {
        expect(run(5, { 5: [ev(EVENT_TYPES.GAME_END, 5 - LATE_SECONDS - 0.5)] })).toHaveLength(1);
    });

    it('tolerates the action being dated EARLY_SECONDS after the change, no more', () => {
        expect(run(5, { 5: [ev(EVENT_TYPES.GAME_END, 5 + EARLY_SECONDS)] })).toEqual([]);
        expect(run(5, { 5: [ev(EVENT_TYPES.GAME_END, 5 + EARLY_SECONDS + 0.5)] })).toHaveLength(1);
    });

    it('accepts an expected action emitted late but dated before the change', () => {
        expect(run(5, { 13: [ev(EVENT_TYPES.GAME_END, 4)] })).toEqual([]);
    });

    it('ignores actions that are not expected ones', () => {
        for (const type of [EVENT_TYPES.DICE_REST, EVENT_TYPES.DICE_THROWN, EVENT_TYPES.HANDS_IN]) {
            expect(run(5, { 5: [ev(type, 5)] })).toHaveLength(1);
        }
    });

    it('stays silent with no hash at all', () => {
        const drift = createOverlayDrift();
        for (let t = 0; t < 40; t++) expect(drift.push(t, undefined, [])).toEqual([]);
    });

    it('does not report a change that does not last', () => {
        const drift = createOverlayDrift();
        const out = [];
        for (let i = 0; i <= 60; i++) {
            const t = i / 2;
            out.push(...drift.push(t, t >= 5 && t < 5.5 ? B : A, []));
        }
        expect(out).toEqual([]);
    });
});

describe('overlay drift end to end', () => {
    const frames = () =>
        Array.from({ length: 40 }, (_, i) => {
            const t = i / 2;
            return { mediaTime: t, data: scene(t >= 3 ? [OVERLAY] : [], 1) };
        });

    it('replays a journal of descriptor records to one overlay.changed', () => {
        const d = createDescriber(CAL);
        const records = frames().map((f) => d.describe(f));
        const events = replay(records, createDetector());
        const drift = events.filter((e) => e.type === EVENT_TYPES.OVERLAY_CHANGED);
        expect(drift).toHaveLength(1);
        expect(drift[0].mediaTime).toBe(3);
        expect(drift[0].payload.distance).toBeGreaterThan(10);
        // The same records through a JSON journal give the same events.
        expect(replay(JSON.parse(JSON.stringify(records)), createDetector())).toEqual(events);
    });

    it('goes out of the worker as an event, and never touches anything but the indicator', async () => {
        const posted = [];
        const core = createWorkerCore({ post: (m) => posted.push(m), openJournal: async (o) => createMemoryJournal(o.session) });
        await core.handle({ kind: 'calibrate', calibration: CAL });
        await core.handle({ kind: 'start', session: 's' });
        for (const f of frames()) await core.handle({ kind: 'frame', mediaTime: f.mediaTime, width: 320, height: 180, buffer: f.data.buffer });
        const kinds = posted.filter((m) => m.kind === 'event').map((m) => m.event.type);
        expect(kinds).toEqual([EVENT_TYPES.OVERLAY_CHANGED]);
    });

    it('is silent when the calibration has no overlay', () => {
        const d = createDescriber(CALIBRATION);
        const events = replay(
            frames().map((f) => d.describe(f)),
            createDetector()
        );
        expect(events.some((e) => e.type === EVENT_TYPES.OVERLAY_CHANGED)).toBe(false);
    });
});
