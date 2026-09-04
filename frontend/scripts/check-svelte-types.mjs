#!/usr/bin/env node
// check-svelte-types.mjs — a non-regressive ceiling on `svelte-check`'s own
// error count, on the same pattern as check-svelte-warnings.mjs (#205) and
// colorTokens.sync.test.js: `jsconfig.json` turns `checkJs: true` on, but
// nothing had ever run `svelte-check` itself in CI (D.13, #214) — so JS type
// errors across the whole frontend, `.svelte` files included, have been
// accumulating with a green build the whole time.
//
// This does not lint anything itself: it shells out to the real `svelte-check`
// (`npm run check`, machine output) and counts the ERROR lines it reports.
// The count must not exceed the ceiling recorded in `.svelte-check-budget`;
// lowering that number is how a future type-error fix locks its own gain in.
// Deliberately a ceiling, not a target of zero: the current backlog (mostly
// implicit-`any` in test helpers and missing `@types/node` for `node:*`
// imports) is real work for a later fiche, not a one-sitting fix.
//
// Usage: node scripts/check-svelte-types.mjs (run from frontend/, as the
// `check:types` npm script does).

import { spawnSync } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const frontendRoot = path.dirname(path.dirname(fileURLToPath(import.meta.url)));
const budgetPath = path.join(frontendRoot, '.svelte-check-budget');

function readBudget() {
    const raw = fs.readFileSync(budgetPath, 'utf8').trim();
    const n = Number(raw);
    if (!Number.isInteger(n) || n < 0) {
        throw new Error(`${budgetPath} must contain a single non-negative integer, found: ${JSON.stringify(raw)}`);
    }
    return n;
}

const result = spawnSync('npx', ['svelte-check', '--tsconfig', './jsconfig.json', '--output', 'machine'], {
    cwd: frontendRoot,
    encoding: 'utf8',
    maxBuffer: 64 * 1024 * 1024
});

const output = result.stdout ?? '';
if (output.trim()) console.log(output.trim());
if (result.stderr?.trim()) console.error(result.stderr.trim());

const errorLines = output.split('\n').filter((line) => / ERROR /.test(line));
const budget = readBudget();

console.log(`\n${errorLines.length} svelte-check error(s), ceiling is ${budget} (.svelte-check-budget).`);

if (errorLines.length > budget) {
    console.error(`\nFAIL: ${errorLines.length} type errors exceed the ${budget}-error ceiling.`);
    console.error('Either fix an error above (and lower .svelte-check-budget to lock the gain in), or, if a new error is unavoidable right now, raise the ceiling deliberately in the same commit.');
    process.exit(1);
}
if (errorLines.length < budget) {
    console.log(`\nNote: only ${errorLines.length} errors remain against a ceiling of ${budget} — consider lowering .svelte-check-budget to ${errorLines.length} to lock this gain in.`);
}
