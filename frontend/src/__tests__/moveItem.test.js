import { describe, test, expect } from 'vitest';
import { moveItem } from '../utils/moveItem.js';

const id = (x) => x;

describe('moveItem', () => {
    test('moves an item before or after another', () => {
        expect(moveItem([1, 2, 3, 4], id, 1, 3, false)).toEqual([2, 1, 3, 4]);
        expect(moveItem([1, 2, 3, 4], id, 1, 3, true)).toEqual([2, 3, 1, 4]);
        expect(moveItem([1, 2, 3, 4], id, 4, 1, false)).toEqual([4, 1, 2, 3]);
        expect(moveItem([1, 2, 3, 4], id, 4, 2, true)).toEqual([1, 2, 4, 3]);
    });

    test('returns the same list when nothing moves', () => {
        const list = [1, 2, 3];
        expect(moveItem(list, id, 2, 2, true)).toBe(list);
        expect(moveItem(list, id, 9, 1, true)).toBe(list);
        expect(moveItem(list, id, 1, 9, true)).toBe(list);
        expect(moveItem(list, id, 1, 2, false)).toBe(list);
        expect(moveItem(list, id, 2, 1, true)).toBe(list);
    });
});
