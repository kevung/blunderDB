#!/usr/bin/env node
// check-svelte-warnings.mjs — a non-regressive ceiling on the Svelte
// compiler's own warnings (a11y roles, stale-closure reactivity bugs, unused
// CSS selectors, …), which `npm run lint`/`format:check` never look at (#205):
// `frontend-lint` in .github/workflows/build.yml only runs ESLint and
// Prettier, so `npm run build` has been quietly emitting 35 warnings — 26 of
// them accessibility — with a green CI the whole time.
//
// This does not lint anything itself: it compiles every `.svelte` file under
// `src/` with the same options `vite.config.js` passes to
// `@sveltejs/vite-plugin-svelte` (`runes: true`, client target) and counts
// the warnings the compiler itself reports — the exact same warnings
// `npm run build` prints to the console, without needing to parse that
// human-oriented, timestamp-prefixed console output. The count must not
// exceed the ceiling recorded in `.svelte-warnings-budget`; lowering that
// number is how a future fix locks its own gain in (measured today: 35 → 20
// after D.5's first batch — the three chart components' frozen legend, a
// same-shaped stale-closure bug in SearchPanel, six redundant `role="region"`
// attributes already implied by `aria-label` on a `<section>`, and five HTML
// `autofocus` attributes replaced by the `autofocus` Svelte action in
// `utils/autofocus.js`). It is deliberately a ceiling, not a target of zero:
// the remaining 20 (keyboard handlers for click-only interactive elements,
// unassociated `<label>`s, a combobox missing ARIA props, a dialog to
// migrate onto `Modal.svelte`, …) are real work for a later fiche, not a
// one-sitting rewrite of a dozen components.
//
// Usage: node scripts/check-svelte-warnings.mjs (run from frontend/, as the
// `check:svelte-warnings` npm script does).

import { compile } from 'svelte/compiler';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const frontendRoot = path.dirname(path.dirname(fileURLToPath(import.meta.url)));
const budgetPath = path.join(frontendRoot, '.svelte-warnings-budget');
const srcDir = path.join(frontendRoot, 'src');

function walk(dir, out = []) {
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
        const full = path.join(dir, entry.name);
        if (entry.isDirectory()) walk(full, out);
        else if (entry.name.endsWith('.svelte')) out.push(full);
    }
    return out;
}

function collectWarnings() {
    const warnings = [];
    for (const file of walk(srcDir)) {
        const source = fs.readFileSync(file, 'utf8');
        const relPath = path.relative(frontendRoot, file);
        const { warnings: fileWarnings } = compile(source, { filename: relPath, runes: true, generate: 'client' });
        for (const w of fileWarnings) {
            warnings.push({ ...w, filename: relPath });
        }
    }
    return warnings;
}

function readBudget() {
    const raw = fs.readFileSync(budgetPath, 'utf8').trim();
    const n = Number(raw);
    if (!Number.isInteger(n) || n < 0) {
        throw new Error(`${budgetPath} must contain a single non-negative integer, found: ${JSON.stringify(raw)}`);
    }
    return n;
}

const warnings = collectWarnings();
const budget = readBudget();

for (const w of warnings) {
    // w.message often embeds a trailing "https://svelte.dev/e/<code>" line;
    // keep only the first line for a compact one-warning-per-line report.
    const firstLine = w.message.split('\n')[0];
    console.log(`${w.filename}:${w.start?.line}:${w.start?.column} ${firstLine} [${w.code}]`);
}
console.log(`\n${warnings.length} Svelte compiler warning(s), ceiling is ${budget} (.svelte-warnings-budget).`);

if (warnings.length > budget) {
    console.error(`\nFAIL: ${warnings.length} warnings exceed the ${budget}-warning ceiling.`);
    console.error('Either fix a warning above (and lower .svelte-warnings-budget to lock the gain in), or, if this new warning is unavoidable, raise the ceiling deliberately in the same commit.');
    process.exit(1);
}
if (warnings.length < budget) {
    console.log(`\nNote: only ${warnings.length} warnings remain against a ceiling of ${budget} — consider lowering .svelte-warnings-budget to ${warnings.length} to lock this gain in.`);
}
