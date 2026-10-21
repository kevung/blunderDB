#!/usr/bin/env node
/* eslint-disable no-console -- a command-line script: its output is the table it prints */
/**
 * Time to a usable window, measured on the production bundle (`npm run build` first).
 *
 * Serves each build with `vite preview`, opens it in headless Chromium behind the e2e
 * Wails mock (no database: the home screen), and reads the `bdb:*` marks of
 * src/utils/startupMarks.js. Prints the median of each milestone over the runs, and
 * the JavaScript fetched before `bdb:ready`. Several builds (`--dist a,b`) are measured
 * in alternation, so a busy machine slows them alike and the comparison holds.
 *
 *   node scripts/measure-startup.mjs [--dist dist,../other/dist] [--runs 9] [--cpu 4] [--lang fr]
 *
 * `--cpu` throttles the CPU (Chromium's DevTools rate): the WebKitGTK WebView of a
 * modest laptop parses and runs JavaScript well below a desktop Chromium.
 *
 * The builds are served from BLUNDERDB_E2E_PORT upwards (default 4179), one port each:
 * a port held by another worktree's server would measure that server's bundle.
 */
import { spawn } from 'node:child_process';
import { chromium } from '@playwright/test';
import { installWailsMock } from '../tests/e2e/helpers/wailsMock.js';

const args = Object.fromEntries(
    process.argv
        .slice(2)
        .join(' ')
        .split('--')
        .filter(Boolean)
        .map((kv) => kv.trim().split(/\s+/))
);
const runs = Number(args.runs ?? 9);
const cpu = Number(args.cpu ?? 1);
const lang = args.lang ?? 'fr';
const dists = (args.dist ?? 'dist').split(',');
const basePort = Number(process.env.BLUNDERDB_E2E_PORT || 4179);

/** @param {string} dist @param {number} port */
async function serve(dist, port) {
    const server = spawn('npx', ['vite', 'preview', '--outDir', dist, '--port', String(port), '--strictPort'], { stdio: 'pipe' });
    await new Promise((resolve, reject) => {
        server.stdout.on('data', (d) => String(d).includes(String(port)) && resolve(undefined));
        server.on('exit', (code) => reject(new Error(`vite preview exited (${code})`)));
    });
    return server;
}

const servers = await Promise.all(dists.map((dist, i) => serve(dist, basePort + i)));
const browser = await chromium.launch();
/** @type {Record<string, number>[][]} */
const samples = dists.map(() => []);
try {
    for (let i = 0; i < runs; i++) {
        for (let d = 0; d < dists.length; d++) {
            const context = await browser.newContext({ viewport: { width: 1280, height: 800 } });
            const page = await context.newPage();
            if (cpu > 1) {
                const cdp = await context.newCDPSession(page);
                await cdp.send('Emulation.setCPUThrottlingRate', { rate: cpu });
            }
            await installWailsMock(page, { config: { GetLanguage: lang } });
            await page.goto(`http://localhost:${basePort + d}/`);
            await page.waitForFunction(() => performance.getEntriesByName('bdb:ready').length > 0, null, { timeout: 60000 });
            // The paint entries land a frame or two after the marks.
            await page.waitForFunction(() => performance.getEntriesByName('first-contentful-paint').length > 0, null, { timeout: 60000 });
            samples[d].push(
                await page.evaluate(() => {
                    const at = (/** @type {string} */ n) => performance.getEntriesByName(n)[0]?.startTime ?? NaN;
                    const ready = at('bdb:ready');
                    const scripts = performance
                        .getEntriesByType('resource')
                        .filter((r) => r.name.endsWith('.js') && r.startTime < ready)
                        .reduce((s, r) => s + /** @type {PerformanceResourceTiming} */ (r).decodedBodySize, 0);
                    return {
                        'modules evaluated': at('bdb:modules-evaluated'),
                        'App mounted': at('bdb:mounted'),
                        'first contentful paint': at('first-contentful-paint'),
                        'locale applied': at('bdb:language'),
                        'ready (home screen usable)': ready,
                        'JS before ready (kB)': scripts / 1000
                    };
                })
            );
            await context.close();
        }
    }
} finally {
    await browser.close();
    servers.forEach((s) => s.kill());
}

const median = (/** @type {number[]} */ xs) => {
    const s = [...xs].sort((a, b) => a - b);
    return s[Math.floor(s.length / 2)];
};
console.log(`| milestone (median of ${runs}, CPU ×${cpu}, ${lang}) | ${dists.join(' | ')} |`);
console.log(`|---|${dists.map(() => '---:').join('|')}|`);
for (const key of Object.keys(samples[0][0])) {
    console.log(`| ${key} | ${samples.map((runsOf) => median(runsOf.map((s) => s[key])).toFixed(0)).join(' | ')} |`);
}
