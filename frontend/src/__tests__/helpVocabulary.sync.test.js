// Guards the invariant that the in-app help (`?`, HelpModal) documents exactly
// the commands the command line actually accepts — no more, no less.
//
// Since the help bundles became a generated artefact (see docs/adr, "the in-app
// help is generated from the documentation"), the tables this file parses are
// rendered from doc/source/cmd_mode.rst and its eight gettext catalogues. So
// what this file really locks is the *documentation* to commandVocabulary.js:
// a command added to the processor and the vocabulary but never written into
// cmd_mode.rst fails here, and so does a stale entry surviving in the manual
// after its command was removed (this is exactly how `filter, fl` outlived the
// panel it used to open — the sync test that would have caught it didn't exist).
//
// commandVocabulary.sync.test.js already locks commandVocabulary.js to
// commandProcessor.js's if/else chain; this file closes the documentation gap.
//
// The exhaustive parse runs on fr.js, the source language. The other eight are
// renderings of the same tables through the .po catalogues, and they are checked
// on the two things a translator can break: a mangled command name or filter
// token (both are meant to travel untranslated), and a dropped section, item or
// row.
//
// A lighter, one-directional check does the same for search filter tokens:
// every token the shared search grammar (searchFilterService.js's
// parseSearchTokens, called by both commandProcessor.js's parseFilters and
// this file's parseSearchCommand — see #203) checks with a plain
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
const REPO_ROOT = path.join(__dirname, '../../..');

// The heading of the filter table is the title of the section cmd_mode.rst
// labels `.. _cmd_filter:`. Reading it from the source rather than hard-coding
// "Filtres de recherche" means renaming that section is an editorial change,
// not a broken test — while moving or deleting the anchor still fails here.
function frenchTitleOfAnchor(rstFile, anchor) {
    const lines = fs.readFileSync(path.join(REPO_ROOT, 'doc/source', rstFile), 'utf8').split('\n');
    const at = lines.findIndex((l) => l.trim() === `.. _${anchor}:`);
    expect(at, `${rstFile} has no \`.. _${anchor}:\` anchor`).toBeGreaterThanOrEqual(0);
    for (let i = at + 1; i < lines.length - 1; i++) {
        const title = lines[i].trim();
        if (!title) continue;
        const underline = lines[i + 1].trim();
        if (underline.length >= title.length && /^(.)\1{3,}$/.test(underline)) return title;
        break;
    }
    throw new Error(`no section title follows \`.. _${anchor}:\` in ${rstFile}`);
}

const FILTER_HEADING = frenchTitleOfAnchor('cmd_mode.rst', 'cmd_filter');

// Every <table> in the help HTML fragment, keyed by the text of the <h3> that
// immediately precedes it (or '' if none). jsdom repairs the occasional
// malformed markup in the source (e.g. an attribute wrapped onto its own
// line), so this is more robust than a regex over the raw string.
function tablesByHeading(html) {
    const doc = new DOMParser().parseFromString(`<div>${html}</div>`, 'text/html');
    const map = new Map();
    for (const table of doc.querySelectorAll('table')) {
        let heading = table.previousElementSibling;
        // A section may open with an intro paragraph or an admonition before
        // its table; walk back to the nearest <h3>.
        while (heading && heading.tagName !== 'H3') heading = heading.previousElementSibling;
        map.set(heading ? heading.textContent.trim() : '', table);
    }
    return map;
}

function firstColumnLabels(table) {
    return [...table.querySelectorAll('tbody tr')].map((row) => row.querySelector('td')?.textContent.trim() ?? '');
}

// `[number]` (position navigation) and `#tag1 tag2 ...` (tag insertion) are
// deliberately not commands — commandVocabulary.js excludes them too (see its
// header comment) — and `blunders, bl [n]` documents an optional argument
// inline; strip it before comparing against the vocabulary's bare `blunders, bl`.
function commandLabelsOf(commandsHtml) {
    return [...tablesByHeading(commandsHtml).entries()]
        .filter(([heading]) => heading !== FILTER_HEADING)
        .flatMap(([, table]) => firstColumnLabels(table))
        .filter((label) => label && !label.startsWith('#') && !label.startsWith('['))
        .map((label) => label.replace(/\s*\[.*\]$/, ''));
}

// The vocabulary's canonical label for a command, in the same "name, alias1,
// alias2" shape the help tables use (e.g. `write!, wr!, w!`).
function vocabLabel(cmd) {
    return cmd.name + (cmd.aliases.length ? ', ' + cmd.aliases.join(', ') : '');
}

