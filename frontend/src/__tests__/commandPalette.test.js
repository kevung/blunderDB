/**
 * commandPalette.test.js — what the palette lists and how it ranks it (#287).
 *
 * The palette owns no list: commands come from commandVocabulary.js, tabs from
 * tabCatalog.js, filters from the library, matches from GetAllMatches. These
 * tests hold the two promises that follow from it — every command of the
 * vocabulary is reachable, by its name and by each of its aliases, and has a
 * description in the nine locales.
 */

import { describe, test, expect } from 'vitest';
import { COMMANDS } from '../commandVocabulary.js';
import { TABS } from '../services/tabCatalog.js';
import { buildPaletteItems, rankPaletteItems, commandLabelKey } from '../services/commandPalette.js';
import { translate } from '../i18n';

import fr from '../i18n/locales/fr.json';
import en from '../i18n/locales/en.json';
import de from '../i18n/locales/de.json';
import el from '../i18n/locales/el.json';
import es from '../i18n/locales/es.json';
import fi from '../i18n/locales/fi.json';
import it from '../i18n/locales/it.json';
import ja from '../i18n/locales/ja.json';
import ru from '../i18n/locales/ru.json';

const LOCALES = { fr, en, de, el, es, fi, it, ja, ru };

const MATCHES = [
    { id: 1, player1_name: 'Alice', player2_name: 'Bob', match_length: 7, match_date: '2026-01-15', tournament_name: 'Open de Lyon' },
    { id: 2, player1_name: 'Carla', player2_name: 'Dimitri', match_length: 11, match_date: '2026-03-02', event: 'Monte-Carlo' }
];
const FILTERS = [
    { id: 10, name: 'Blitz ratés', command: 's gt:blitz E>100', pinned: false },
    { id: 11, name: 'Courses serrées', command: 's nc w45,55', pinned: true }
];

const items = buildPaletteItems({ translate, matches: MATCHES, filters: FILTERS });
const first = (/** @type {string} */ query) => rankPaletteItems(items, query)[0]?.item;

describe('every command of the vocabulary is reachable in the palette', () => {
    for (const cmd of COMMANDS) {
        test(`${cmd.name}`, () => {
            expect(items.some((i) => i.id === `command:${cmd.name}`)).toBe(true);
            for (const form of [cmd.name, ...cmd.aliases]) {
                expect(first(form)?.id, `typing "${form}" should put ${cmd.name} first`).toBe(`command:${cmd.name}`);
            }
        });
    }

    test('each has a description in the nine locales', () => {
        for (const [lang, messages] of Object.entries(LOCALES)) {
            for (const cmd of COMMANDS) {
                const leaf = messages.palette?.cmd?.[cmd.name];
                expect(typeof leaf === 'string' && leaf.length > 0, `${lang}: missing ${commandLabelKey(cmd.name)}`).toBe(true);
            }
        }
    });

    test('the locales describe no command the vocabulary has dropped', () => {
        const names = new Set(COMMANDS.map((c) => c.name));
        for (const [lang, messages] of Object.entries(LOCALES)) {
            for (const name of Object.keys(messages.palette.cmd)) expect(names.has(name), `${lang}: palette.cmd.${name}`).toBe(true);
        }
    });
});

describe('the other sources', () => {
    test('every tab of the tabbed panel is listed', () => {
        for (const tab of TABS) expect(items.some((i) => i.id === `tab:${tab.id}`)).toBe(true);
    });

    test('a match is found by a player, approximately, and by its tournament', () => {
        expect(first('dimtri')?.id).toBe('match:2');
        expect(first('lyon')?.id).toBe('match:1');
    });

    test('a saved filter is found by its name, accents folded', () => {
        expect(first('blitz rates')?.id).toBe('filter:10');
    });

    test('a description finds its command: "matrice" is the cube matrix', () => {
        const fr = buildPaletteItems({ translate: (key) => key.split('.').reduce((o, k) => o?.[k], LOCALES.fr) ?? key });
        const hit = rankPaletteItems(fr, 'matrice videau')[0]?.item;
        expect(hit?.id).toBe('command:cm');
        // The example manuel.rst gives, verbatim.
        expect(rankPaletteItems(fr, 'mtrc')[0]?.item.id).toBe('command:cm');
    });

    test('at rest, pinned filters come first and matches last, newest first', () => {
        const resting = rankPaletteItems(items, '', 500).map((r) => r.item.id);
        expect(resting[0]).toBe('filter:11');
        expect(resting.slice(-2)).toEqual(['match:2', 'match:1']);
    });

    test('a query nothing matches returns nothing', () => {
        expect(rankPaletteItems(items, 'zzzzqqq')).toEqual([]);
    });
});
