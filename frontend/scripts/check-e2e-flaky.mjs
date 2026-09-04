#!/usr/bin/env node
// check-e2e-flaky.mjs — fails the job if any Playwright spec was flaky
// (failed at least once, then passed on retry) in the run that just produced
// playwright-report/results.json.
//
// `retries: 2` in playwright.config.js (CI only) means a genuinely broken
// spec can still turn the job green — Playwright reports it as "flaky", not
// "failed", the moment a retry passes. D.13 (#214) wants that treated as a
// real signal instead of silently swallowed: `stats.flaky` in the JSON
// reporter's own output is exactly this count, no need to walk `suites`.
//
// Usage: node scripts/check-e2e-flaky.mjs (run from frontend/, after
// `npm run test:e2e`, as the CI `frontend-e2e` job does).

import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const frontendRoot = path.dirname(path.dirname(fileURLToPath(import.meta.url)));
const reportPath = path.join(frontendRoot, 'playwright-report', 'results.json');

if (!fs.existsSync(reportPath)) {
    console.error(`No report at ${reportPath} — run \`npm run test:e2e\` first.`);
    process.exit(1);
}

const report = JSON.parse(fs.readFileSync(reportPath, 'utf8'));
const flaky = report.stats?.flaky ?? 0;

console.log(`${flaky} flaky spec(s) (playwright-report/results.json).`);

if (flaky > 0) {
    // Walk the suite tree once, only to name the offenders for the log — the
    // pass/fail decision above already used the report's own count.
    function collectFlaky(suites, names = []) {
        for (const suite of suites ?? []) {
            for (const spec of suite.specs ?? []) {
                for (const test of spec.tests ?? []) {
                    if (test.status === 'flaky') names.push(`${suite.title} > ${spec.title}`);
                }
            }
            collectFlaky(suite.suites, names);
        }
        return names;
    }
    for (const name of collectFlaky(report.suites)) console.error(`  flaky: ${name}`);
    console.error(`\nFAIL: ${flaky} spec(s) needed a retry to pass. A flaky spec hides a real bug or a real race — fix it, don't just let the retry absorb it.`);
    process.exit(1);
}
