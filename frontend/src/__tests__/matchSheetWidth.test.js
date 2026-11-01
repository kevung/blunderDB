import { afterEach, describe, expect, it } from 'vitest';
import { clampSheetWidth, rememberedSheetShare, rememberSheetShare } from '../utils/matchSheet.js';

afterEach(() => localStorage.clear());

describe('clampSheetWidth', () => {
    it('leaves each pane a readable width', () => {
        expect(clampSheetWidth(50, 1000)).toBe(200);
        expect(clampSheetWidth(950, 1000)).toBe(800);
        expect(clampSheetWidth(600, 1000)).toBe(600);
    });

    it('splits in half a panel too narrow for both', () => {
        expect(clampSheetWidth(50, 300)).toBe(150);
        expect(clampSheetWidth(290, 300)).toBe(150);
    });
});

describe('rememberedSheetShare', () => {
    it('gives back what was kept', () => {
        rememberSheetShare(0.7);
        expect(rememberedSheetShare()).toBe(0.7);
    });

    it.each(['abc', '0', '1', '-0.2', '3'])('falls back to the default on %s', (raw) => {
        localStorage.setItem('blunderdb.matchSheetShare', raw);
        expect(rememberedSheetShare()).toBe(0.55);
    });
});