const booleanFilterTokens = () => {
    const grammarSrc = fs.readFileSync(path.join(__dirname, '../services/searchFilterService.js'), 'utf8');
    return [...new Set([...grammarSrc.matchAll(/filters\.includes\('([^']+)'\)/g)].map((m) => m[1]))];
};

describe('help ↔ commandVocabulary sync (fr.js)', () => {
    const commandLabels = commandLabelsOf(helpFr.commands);

    test('every commandVocabulary.js entry is documented in the commands table', () => {
        const documented = new Set(commandLabels);
        const missing = COMMANDS.map(vocabLabel).filter((label) => !documented.has(label));
        expect(missing, `commandVocabulary.js entries missing from doc/source/cmd_mode.rst: ${missing.join(', ')}`).toEqual([]);
    });

    test('no ghost command entries in the commands table (e.g. the old `filter, fl`)', () => {
        const expected = new Set(COMMANDS.map(vocabLabel));
        const ghosts = commandLabels.filter((label) => !expected.has(label));
        expect(ghosts, `doc/source/cmd_mode.rst documents commands absent from commandVocabulary.js: ${ghosts.join(', ')}`).toEqual([]);
    });
});

describe('help ↔ parseFilters sync (fr.js)', () => {
    test('every boolean filters.includes(...) token in the search grammar is documented in the filter table', () => {
        const tokens = booleanFilterTokens();
        expect(tokens.length, 'no filters.includes(...) tokens found — has parseFilters been rewritten?').toBeGreaterThan(0);

        const filterTable = tablesByHeading(helpFr.commands).get(FILTER_HEADING);
        expect(filterTable, `no table under the "${FILTER_HEADING}" heading in the commands tab`).toBeTruthy();
        // Rows combine aliases in one cell (e.g. `cube, cub, cu, c`); split each
        // on commas so every individual token can be checked on its own.
        const documentedTokens = new Set(firstColumnLabels(filterTable).flatMap((label) => label.split(',').map((t) => t.trim())));

        const missing = tokens.filter((tok) => !documentedTokens.has(tok));
        expect(missing, `Filter tokens checked via filters.includes(...) but missing from the cmd_mode.rst filter table: ${missing.join(', ')}`).toEqual([]);
    });
});

describe('help translations stay structurally in sync', () => {
    // Translating help/*.js does not require re-implementing the DOM parse
    // above for all 9 languages: as long as every translation carries, tab by
    // tab, the same number of <h3> sections, <li> items and <tr> rows as
    // fr.js, no section, item or row was dropped or left behind when fr.js
    // changed. This does not verify translation quality — only that the
    // structural change reached every language file, in every tab (a row
    // added to the commands table used to be the only thing checked; a new
    // shortcut or manual section could still silently miss a language).
    const TABS = ['manual', 'shortcuts', 'commands', 'about'];
    const COUNTED = ['h3', 'li', 'tr'];
    const shape = (html) => {
        const doc = new DOMParser().parseFromString(`<div>${html}</div>`, 'text/html');
        return Object.fromEntries(COUNTED.map((tag) => [tag, doc.querySelectorAll(tag).length]));
    };
    const frShape = Object.fromEntries(TABS.map((tab) => [tab, shape(helpFr[tab])]));

    test('fr.js has sections, items and rows to compare against', () => {
        for (const tab of TABS) {
            expect(typeof helpFr[tab], `help/fr.js has no "${tab}" tab`).toBe('string');
        }
        expect(frShape.commands.tr).toBeGreaterThan(0);
        expect(frShape.shortcuts.tr).toBeGreaterThan(0);
        expect(frShape.manual.h3).toBeGreaterThan(0);
        expect(frShape.manual.li + frShape.about.li).toBeGreaterThan(0);
    });

    const helpDir = path.join(__dirname, '../i18n/help');
    const otherLanguages = fs
        .readdirSync(helpDir)
        .filter((f) => f.endsWith('.js') && f !== 'index.js' && f !== 'fr.js')
        .map((f) => f.replace(/\.js$/, ''));

    test('the 8 translations are all present', () => {
        expect(otherLanguages.sort()).toEqual(['de', 'el', 'en', 'es', 'fi', 'it', 'ja', 'ru']);
    });

    test.each(otherLanguages)('%s.js has the same h3/li/tr counts as fr.js in every tab', async (lang) => {
        const mod = await import(`../i18n/help/${lang}.js`);
        const langShape = Object.fromEntries(TABS.map((tab) => [tab, shape(mod.default[tab] ?? '')]));
        expect(langShape, `${lang}.js drifted structurally from fr.js (counts are h3/li/tr per tab)`).toEqual(frShape);
    });

    // Command names and filter tokens are code, not prose: they must survive
    // translation byte for byte. A catalogue that "translated" `write!, wr!, w!`
    // documents a command nobody can type, in one language only — the kind of
    // gap the structural counts above cannot see.
    test.each(otherLanguages)('%s.js documents every command and boolean filter token verbatim', async (lang) => {
        const mod = await import(`../i18n/help/${lang}.js`);
        const cells = new Set(
            [...new DOMParser().parseFromString(`<div>${mod.default.commands}</div>`, 'text/html').querySelectorAll('tbody tr')].flatMap((row) => {
                const first = row.querySelector('td')?.textContent.trim() ?? '';
                return [first.replace(/\s*\[.*\]$/, ''), ...first.split(',').map((t) => t.trim())];
            })
        );
        const missing = [...COMMANDS.map(vocabLabel), ...booleanFilterTokens()].filter((label) => !cells.has(label));
        expect(missing, `${lang}: command labels / filter tokens altered or missing in the translated tables: ${missing.join(', ')}`).toEqual([]);
    });
});
