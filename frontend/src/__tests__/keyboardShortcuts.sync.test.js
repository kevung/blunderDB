// Guards the invariant that doc/source/raccourcis.rst — the only statement of
// the keyboard shortcuts, and the source the in-app help is generated from
// (ADR-0034) — names every letter shortcut keyboardService.js actually binds.
//
// commandVocabulary.sync.test.js and helpVocabulary.sync.test.js lock the
// command line to its documentation; until this file, the shortcuts had no
// such guard (tasks/critique-doc-2026-09, persona 7 #11). The check is
// one-directional and deliberately narrow: it extracts the `letter('x')` /
// `isShiftLetter(event, 'x')` bindings and their CTRL / SHIFT modifiers from
// the dispatcher's source, renders each as the French label the table uses
// (`CTRL-MAJ-I`, `MAJ-J`, `CTRL-N`, `p`), and requires that label in some key
// cell of the table. Arrow, Page, Escape and digit bindings are matched by
// hand-written code paths too varied to extract reliably, and stay reviewed by
// eye — the same trade-off helpVocabulary.sync.test.js makes for prefix
// filters.

import { describe, test, expect } from 'vitest';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const here = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(here, '..', '..', '..');
const service = fs.readFileSync(path.join(here, '..', 'services', 'keyboardService.js'), 'utf8');
const rst = fs.readFileSync(path.join(repoRoot, 'doc', 'source', 'raccourcis.rst'), 'utf8');

// Every quoted first cell of a csv-table row ("CTRL-PageUp, MAJ-J") yields its
// alternatives ("CTRL-PageUp", "MAJ-J"), upper-cased for comparison.
function documentedKeys() {
    const keys = new Set();
    for (const line of rst.split('\n')) {
        const m = /^ {3}"([^"]+)", "/.exec(line);
        if (!m) continue;
        for (const alt of m[1].split(',')) keys.add(alt.trim().toUpperCase());
    }
    return keys;
}

// Bindings the dispatcher makes, rendered as the table's labels.
function boundLabels() {
    const labels = new Map(); // label → first source line, for the message
    const lines = service.split('\n');
    lines.forEach((line, i) => {
        const code = line.replace(/\/\/.*$/, '');
        for (const m of code.matchAll(/letter\('([a-z])'\)/g)) {
            const ctrl = /event\.ctrlKey\s*&&/.test(code) && !/!event\.ctrlKey/.test(code);
            const shift = /event\.shiftKey\s*&&/.test(code);
            const key = m[1].toUpperCase();
            const label = ctrl && shift ? `CTRL-MAJ-${key}` : ctrl ? `CTRL-${key}` : key;
            if (!labels.has(label)) labels.set(label, i + 1);
        }
        for (const m of code.matchAll(/isShiftLetter\(event, '([a-z])'\)/g)) {
            const label = `MAJ-${m[1].toUpperCase()}`;
            if (!labels.has(label)) labels.set(label, i + 1);
        }
    });
    return labels;
}

describe('raccourcis.rst documents every letter shortcut keyboardService.js binds', () => {
    const documented = documentedKeys();
    const bound = boundLabels();

    test('the extraction sees the dispatcher (sanity)', () => {
        expect(bound.size).toBeGreaterThan(10);
        expect(documented.size).toBeGreaterThan(30);
    });

    for (const [label, line] of bound) {
        test(`${label} (keyboardService.js:${line}) has a row in raccourcis.rst`, () => {
            expect(documented.has(label), `no key cell "${label}" in doc/source/raccourcis.rst — add the row (then run make help)`).toBe(true);
        });
    }
});
