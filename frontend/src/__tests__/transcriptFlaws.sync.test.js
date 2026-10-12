/**
 * Une Incohérence se lit par la règle qu'elle enfreint : chaque règle que le
 * Replay pose (pkg/blunderdb/transcript/replay.go) a sa clé, traduite dans les
 * neuf langues, et la marque nomme l'élément en cause.
 */
import { describe, test, expect } from 'vitest';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { RULE_KEY, KIND_KEY, flawText } from '../utils/transcriptFlaws.js';

const here = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(here, '..', '..', '..');
const replayGo = fs.readFileSync(path.join(repoRoot, 'pkg', 'blunderdb', 'transcript', 'replay.go'), 'utf8');
const goRules = [...replayGo.matchAll(/^\s*Rule\w+\s+InconsistencyRule = "(\w+)"/gm)].map((m) => m[1]);
const goKinds = [...replayGo.matchAll(/^\s*\w+\s+InconsistencyKind = "(\w+)"/gm)].map((m) => m[1]);
const LANGS = ['fr', 'en', 'de', 'es', 'it', 'fi', 'el', 'ja', 'ru'];

/** @param {any} tree @param {string} key */
function lookup(tree, key) {
    return key.split('.').reduce((node, part) => node?.[part], tree);
}

describe('transcript inconsistency rules', () => {
    test('every rule of the Replay has a key, and no key outlives its rule', () => {
        expect(goRules.length).toBeGreaterThan(15);
        expect(Object.keys(RULE_KEY).sort()).toEqual([...goRules].sort());
        expect(Object.keys(KIND_KEY).sort()).toEqual([...goKinds].sort());
    });

    test.each(LANGS)('%s translates every rule and every kind', (lang) => {
        const tree = JSON.parse(fs.readFileSync(path.join(here, '..', 'i18n', 'locales', `${lang}.json`), 'utf8'));
        for (const key of [...Object.values(RULE_KEY), ...Object.values(KIND_KEY)]) {
            expect(typeof lookup(tree, key), `${lang}: ${key}`).toBe('string');
        }
    });

    const t = (/** @type {string} */ key, /** @type {Record<string, any>} */ params = {}) => `${key}${JSON.stringify(params)}`;
    const names = (/** @type {number} */ side) => ['Alice', 'Bob'][side];

    test('names the player in cause, not a number', () => {
        expect(flawText({ kind: 'impossible_cube', rule: 'cube_not_held', args: { player: 2 } }, t, names)).toBe('transcript.rule.cube_not_held{"player":"Bob"}');
    });

    test('carries the dice and the derived score', () => {
        expect(flawText({ kind: 'inconsistent_dice', rule: 'dice_unused', args: { d1: 5, d2: 2 } }, t, names)).toBe('transcript.rule.dice_unused{"d1":5,"d2":2}');
        expect(flawText({ kind: 'score_mismatch', rule: 'score_differs', args: { a: 3, b: 1 } }, t, names)).toBe('transcript.rule.score_differs{"a":3,"b":1}');
    });

    test('a flaw without a rule falls back on its kind, timecodes included', () => {
        expect(flawText({ kind: 'timecode_backwards' }, t, names)).toBe('transcript.inconsistency.timecode_backwards{}');
        expect(flawText({ kind: 'mystery' }, t, names)).toBe('mystery');
    });
});
