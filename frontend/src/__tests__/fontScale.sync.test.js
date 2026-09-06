/**
 * fontScale.sync.test.js
 *
 * ADR-0008: the interface has one type scale, declared once as tokens in style.css, and a
 * component never declares an absolute font size. The migration that established this was
 * visual and untested, so the rule only held as long as reviewers remembered it — the two
 * components fixed right before the ADR each came back with their own base size.
 *
 * This test walks every .svelte file and fails on any `font-size` whose value is not one of
 * the tokens declared in style.css (or `inherit`), and on any `font` shorthand other than
 * `inherit` (a `font: 12px monospace` would smuggle a size past the first check). The three
 * chrome tokens are named exceptions in the ADR and are accepted only where it places them;
 * anywhere else they are just another way of leaving the scale.
 *
 * Adding an exception means amending docs/adr/0008 first, then CHROME_TOKEN_SCOPES below.
 */

import { describe, test, expect } from 'vitest';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const SRC = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');

// Where ADR-0008 allows each chrome token (paths relative to src/, POSIX separators). The
// body tokens (base, small, title) are allowed everywhere and are not listed.
const CHROME_TOKEN_SCOPES = {
    // "Dialog titles": every modal, plus App.svelte's file-drop overlay, which is read the
    // same way a modal title is.
    '--font-size-dialog-title': [/Modal\.svelte$/, /^App\.svelte$/],
    // "Dialog close crosses".
    '--font-size-dialog-close': [/Modal\.svelte$/],
    // "Figures meant to be read at a glance": the statistics tabs and the running counters
    // of the two import-progress dialogs.
    '--font-size-stat-figure': [/^components\/stats\//, /^components\/(FileImportProgressModal|ImportProgressModal)\.svelte$/]
};

function svelteFiles(dir) {
    return fs.readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
        const full = path.join(dir, entry.name);
        if (entry.isDirectory()) return entry.name === '__tests__' ? [] : svelteFiles(full);
        return entry.name.endsWith('.svelte') ? [full] : [];
    });
}

function declaredTokens() {
    const css = fs.readFileSync(path.join(SRC, 'tokens.css'), 'utf8') + fs.readFileSync(path.join(SRC, 'style.css'), 'utf8');
    return new Set([...css.matchAll(/(--font-size-[\w-]+)\s*:/g)].map((m) => m[1]));
}

// Every `font-size:` / `font:` declaration in the sources, with its location.
function declarations() {
    const found = [];
    for (const file of svelteFiles(SRC)) {
        const rel = path.relative(SRC, file).split(path.sep).join('/');
        const lines = fs.readFileSync(file, 'utf8').split('\n');
        lines.forEach((line, i) => {
            for (const m of line.matchAll(/\bfont(-size)?\s*:\s*([^;{}]+?)\s*;/g)) {
                found.push({ file: rel, line: i + 1, property: m[1] ? 'font-size' : 'font', value: m[2] });
            }
        });
    }
    return found;
}

describe('one type scale (ADR-0008)', () => {
    const tokens = declaredTokens();
    const all = declarations();
    const sizes = all.filter((d) => d.property === 'font-size');
    const shorthands = all.filter((d) => d.property === 'font');
    const where = (d) => `${d.file}:${d.line} ${d.property}: ${d.value}`;

    test('style.css declares the scale', () => {
        expect(tokens).toContain('--font-size-base');
        expect(tokens).toContain('--font-size-small');
        expect(tokens).toContain('--font-size-title');
    });

    test('the components declare a meaningful number of font sizes', () => {
        expect(sizes.length).toBeGreaterThan(200);
    });

    test('every font-size is a token from style.css, or inherit', () => {
        const offenders = sizes.filter((d) => {
            if (d.value === 'inherit') return false;
            const m = /^var\((--font-size-[\w-]+)\)$/.exec(d.value);
            return !m || !tokens.has(m[1]);
        });
        expect(offenders.map(where), 'absolute or unknown font-size (use the tokens of style.css, see ADR-0008)').toEqual([]);
    });

    test('the chrome tokens stay where ADR-0008 places them', () => {
        const offenders = sizes.filter((d) => {
            const m = /^var\((--font-size-[\w-]+)\)$/.exec(d.value);
            const scopes = m && CHROME_TOKEN_SCOPES[m[1]];
            return scopes && !scopes.some((re) => re.test(d.file));
        });
        expect(offenders.map(where), 'chrome token used outside its named exception').toEqual([]);
    });

    test('a font shorthand can only be inherit, and the global rule already says so', () => {
        // style.css sets `font: inherit` on form controls since 2026-08-11; a local copy is
        // redundant, and any other shorthand smuggles a size or family past the scale.
        expect(shorthands.map(where)).toEqual([]);
    });
});
