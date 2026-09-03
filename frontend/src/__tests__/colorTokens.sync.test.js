/**
 * colorTokens.sync.test.js
 *
 * ADR-0031: the interface has one colour palette, declared once as tokens in style.css
 * (--color-text, --color-text-muted, --color-border, --color-surface, --color-surface-alt,
 * --color-primary, --color-danger) plus --font-family-ui/-mono and a spacing/radius scale.
 * Unlike ADR-0008 (the type scale, which reached zero unexplained absolute values before its
 * guard test landed), the colour migration is still in progress: 635 hex literals lived in
 * component <style> blocks when this test was written, spread across 38 files — moving every
 * one in a single pass was not attempted (see the ADR's Consequences).
 *
 * So this is a CEILING, on the same pattern as `check-svelte-warnings.mjs` /
 * `.svelte-warnings-budget` (#205): it counts hex colour literals (`#abc`, `#aabbcc`, with
 * optional alpha) inside every component's <style> block — style.css itself is exempt, since
 * that is where the tokens are declared — and fails if that count exceeds the ceiling
 * recorded in `.color-token-budget`. A fix that replaces hex literals with tokens lowers the
 * count and should lower the budget file in the same commit, locking the gain in; a new
 * component may still reach for a hex value today (the migration is progressive), but must
 * not push the total past the recorded ceiling.
 *
 * Board colours are NOT in scope: they are a user preference (boardColorsStore.js, the
 * "Couleurs" tab of Configuration), read from JS, never written as a literal in a <style>
 * block — this test only looks inside <style>, so it cannot see them either way.
 */

import { describe, test, expect } from 'vitest';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const SRC = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const FRONTEND_ROOT = path.resolve(SRC, '..');
const BUDGET_PATH = path.join(FRONTEND_ROOT, '.color-token-budget');

const HEX_COLOR = /#[0-9a-fA-F]{3,8}\b/g;
const STYLE_BLOCK = /<style[^>]*>([\s\S]*?)<\/style>/g;

function svelteFiles(dir) {
    return fs.readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
        const full = path.join(dir, entry.name);
        if (entry.isDirectory()) return entry.name === '__tests__' ? [] : svelteFiles(full);
        return entry.name.endsWith('.svelte') ? [full] : [];
    });
}

// Every hex colour literal found inside a <style> block, with its location. style.css is
// out of scope on purpose: it is where the tokens themselves are declared.
function offenders() {
    const found = [];
    for (const file of svelteFiles(SRC)) {
        const rel = path.relative(SRC, file).split(path.sep).join('/');
        const source = fs.readFileSync(file, 'utf8');
        for (const styleMatch of source.matchAll(STYLE_BLOCK)) {
            const block = styleMatch[1];
            const blockStart = styleMatch.index;
            for (const m of block.matchAll(HEX_COLOR)) {
                const upTo = source.slice(0, blockStart + m.index);
                const line = upTo.split('\n').length;
                found.push({ file: rel, line, value: m[0] });
            }
        }
    }
    return found;
}

function readBudget() {
    const raw = fs.readFileSync(BUDGET_PATH, 'utf8').trim();
    const n = Number(raw);
    if (!Number.isInteger(n) || n < 0) {
        throw new Error(`${BUDGET_PATH} must contain a single non-negative integer, found: ${JSON.stringify(raw)}`);
    }
    return n;
}

describe('one colour palette (ADR-0031)', () => {
    test('style.css declares the palette', () => {
        const css = fs.readFileSync(path.join(SRC, 'style.css'), 'utf8');
        for (const token of [
            '--color-text',
            '--color-text-muted',
            '--color-border',
            '--color-surface',
            '--color-surface-alt',
            '--color-primary',
            '--color-danger',
            '--font-family-ui',
            '--font-family-mono'
        ]) {
            expect(css, `style.css is missing ${token}`).toContain(token);
        }
    });

    test('hex colour literals in component styles stay under the ceiling (.color-token-budget)', () => {
        const found = offenders();
        const budget = readBudget();
        const where = found.map((d) => `${d.file}:${d.line} ${d.value}`);
        const detail =
            found.length > budget
                ? `\n${where.join('\n')}`
                : found.length < budget
                  ? ` (only ${found.length} remain against a ceiling of ${budget} — consider lowering .color-token-budget to ${found.length} to lock this gain in)`
                  : '';

        expect(
            found.length,
            `${found.length} hex colour literal(s) in <style> blocks exceed the ${budget}-ceiling of .color-token-budget. Use the tokens of style.css (ADR-0031), or lower the ceiling deliberately in the same commit if the value truly cannot be a token.${detail}`
        ).toBeLessThanOrEqual(budget);
    });
});
