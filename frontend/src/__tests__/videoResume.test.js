/**
 * videoResume.test.js — where a draft's video resumes: the instant kept for the draft, else
 * a little before the last Repère, and storage that fails is no failure.
 */

import { describe, test, expect, beforeEach, afterEach, vi } from 'vitest';
import { saveVideoResume, loadVideoResume, lastRepere, videoResumeMs, RESUME_LEAD_MS } from '../services/videoResume.js';

describe('videoResume', () => {
    beforeEach(() => {
        localStorage.clear();
    });
    afterEach(() => vi.unstubAllGlobals());

    test('the instant kept for a draft wins, per draft', () => {
        saveVideoResume(7, 123456.4);
        expect(loadVideoResume(7)).toBe(123456);
        expect(loadVideoResume(8)).toBeNull();
        expect(videoResumeMs(7, [{ kind: 'checker', tick_ms: 900000 }])).toBe(123456);
    });

    test('without one, a little before the latest Repère, roll or action', () => {
        const actions = [{ kind: 'checker', roll_tick_ms: 10000, tick_ms: 15000 }, { kind: 'double', tick_ms: 18000 }, { kind: 'checker', roll_tick_ms: 25000 }, { kind: 'take' }];
        expect(lastRepere(actions)).toBe(25000);
        expect(videoResumeMs(3, actions)).toBe(25000 - RESUME_LEAD_MS);
        expect(videoResumeMs(3, [{ kind: 'checker', tick_ms: 1000 }])).toBe(0);
        expect(videoResumeMs(3, [])).toBe(0);
    });

    test('a storage that throws keeps nothing and breaks nothing', () => {
        const denied = () => {
            throw new Error('denied');
        };
        vi.stubGlobal('localStorage', { getItem: denied, setItem: denied });
        expect(() => saveVideoResume(1, 5000)).not.toThrow();
        expect(loadVideoResume(1)).toBeNull();
        expect(videoResumeMs(1, [{ kind: 'checker', roll_tick_ms: 9000 }])).toBe(6000);
    });
});
