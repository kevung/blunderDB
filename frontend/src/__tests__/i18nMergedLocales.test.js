import { describe, test, expect } from 'vitest';
import { merge } from '../../i18nMergedLocales.js';

const loaders = import.meta.glob('../i18n/locales/*.json', { query: '?merged', import: 'default' });

/** @param {any} o @param {string} p @returns {string[]} */
function keys(o, p = '') {
    return Object.entries(o).flatMap(([k, v]) => (v && typeof v === 'object' ? keys(v, `${p}${k}.`) : [`${p}${k}`]));
}

describe('locales repliées sur l’anglais au build', () => {
    test('merge garde la langue, complète avec la base, récursivement', () => {
        expect(merge({ a: 'A', n: { x: 'X', y: 'Y' } }, { n: { x: 'x2' } })).toEqual({ a: 'A', n: { x: 'x2', y: 'Y' } });
    });

    test('chaque locale chargée porte toutes les clés anglaises : aucune clé brute', async () => {
        const en = keys(await loaders['../i18n/locales/en.json']());
        for (const [path, load] of Object.entries(loaders)) {
            const have = new Set(keys(await load()));
            const missing = en.filter((k) => !have.has(k));
            expect(missing, path).toEqual([]);
        }
    });
});
