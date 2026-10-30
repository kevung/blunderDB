import { describe, expect, it } from 'vitest';
import { COMMANDS, EVENT_TYPES, makeEvent, normalizeCalibration, STATES } from '../vision/contract.js';
import { createVision } from '../vision/vision.js';

describe('vision contract', () => {
    it('builds a frozen event with the four contract fields', () => {
        const ev = makeEvent(EVENT_TYPES.DICE_REST, 12.5, STATES.SURE, { side: 'left' });
        expect(ev).toEqual({ type: 'dice.rest', mediaTime: 12.5, state: 'sure', payload: { side: 'left' } });
        expect(Object.isFrozen(ev)).toBe(true);
        expect(Object.isFrozen(ev.payload)).toBe(true);
        expect(makeEvent(EVENT_TYPES.HANDS_IN, 0, STATES.CHECK).payload).toEqual({});
    });

    it('refuses what the contract does not name', () => {
        expect(() => makeEvent('dice.guess', 1, STATES.SURE)).toThrow(TypeError);
        expect(() => makeEvent(EVENT_TYPES.DICE_REST, -1, STATES.SURE)).toThrow(RangeError);
        expect(() => makeEvent(EVENT_TYPES.DICE_REST, NaN, STATES.SURE)).toThrow(RangeError);
        expect(() => makeEvent(EVENT_TYPES.DICE_REST, 1, '0.93')).toThrow(TypeError);
        expect(() => makeEvent(EVENT_TYPES.DICE_REST, 1, STATES.SURE, null)).toThrow(TypeError);
    });

    it('freezes three states, never a percentage', () => {
        expect(Object.values(STATES)).toEqual(['sure', 'check', 'uncertain']);
        expect(Object.isFrozen(STATES) && Object.isFrozen(EVENT_TYPES) && Object.isFrozen(COMMANDS)).toBe(true);
    });

    it('exposes every command as a method of the facade', () => {
        const worker = { postMessage() {}, terminate() {}, onmessage: null, onerror: null };
        const vision = createVision({ video: /** @type {any} */ ({ currentTime: 0 }), createWorker: () => worker, requestFrame: () => 1, cancelFrame: () => {} });
        for (const name of Object.values(COMMANDS)) expect(typeof vision[name]).toBe('function');
        expect(typeof vision.subscribe).toBe('function');
    });

    it('normalises a calibration and refuses an ROI outside the frame', () => {
        const cal = normalizeCalibration({ rois: [{ name: 'd', x: 0, y: 0, w: 10, h: 10 }], witness: { x: 300, y: 0, w: 20, h: 10 } });
        expect(cal.width).toBe(320);
        expect(cal.height).toBe(180);
        expect(() => normalizeCalibration({ rois: [{ name: 'd', x: 315, y: 0, w: 10, h: 10 }], witness: { x: 0, y: 0, w: 5, h: 5 } })).toThrow(RangeError);
        expect(() => normalizeCalibration({ rois: [{ name: 'd', x: 0, y: 0, w: 2, h: 10 }], witness: { x: 0, y: 0, w: 5, h: 5 } })).toThrow(RangeError);
        expect(() =>
            normalizeCalibration({
                rois: [
                    { name: 'd', x: 0, y: 0, w: 5, h: 5 },
                    { name: 'd', x: 9, y: 9, w: 5, h: 5 }
                ],
                witness: { x: 0, y: 0, w: 5, h: 5 }
            })
        ).toThrow(TypeError);
        expect(() => normalizeCalibration({ rois: [{ name: 'd', x: 0.5, y: 0, w: 5, h: 5 }], witness: { x: 0, y: 0, w: 5, h: 5 } })).toThrow(TypeError);
    });
});
