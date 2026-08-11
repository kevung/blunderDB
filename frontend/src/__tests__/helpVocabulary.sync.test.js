// Guards the invariant that the in-app help (`?`, HelpModal) documents exactly
// the commands the command line actually accepts — no more, no less.
//
// commandVocabulary.sync.test.js already locks commandVocabulary.js to
// commandProcessor.js's if/else chain. This file closes the remaining gap: the
// *documentation* (help/fr.js's "Commandes" table) can still drift from
// commandVocabulary.js in either direction — a command added to the processor
// and vocabulary but never documented, or a stale entry surviving in the help
// after its command was removed (this is exactly how `filter, fl` outlived the
// panel it used to open — the sync test that would have caught it didn't exist).
//
// Only fr.js is parsed: it is the source language (the other 8 are declined
// translations of the same table structure, see i18nKeys.sync.test.js and the
// row-count check in this file for how the *other* 8 are kept honest without
// re-implementing this whole parse per language).
//
// A lighter, one-directional check does the same for search filter tokens:
// every token commandProcessor.js's parseFilters checks with a plain
// `filters.includes('token')` (boolean flags like `nc`, `fl`, `co`, `xco`,
// `D1`, `M`, `i`, `x`, plus the cube/score alias groups) must appear in the
// help's filter table. Range/prefix filters (`p>x`, `e<x`, `max`, `idx`, …)
// are matched with `.startsWith()`/regexes rather than `.includes()` and are
// NOT mechanically covered here — turning those into a reliable extraction
// would require re-deriving parseFilters' prefix logic in the test, which is
// itself a drift risk. They stay documented but only reviewed by hand; see
// doc/source/cmd_mode.rst, which is kept as the authoritative filter list.

import { describe, test, expect } from 'vitest';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { COMMANDS } from '../commandVocabulary.js';
import helpFr from '../i18n/help/fr.js';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

// Every <table> in the help HTML fragment, keyed by the text of the <h3> that
// immediately precedes it (or '' if none). jsdom repairs the occasional
// malformed markup in the source (e.g. an attribute wrapped onto its own
// line), so this is more robust than a regex over the raw string.
function tablesByHeading(html) {
    const doc = new DOMParser().parseFromString(`<div>${html}</div>`, 'text/html');
    const map = new Map();
    for (const table of doc.querySelectorAll('table')) {
        const heading = table.previousElementSibling;
        const key = heading && heading.tagName === 'H3' ? heading.textContent.trim() : '';
        map.set(key, table);
    }
    return map;
}

function firstColumnLabels(table) {
    return [...table.querySelectorAll('tbody tr')].map((row) => row.querySelector('td')?.textContent.trim() ?? '');
}

// The vocabulary's canonical label for a command, in the same "name, alias1,
// alias2" shape the help tables use (e.g. `write!, wr!, w!`).
function vocabLabel(cmd) {
    return cmd.name + (cmd.aliases.length ? ', ' + cmd.aliases.join(', ') : '');
}

describe('help ↔ commandVocabulary sync (fr.js)', () => {
    const tables = tablesByHeading(helpFr.commands);
    // Every command table except "Filtres" (filter tokens are a different
    // vocabulary, checked separately below) documents real commands.
    const commandLabels = [...tables.entries()]
        .filter(([heading]) => heading !== 'Filtres')
        .flatMap(([, table]) => firstColumnLabels(table))
        // `[number]` (position navigation) and `#tag1 tag2 ...` (tag insertion)
        // are deliberately not commands — commandVocabulary.js excludes them
        // too (see its header comment) — so they are not part of this sync.
        .filter((label) => label && !label.startsWith('#') && !label.startsWith('['))
        // `blunders, bl [n]` documents an optional argument inline; strip it
        // before comparing against the vocabulary's bare `blunders, bl`.
        .map((label) => label.replace(/\s*\[.*\]$/, ''));

    test('every commandVocabulary.js entry is documented in the commands table', () => {
        const documented = new Set(commandLabels);
        const missing = COMMANDS.map(vocabLabel).filter((label) => !documented.has(label));
        expect(missing, `commandVocabulary.js entries missing from help/fr.js: ${missing.join(', ')}`).toEqual([]);
    });

    test('no ghost command entries in the commands table (e.g. the old `filter, fl`)', () => {
        const expected = new Set(COMMANDS.map(vocabLabel));
        const ghosts = commandLabels.filter((label) => !expected.has(label));
        expect(ghosts, `help/fr.js documents commands absent from commandVocabulary.js: ${ghosts.join(', ')}`).toEqual([]);
    });
});

describe('help ↔ parseFilters sync (fr.js)', () => {
    test('every boolean filters.includes(...) token in commandProcessor.js is documented in the filter table', () => {
        const processorSrc = fs.readFileSync(path.join(__dirname, '../commandProcessor.js'), 'utf8');
        const tokens = [...new Set([...processorSrc.matchAll(/filters\.includes\('([^']+)'\)/g)].map((m) => m[1]))];
        expect(tokens.length, 'no filters.includes(...) tokens found — has parseFilters been rewritten?').toBeGreaterThan(0);

        const filterTable = tablesByHeading(helpFr.commands).get('Filtres');
        // Rows combine aliases in one cell (e.g. `cube, cub, cu, c`); split each
        // on commas so every individual token can be checked on its own.
        const documentedTokens = new Set(firstColumnLabels(filterTable).flatMap((label) => label.split(',').map((t) => t.trim())));

        const missing = tokens.filter((tok) => !documentedTokens.has(tok));
        expect(missing, `Filter tokens checked via filters.includes(...) but missing from the help/fr.js filter table: ${missing.join(', ')}`).toEqual([]);
    });
});

describe('help translations stay structurally in sync', () => {
    // Translating help/*.js does not require re-implementing the DOM parse
    // above for all 9 languages: as long as every translation's "commands"
    // table carries the same number of <tr> rows as fr.js, no row was dropped
    // or left behind when fr.js gained the entries checked above. This does
    // not verify translation quality — only that the structural change (row
    // added/removed) reached every language file.
    const rowCount = (commandsHTML) => [...new DOMParser().parseFromString(`<div>${commandsHTML}</div>`, 'text/html').querySelectorAll('table tbody tr')].length;
    const frCount = rowCount(helpFr.commands);

    test('fr.js has rows to compare against', () => {
        expect(frCount).toBeGreaterThan(0);
    });

    const helpDir = path.join(__dirname, '../i18n/help');
    const otherLanguages = fs
        .readdirSync(helpDir)
        .filter((f) => f.endsWith('.js') && f !== 'index.js' && f !== 'fr.js')
        .map((f) => f.replace(/\.js$/, ''));

    test.each(otherLanguages)('%s.js has the same commands-table row count as fr.js', async (lang) => {
        const mod = await import(`../i18n/help/${lang}.js`);
        expect(rowCount(mod.default.commands)).toBe(frCount);
    });
});
