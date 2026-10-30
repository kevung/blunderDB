import { describe, expect, it } from 'vitest';
import { createFrameClock } from '../vision/clock.js';
import { createFrameGrabber, createRateGate, TaintedFrameError } from '../vision/sampler.js';

/** A manual rAF: `tick()` runs the pending callback, as a composited frame would. */
function manualRaf() {
    /** @type {Map<number, FrameRequestCallback>} */
    const pending = new Map();
    let next = 0;
    return {
        requestFrame: (/** @type {FrameRequestCallback} */ cb) => (pending.set(++next, cb), next),
        cancelFrame: (/** @type {number} */ id) => void pending.delete(id),
        tick() {
            const cbs = [...pending.values()];
            pending.clear();
            for (const cb of cbs) cb(0);
        },
        get pending() {
            return pending.size;
        }
    };
}

/** A video whose rVFC exists; `callbacks` holds what was requested, never called unless a test does. */
function video() {
    /** @type {Map<number, Function>} */
    const callbacks = new Map();
    let id = 0;
    return {
        currentTime: 0,
        callbacks,
        requestVideoFrameCallback: (/** @type {Function} */ cb) => (callbacks.set(++id, cb), id),
        cancelVideoFrameCallback: (/** @type {number} */ i) => void callbacks.delete(i),
        /** @param {number} mediaTime */
        present(mediaTime) {
            const cbs = [...callbacks.values()];
            callbacks.clear();
            for (const cb of cbs) cb(0, { mediaTime });
        }
    };
}

describe('frame clock', () => {
    it('falls back to rAF when rVFC exists but never calls back, one frame per currentTime change', () => {
        const raf = manualRaf();
        const v = video();
        /** @type {number[]} */
        const frames = [];
        /** @type {string[]} */
        const sources = [];
        const clock = createFrameClock({ video: /** @type {any} */ (v), onFrame: (t) => frames.push(t), onSource: (s) => sources.push(s), ...raf });
        clock.start();
        v.currentTime = 1 / 30;
        raf.tick();
        raf.tick(); // same composited frame twice: retained once
        v.currentTime = 2 / 30;
        raf.tick();
        expect(frames).toEqual([1 / 30, 2 / 30]);
        expect(sources).toEqual(['raf']);
        expect(clock.source).toBe('raf');
        expect(v.callbacks.size).toBe(1); // the probe stays armed
    });

    it('switches to rVFC mediaTime on the first callback and stops the rAF loop', () => {
        const raf = manualRaf();
        const v = video();
        /** @type {number[]} */
        const frames = [];
        /** @type {string[]} */
        const sources = [];
        const clock = createFrameClock({ video: /** @type {any} */ (v), onFrame: (t) => frames.push(t), onSource: (s) => sources.push(s), ...raf });
        clock.start();
        v.currentTime = 0.5;
        raf.tick();
        v.present(0.5333);
        v.currentTime = 0.6;
        raf.tick();
        v.present(0.5667);
        expect(frames).toEqual([0.5, 0.5333, 0.5667]);
        expect(sources).toEqual(['raf', 'rvfc']);
        expect(raf.pending).toBe(0);
    });

    it('works without rVFC at all, and stop cancels everything', () => {
        const raf = manualRaf();
        const v = { currentTime: 0 };
        /** @type {number[]} */
        const frames = [];
        const clock = createFrameClock({ video: /** @type {any} */ (v), onFrame: (t) => frames.push(t), ...raf });
        clock.start();
        v.currentTime = 0.1;
        raf.tick();
        clock.stop();
        v.currentTime = 0.2;
        raf.tick();
        expect(frames).toEqual([0.1]);
        expect(raf.pending).toBe(0);
    });
});

describe('rate gate', () => {
    it('keeps one frame per 1/fps slot of media time', () => {
        const gate = createRateGate(10);
        const kept = [];
        for (let i = 0; i < 90; i++) if (gate.accept(i / 30)) kept.push(i);
        expect(kept.length).toBe(30);
        expect(kept.slice(0, 4)).toEqual([0, 3, 6, 9]);
    });

    it('keeps the same media density in accelerated playback, and reopens after a backward seek', () => {
        const gate = createRateGate(2);
        // At 4× the clock sees about one media frame in three: still two kept per media second.
        let kept = 0;
        for (let i = 0; i < 300; i += 3) if (gate.accept(i / 30)) kept++;
        expect(kept).toBe(20);
        expect(gate.accept(1.0)).toBe(true);
    });

    it('keeps everything at fps 0', () => {
        const gate = createRateGate(0);
        expect([0, 0.01, 0.02].every((t) => gate.accept(t))).toBe(true);
    });
});

describe('frame grabber', () => {
    /** @param {() => ImageData} getImageData */
    function canvas(getImageData) {
        /** @type {any[]} */
        const ctxArgs = [];
        return {
            ctxArgs,
            create: () => ({ getContext: (/** @type {any[]} */ ...args) => (ctxArgs.push(args), { drawImage() {}, getImageData }) })
        };
    }

    it('reads 320×180 with willReadFrequently', () => {
        const c = canvas(() => /** @type {any} */ ({ width: 320, height: 180, data: new Uint8ClampedArray(320 * 180 * 4) }));
        const grabber = createFrameGrabber({ video: /** @type {any} */ ({}), createCanvas: c.create });
        expect(grabber.grab().data.length).toBe(320 * 180 * 4);
        expect(c.ctxArgs[0]).toEqual(['2d', { willReadFrequently: true }]);
    });

    it('turns a SecurityError into TaintedFrameError', () => {
        const c = canvas(() => {
            throw Object.assign(new Error('tainted'), { name: 'SecurityError' });
        });
        const grabber = createFrameGrabber({ video: /** @type {any} */ ({}), createCanvas: c.create });
        expect(() => grabber.grab()).toThrow(TaintedFrameError);
    });
});
