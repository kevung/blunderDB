import { describe, test, expect } from 'vitest';
import { modifierSelect, idsFor } from '../utils/multiSelect.js';

const rows = [1, 2, 3, 4, 5].map((id) => ({ id }));
/** @param {Parameters<typeof modifierSelect>} args */
const ms = (...args) => /** @type {{ keys: Set<number>, anchor: number }} */ (modifierSelect(...args));
const key = (/** @type {{ id: number }} */ r) => r.id;

describe('modifierSelect', () => {
    test('a plain click is not handled', () => {
        expect(modifierSelect(rows, key, new Set(), null, rows[0], {})).toBeNull();
    });
    test('Ctrl+click adds, keeps the single selection as seed, then removes', () => {
        let r = ms(rows, key, new Set(), null, rows[2], { ctrlKey: true }, 1);
        expect([...r.keys]).toEqual([1, 3]);
        r = ms(rows, key, r.keys, r.anchor, rows[2], { metaKey: true });
        expect([...r.keys]).toEqual([1]);
    });
    test('Shift+click selects the range from the anchor, either direction', () => {
        let r = ms(rows, key, new Set(), 4, rows[1], { shiftKey: true });
        expect([...r.keys].sort()).toEqual([2, 3, 4]);
        r = ms(rows, key, new Set(), null, rows[3], { shiftKey: true }, 2);
        expect([...r.keys].sort()).toEqual([2, 3, 4]);
    });
    test('Ctrl+Shift+click extends the selection', () => {
        const r = ms(rows, key, new Set([1]), 3, rows[4], { shiftKey: true, ctrlKey: true });
        expect([...r.keys].sort()).toEqual([1, 3, 4, 5]);
    });
});

describe('idsFor', () => {
    test('the selection in display order when the row belongs to it, the row alone otherwise', () => {
        expect(idsFor(rows, key, new Set([4, 2]), 2)).toEqual([2, 4]);
        expect(idsFor(rows, key, new Set([4, 2]), 3)).toEqual([3]);
        expect(idsFor(rows, key, new Set([4]), 4)).toEqual([4]);
    });
});
