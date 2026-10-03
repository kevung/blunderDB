/**
 * shortcutMap.test.js — no two keyboard scopes fight over a chord without saying so.
 *
 * Two scopes that can be active together (no shared `group`) and bind one chord either sit at
 * different tiers, and the earlier one lists the chord in `shadows`, or the press would go to
 * whichever registered first. A `shadows` entry that takes nothing from anyone is stale.
 */
import { describe, test, expect } from 'vitest';
import { SHORTCUTS, TIER } from '../services/shortcutMap.js';

const CHORD = /^(Ctrl\+)?(Alt\+)?(Shift\+)?([a-z0-9]|[A-Z]|[A-Z][A-Za-z0-9]+|[?/])$/;

const scopes = Object.entries(SHORTCUTS);

/**
 * Every pair of scopes that can be active together, the earlier tier first.
 * @returns {[string, import('../services/shortcutMap.js').ScopeDecl, string, import('../services/shortcutMap.js').ScopeDecl][]}
 */
function concurrentPairs() {
    const pairs = [];
    for (let i = 0; i < scopes.length; i++) {
        for (let j = i + 1; j < scopes.length; j++) {
            let [a, b] = [scopes[i], scopes[j]];
            if (a[1].group && a[1].group === b[1].group) continue;
            if (b[1].tier < a[1].tier) [a, b] = [b, a];
            pairs.push([a[0], a[1], b[0], b[1]]);
        }
    }
    return pairs;
}

describe('shortcutMap', () => {
    test('every scope has a known tier and well-formed chords', () => {
        const tiers = new Set(Object.values(TIER));
        for (const [name, decl] of scopes) {
            expect(tiers.has(decl.tier), name).toBe(true);
            for (const chord of decl.keys) expect(chord, `${name}: ${chord}`).toMatch(CHORD);
            // Shift with a letter is spelled with the capital (Shift+J), never Shift+j.
            for (const chord of decl.keys) expect(chord, `${name}: ${chord}`).not.toMatch(/Shift\+[a-z]$/);
        }
    });

    test('no scope declares a chord twice', () => {
        for (const [name, decl] of scopes) {
            const seen = new Set();
            const twice = decl.keys.filter((k) => (seen.has(k) ? true : (seen.add(k), false)));
            expect(twice, name).toEqual([]);
        }
    });

    test('two scopes active together never bind one chord at the same tier', () => {
        const clashes = [];
        for (const [an, a, bn, b] of concurrentPairs()) {
            if (a.tier !== b.tier) continue;
            for (const k of a.keys) if (b.keys.includes(k)) clashes.push(`${an} / ${bn}: ${k}`);
        }
        expect(clashes).toEqual([]);
    });

    test('a chord taken from a later tier is declared in shadows', () => {
        const undeclared = [];
        for (const [an, a, bn, b] of concurrentPairs()) {
            if (a.tier === b.tier) continue;
            for (const k of a.keys) if (b.keys.includes(k) && !(a.shadows ?? []).includes(k)) undeclared.push(`${an} takes ${k} from ${bn}`);
        }
        expect(undeclared).toEqual([]);
    });

    test('every shadows entry is a bound chord that some later scope also binds', () => {
        const stale = [];
        const pairs = concurrentPairs();
        for (const [name, decl] of scopes) {
            for (const k of decl.shadows ?? []) {
                const bound = decl.keys.includes(k);
                const taken = pairs.some(([an, a, , b]) => an === name && a.tier < b.tier && b.keys.includes(k));
                if (!bound || !taken) stale.push(`${name}: ${k}`);
            }
        }
        expect(stale).toEqual([]);
    });

    test('Home, End, PageUp and PageDown browse the list from the global tier', () => {
        for (const k of ['Home', 'End', 'PageUp', 'PageDown']) expect(SHORTCUTS.global.keys).toContain(k);
    });
});
