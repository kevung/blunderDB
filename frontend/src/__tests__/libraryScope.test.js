/**
 * databaseLoadedStore stays true from one library to the next: a store that holds data of the
 * library and is neither photographed with its group nor read again on a change shows one
 * library's data in another. Every exported store of src/stores must say which it is.
 */

import { test, expect } from 'vitest';
import { PHOTOGRAPHED, PHOTOGRAPHED_ELSEWHERE, RELOADED, RELOADED_ELSEWHERE, REFUSED, OUTSIDE } from '../stores/libraryScope.js';

const modules = /** @type {Record<string, Record<string, any>>} */ (import.meta.glob('../stores/*.js', { eager: true }));

/** @type {Map<string, any>} */
const exported = new Map();
for (const mod of Object.values(modules)) {
    for (const [name, value] of Object.entries(mod)) {
        if (name.endsWith('Store') && typeof value === 'object' && value !== null) exported.set(name, value);
    }
}

const classified = [...Object.keys(PHOTOGRAPHED), ...PHOTOGRAPHED_ELSEWHERE, ...Object.keys(RELOADED), ...RELOADED_ELSEWHERE, ...REFUSED, ...OUTSIDE];

test('every store of src/stores is photographed, reloaded, refused or outside the library', () => {
    const missing = [...exported.keys()].filter((name) => !classified.includes(name));
    expect(missing, 'classify these in src/stores/libraryScope.js').toEqual([]);
});

test('each store is classified once, and only stores that exist', () => {
    const twice = classified.filter((name, i) => classified.indexOf(name) !== i);
    expect(twice).toEqual([]);
    const unknown = classified.filter((name) => !exported.has(name));
    expect(unknown).toEqual([]);
});

test('the scoped tables hold the exported stores themselves', () => {
    for (const [name, [store, initial]] of [...Object.entries(PHOTOGRAPHED), ...Object.entries(RELOADED)]) {
        expect(store, name).toBe(exported.get(name));
        expect(typeof store.set, name).toBe('function');
        expect(typeof initial, name).toBe('function');
    }
});
