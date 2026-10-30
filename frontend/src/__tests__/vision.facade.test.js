import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { createVision } from '../vision/vision.js';
import { TaintedFrameError } from '../vision/sampler.js';
import { CALIBRATION } from './fixtures/vision/frames.js';

function harness({ grab } = {}) {
    /** @type {any[]} */
    const sent = [];
    const worker = {
        onmessage: /** @type {any} */ (null),
        onerror: /** @type {any} */ (null),
        terminated: false,
        postMessage: (/** @type {any} */ m, /** @type {any[]} */ transfer) => sent.push({ m, transfer }),
        terminate() {
            this.terminated = true;
        },
        reply: (/** @type {any} */ data) => worker.onmessage({ data })
    };
    /** @type {FrameRequestCallback | null} */
    let pending = null;
    const video = { currentTime: 0 };
    const grabs = { n: 0 };
    const vision = createVision({
        video: /** @type {any} */ (video),
        createWorker: () => worker,
        createGrabber: () => ({
            grab: grab ?? (() => (grabs.n++, /** @type {any} */ ({ width: 320, height: 180, data: new Uint8ClampedArray(320 * 180 * 4) })))
        }),
        requestFrame: (cb) => ((pending = cb), 1),
        cancelFrame: () => (pending = null)
    });
    /** @type {any[]} */
    const events = [];
    vision.subscribe((e) => events.push(e));
    /** @param {number} t */
    const frame = (t) => {
        video.currentTime = t;
        const cb = pending;
        pending = null;
        cb?.(0);
    };
    return { vision, worker, sent, events, frame, grabs, frames: () => sent.filter((s) => s.m.kind === 'frame') };
}

describe('vision facade', () => {
    it('loads the worker as an ES module through new URL(import.meta.url), the path the V0.1 bench validated', () => {
        const src = readFileSync(join(__dirname, '..', 'vision', 'vision.js'), 'utf8');
        expect(src).toContain("new Worker(new URL('./worker.js', import.meta.url), { type: 'module' })");
        // Code only, comments stripped: they name the traps on purpose.
        const code = src.replace(/\/\*[\s\S]*?\*\//g, '').replace(/\/\/.*$/gm, '');
        expect(code).not.toMatch(/\.play\(|isConfigSupported|VideoDecoder/);
    });

    it('samples at the worker rate, transfers the buffer, and keeps one frame in flight', () => {
        const h = harness();
        h.vision.calibrate(CALIBRATION);
        h.vision.start({ session: 'draft/plein' });
        expect(h.sent.map((s) => s.m.kind)).toEqual(['calibrate', 'start']);
        h.frame(0.1);
        expect(h.events.map((e) => [e.type, e.payload.source])).toEqual([['vision.clock', 'raf']]);
        const [first] = h.frames();
        expect(first.m.mediaTime).toBe(0.1);
        expect(first.transfer).toEqual([first.m.buffer]);
        h.frame(0.6); // previous frame still in flight: skipped
        expect(h.frames().length).toBe(1);
        h.worker.reply({ kind: 'frameDone' });
        h.frame(0.7); // 0.6 was skipped before the gate, so 0.7 still opens slot 1
        expect(h.frames().length).toBe(2);
        h.worker.reply({ kind: 'frameDone' });
        h.frame(0.8); // slot 1 already taken at 2 img/s
        expect(h.frames().length).toBe(2);
        h.worker.reply({ kind: 'rate', fps: 10 });
        h.frame(0.9);
        expect(h.frames().length).toBe(3);
    });

    it('re-validates and forwards worker events as frozen contract events', () => {
        const h = harness();
        h.worker.reply({ kind: 'event', event: { type: 'frame.uncertain', mediaTime: 3, state: 'uncertain', payload: { witness: 0.9 } } });
        expect(h.events).toEqual([{ type: 'frame.uncertain', mediaTime: 3, state: 'uncertain', payload: { witness: 0.9 } }]);
        expect(Object.isFrozen(h.events[0])).toBe(true);
        expect(() => h.worker.reply({ kind: 'event', event: { type: 'nope', mediaTime: 3, state: 'sure', payload: {} } })).toThrow();
    });

    it('stops and reports tainted when the canvas becomes unreadable', () => {
        const h = harness({
            grab: () => {
                throw new TaintedFrameError();
            }
        });
        h.vision.calibrate(CALIBRATION);
        h.vision.start({ session: 's' });
        h.frame(1);
        expect(h.events.at(-1)).toMatchObject({ type: 'vision.error', state: 'uncertain', payload: { reason: 'tainted' } });
        h.frame(2);
        expect(h.frames().length).toBe(0);
        expect(h.sent.at(-1).m).toEqual({ kind: 'stop' });
    });

    it('starts again after a tainted halt, without a stop() in between', () => {
        let tainted = true;
        const h = harness({
            grab: () => {
                if (tainted) throw new TaintedFrameError();
                return /** @type {any} */ ({ width: 320, height: 180, data: new Uint8ClampedArray(320 * 180 * 4) });
            }
        });
        h.vision.calibrate(CALIBRATION);
        h.vision.start({ session: 's' });
        h.frame(1);
        tainted = false;
        h.vision.start({ session: 's' });
        expect(h.sent.map((s) => s.m.kind)).toEqual(['calibrate', 'start', 'stop', 'start']);
        h.frame(2);
        expect(h.frames().length).toBe(1);
    });

    it('resolves clearJournal on the worker acknowledgement, and dispose terminates', async () => {
        const h = harness();
        const p = h.vision.clearJournal('s');
        const msg = h.sent.at(-1).m;
        expect(msg).toMatchObject({ kind: 'clearJournal', session: 's' });
        h.worker.reply({ kind: 'journalCleared', id: msg.id });
        await expect(p).resolves.toBeUndefined();
        const q = h.vision.clearJournal();
        h.vision.dispose();
        await expect(q).rejects.toThrow('disposed');
        expect(h.worker.terminated).toBe(true);
    });

    it('needs a calibration before any frame and a session to start', () => {
        const h = harness();
        expect(() => h.vision.start(/** @type {any} */ ({}))).toThrow(TypeError);
        h.vision.start({ session: 's' });
        h.frame(1);
        expect(h.frames().length).toBe(0);
    });
});
