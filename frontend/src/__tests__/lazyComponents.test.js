/**
 * The panels and modals kept off the first screen: each registry entry must resolve to a
 * component, or its tab or modal would stay empty — and the preload is what makes a first
 * click find its component already there.
 */
import { describe, test, expect } from 'vitest';
import { get } from 'svelte/store';
import { panels, modals, preloadDeferredComponents, lazyRegistry } from '../components/lazyComponents.js';
import { TABS } from '../services/tabCatalog.js';

// The first import of a component compiles it and its whole subtree: slow on a loaded runner.
const SLOW = 60000;

const lazyRegistryForTest = (loader) => lazyRegistry({ x: loader });

describe('lazyComponents', () => {
    test('nothing is loaded before it is asked for', () => {
        expect(get(panels.components)).toEqual({});
        expect(get(modals.components)).toEqual({});
    });

    test(
        'loading one panel fetches it alone, and only once',
        async () => {
            await Promise.all([panels.load('matches'), panels.load('matches')]);
            expect(Object.keys(get(panels.components))).toEqual(['matches']);
            expect(typeof get(panels.components).matches).toBe('function');
        },
        SLOW
    );

    test('an unknown name is ignored', async () => {
        await panels.load('no-such-tab');
        expect(get(panels.components)['no-such-tab']).toBeUndefined();
    });

    test(
        'the preload loads every tab of the catalog and every modal',
        async () => {
            await preloadDeferredComponents();
            const loaded = get(panels.components);
            for (const tab of TABS) expect(typeof loaded[tab.id], tab.id).toBe('function');
            for (const component of Object.values(get(modals.components))) expect(typeof component).toBe('function');
            expect(Object.keys(get(modals.components)).length).toBe(16);
        },
        SLOW
    );

    test('a failed fetch is reported, then retried on the next load', async () => {
        let calls = 0;
        const registry = lazyRegistryForTest(() => (++calls === 1 ? Promise.reject(new Error('net')) : Promise.resolve({ default: () => {} })));
        await registry.load('x');
        expect(get(registry.failed).has('x')).toBe(true);
        await registry.load('x');
        expect(get(registry.failed).has('x')).toBe(false);
        expect(typeof get(registry.components).x).toBe('function');
    });
});
