import { describe, test, expect } from 'vitest';
import fr from '../i18n/locales/fr.json';

// Words that mean something in French only when translated: an English one left in fr.json is
// an untranslated string shown as is. Technical names (PR, MWC, Anki) are not on the list.
const ENGLISH = /\b(Dashboard|Reset|New|Search|Learning|Review|board|Breakdown|BREAKDOWN|[Cc]ube|[Cc]ubes)\b/;

/** @param {any} o @param {string} path @returns {Generator<[string, string]>} */
function* strings(o, path = '') {
    if (typeof o === 'string') yield [path, o];
    else if (o && typeof o === 'object') for (const [k, v] of Object.entries(o)) yield* strings(v, path ? `${path}.${k}` : k);
}

describe('fr.json carries no English UI word', () => {
    test('no string contains a word of the English list', () => {
        const bad = [...strings(fr)].filter(([, v]) => ENGLISH.test(v)).map(([k, v]) => `${k}: ${v}`);
        expect(bad).toEqual([]);
    });
});

// CONTEXT.md, « Pions / videau » : le type de décision se dit pions ou videau partout ; « coup »
// ne nomme qu'un jeu précis.
describe('the decision type is named pions / videau everywhere', () => {
    test.each([
        ['stats.decisionChecker', 'Pions'],
        ['stats.decisionCube', 'Videau'],
        ['stats.decisionTypeCube', 'Videau'],
        ['stats.cardLabelChecker', 'PR pions'],
        ['stats.cardLabelCube', 'PR videau'],
        ['stats.playersColPRChecker', 'PR pions'],
        ['stats.playersColPRCube', 'PR videau'],
        ['search.decision.checker', 'Pions'],
        ['search.decision.cube', 'Videau']
    ])('%s reads %s', (path, expected) => {
        const value = path.split('.').reduce((o, k) => o?.[k], /** @type {any} */ (fr));
        expect(value).toBe(expected);
    });
});
