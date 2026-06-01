/**
 * i18n.locales.test.js
 *
 * Guards locale integrity:
 *   - every locale exposes EXACTLY the same set of leaf keys as English
 *     (no missing keys, no extras),
 *   - placeholder tokens ({n}, {name}, ...) match English per key, so a
 *     translation can never drop or invent an interpolation slot,
 *   - every help/<lang>.js module exports non-empty manual/shortcuts/commands/about.
 */

import { describe, test, expect } from 'vitest';

import en from '../i18n/locales/en.json';
import fi from '../i18n/locales/fi.json';
import el from '../i18n/locales/el.json';
import ja from '../i18n/locales/ja.json';
import de from '../i18n/locales/de.json';
import es from '../i18n/locales/es.json';
import fr from '../i18n/locales/fr.json';
import it from '../i18n/locales/it.json';
import ru from '../i18n/locales/ru.json';

import { LOCALES } from '../i18n';

const locales = { en, fi, el, ja, de, es, fr, it, ru };

// Flatten a nested dictionary into a map of dotted-key -> string value.
function flatten(dict, prefix = '', out = {}) {
    for (const [k, v] of Object.entries(dict)) {
        const nk = prefix ? `${prefix}.${k}` : k;
        if (v && typeof v === 'object') flatten(v, nk, out);
        else out[nk] = v;
    }
    return out;
}

function placeholders(str) {
    const set = new Set();
    if (typeof str === 'string') {
        const re = /\{(\w+)\}/g;
        let m;
        while ((m = re.exec(str)) !== null) set.add(m[1]);
    }
    return set;
}

const enFlat = flatten(en);
const enKeys = Object.keys(enFlat).sort();

test('LOCALES list matches the shipped locale files', () => {
    expect(LOCALES.slice().sort()).toEqual(Object.keys(locales).sort());
});

describe.each(Object.keys(locales).filter((c) => c !== 'en'))('locale %s', (code) => {
    const flat = flatten(locales[code]);
    const keys = Object.keys(flat).sort();

    test('has exactly the English key set', () => {
        const missing = enKeys.filter((k) => !(k in flat));
        const extra = keys.filter((k) => !(k in enFlat));
        expect({ missing, extra }).toEqual({ missing: [], extra: [] });
    });

    test('preserves placeholder tokens per key', () => {
        const mismatches = [];
        for (const k of enKeys) {
            const want = [...placeholders(enFlat[k])].sort();
            const got = [...placeholders(flat[k])].sort();
            if (want.join(',') !== got.join(',')) mismatches.push({ key: k, want, got });
        }
        expect(mismatches).toEqual([]);
    });

    test('has no empty string values', () => {
        const empty = keys.filter((k) => typeof flat[k] === 'string' && flat[k].trim() === '');
        expect(empty).toEqual([]);
    });
});

describe('help modules', () => {
    const HELP_TABS = ['manual', 'shortcuts', 'commands', 'about'];

    test.each(Object.keys(locales))('help/%s.js exports all four tabs, non-empty', async (code) => {
        const mod = await import(`../i18n/help/${code}.js`);
        const help = mod.default;
        expect(help).toBeTruthy();
        for (const tab of HELP_TABS) {
            expect(typeof help[tab]).toBe('string');
            expect(help[tab].length).toBeGreaterThan(0);
        }
    });
});
