#!/usr/bin/env node
/**
 * Rendering cost of the frontend on a large database, backend excluded (`npm run build` first;
 * `vite build --sourcemap` for the profiles to name source lines).
 *
 * Serves the build with `vite preview`, opens it in headless Chromium behind the e2e Wails mock
 * extended to the volumes of a large library (20 000 positions, 1 000 matches with badges,
 * 60 tournaments, 200 collections, one of 2 000 positions, 50 Anki decks, a search of 5 000
 * results), and times each interaction from the ready state, inside the page: from the action
 * to the end of the frame that follows the last DOM mutation (or the last frame later than
 * 50 ms), once the page has been quiet for a while. Each run is a fresh page. Prints, per
 * scenario, the median over the runs of that time, of the long tasks it contains, of the
 * script / layout / style time Chromium reports (CDP Performance metrics), and the DOM node
 * count after it.
 *
 * The step scenarios (j/k, scrolling) time each step instead, from the key or the scroll to
 * the forced layout once Svelte has flushed, and report the median and p95 of a step and the
 * ratio of the last quarter of the steps to the first (flat when the list is virtualised).
 *
 *   node scripts/measure-render.mjs [--dist dist] [--runs 5] [--cpu 1] [--port 4199]
 *        [--only regex] [--profile 5] [--json out.json]
 *
 * `--profile N` profiles once more the N scenarios of highest main-thread time (CDP Profiler, 100 µs sampling)
 * and aggregates the samples by source function (through the build's source maps when
 * present) and by enclosing src/ file. Chromium is not the WebKitGTK of the real app: the
 * ranking holds, the absolute times do not (`--cpu 4` approaches a modest laptop).
 */
/* eslint-disable no-console -- a command-line report, printed to stdout */
import { spawn } from 'node:child_process';
import { readFileSync, existsSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { chromium } from '@playwright/test';
import { SourceMapConsumer } from 'source-map-js';
import { installWailsMock } from '../tests/e2e/helpers/wailsMock.js';
import { openLibraryMock } from '../tests/e2e/helpers/fixtures.js';
import { showcaseLibrary, showcaseAnalysis, showcaseGalleryMock } from '../tests/e2e/helpers/showcase.js';

const args = Object.fromEntries(
    process.argv
        .slice(2)
        .join(' ')
        .split('--')
        .filter(Boolean)
        .map((kv) => {
            const [k, ...v] = kv.trim().split(/\s+/);
            return [k, v.join(' ') || 'true'];
        })
);
const runs = Number(args.runs ?? 5);
const cpu = Number(args.cpu ?? 1);
const port = Number(args.port ?? 4199);
const dist = args.dist ?? 'dist';
const only = args.only ? new RegExp(args.only) : null;
const profileTop = Number(args.profile ?? 0);
if (port === 5173) throw new Error('5173 is shared with the other worktrees and gammonGo: pick another port');

const VOLUME = { positions: 20000, matches: 1000, tournaments: 60, collections: 200, collectionRows: 2000, ankiDecks: 50, searchResults: 5000 };

// ── Mock of a large library ────────────────────────────────────────────────────

/** Overrides that installWailsMock serialises: the small constants. The volumes are generated in the page. */
function baseOverrides() {
    const gallery = showcaseGalleryMock().database;
    const keep = ['ComputeStats', 'ComputeStudyPlan', 'GetPlayerTable', 'GetAllPlayerNames', 'GetStatsAnalysisEngines', 'GetStatsDateRange', 'GetCommentsByPosition'];
    return openLibraryMock({
        config: { GetLanguage: 'en', GetPanelPosition: 'bottom', GetTabPanelHeights: { '*': 320 } },
        database: Object.fromEntries(keep.map((k) => [k, gallery[k]]))
    });
}

/** Runs in the page, after installWailsMock: replaces the volume bindings by generators. */
function installVolumes({ volume, templates, analysis }) {
    const db = window.go.database.Database;
    const names = ['Alice', 'Bob', 'Carla', 'Dmitri', 'Eve', 'Farid', 'Gita', 'Hugo', 'Ines', 'Jonas', 'Kaori', 'Lars', 'Mira', 'Nils', 'Olga', 'Pavel', 'Quinn', 'Rosa', 'Sven', 'Tariq'];
    const mwc7 = (i) => ({
        available: true,
        loss: 0.02 + (i % 17) / 400,
        has_interval: true,
        low: 0.01,
        high: 0.06,
        elo: 1700 + (i % 300),
        elo_low: 1600,
        elo_high: 1900,
        elo_floored: false,
        matches: 1
    });
    const position = (id) => ({ ...templates[id % templates.length], id });
    const ids = (n, from = 1, stride = 1) => Array.from({ length: n }, (_, i) => from + i * stride);
    const slice = (list, offset = 0, limit = 0) => list.slice(offset, limit > 0 ? offset + limit : undefined);

    const library = ids(volume.positions);
    db.CountPositions = async () => library.length;
    db.ListPositionIDs = async (offset, limit) => slice(library, offset, limit);
    // Constant-time answers: the mock's own cost must not count in the page's time.
    db.IndexOfPosition = async (id) => (id >= 1 && id <= volume.positions ? id - 1 : -1);
    db.LoadPositionsByIDs = async (want) => (want || []).filter((id) => id >= 1 && id <= volume.positions).map(position);
    db.LoadPosition = async (id) => position(id);
    db.LoadAnalysis = async (id) => ({ ...analysis, positionId: id });
    db.LoadPositionView = async (id) => ({ analysis: { ...analysis, positionId: id }, comment: id % 3 ? '' : 'Keep the 5-point.' });

    const found = ids(volume.searchResults, 1, 4);
    db.CountPositionsByFilters = async () => found.length;
    db.SearchPositionIDs = async (_q, offset, limit) => slice(found, offset, limit);
    db.IndexOfPositionByFilters = async (_q, id) => ((id - 1) % 4 === 0 && (id - 1) / 4 < found.length ? (id - 1) / 4 : -1);

    const tournaments = ids(volume.tournaments).map((id) => ({
        id,
        name: `Open ${id}`,
        date: `2025-${String((id % 12) + 1).padStart(2, '0')}-14`,
        effectiveDate: `2025-${String((id % 12) + 1).padStart(2, '0')}-14`,
        location: ['Paris', 'Lyon', 'Monaco', 'Berlin'][id % 4],
        sortOrder: id,
        matchCount: Math.round(volume.matches / volume.tournaments),
        comment: '',
        pr: 3 + (id % 9) / 2,
        mwc_loss: 0.02 + (id % 7) / 300,
        ref_player: 'Alice',
        mwc7: mwc7(id)
    }));
    const matches = ids(volume.matches).map((id) => {
        const t = tournaments[id % tournaments.length];
        return {
            id,
            player1_name: names[id % names.length],
            player2_name: names[(id * 7 + 3) % names.length],
            event: t.name,
            location: t.location,
            round: `R${(id % 7) + 1}`,
            match_length: [5, 7, 9, 11][id % 4],
            match_date: `2025-${String((id % 12) + 1).padStart(2, '0')}-${String((id % 28) + 1).padStart(2, '0')}T00:00:00Z`,
            import_date: '2026-01-16T00:00:00Z',
            file_path: `/data/matches/m${id}.xg`,
            game_count: 3 + (id % 9),
            tournament_id: t.id,
            tournament_name: t.name,
            last_visited_position: 0,
            comment: '',
            tournament_sort_order: id,
            pr: 2 + (id % 23) / 3,
            mwc_loss: 0.01 + (id % 13) / 400,
            pr2: 3 + (id % 19) / 3,
            mwc_loss2: 0.012 + (id % 11) / 400,
            mwc7: mwc7(id),
            mwc7_p2: mwc7(id + 5)
        };
    });
    const SORT = { player1: 'player1_name', player2: 'player2_name', length: 'match_length', tournament: 'tournament_name', date: 'match_date' };
    const listed = (o = {}) => {
        let rows = matches;
        const text = (o.Text || '').toLowerCase();
        if (text) rows = rows.filter((m) => `${m.player1_name} ${m.player2_name} ${m.event}`.toLowerCase().includes(text));
        if (Array.isArray(o.IDs)) rows = rows.filter((m) => o.IDs.includes(m.id));
        const key = (o.Sort || '').replace(/_(asc|desc)$/, '');
        const field = SORT[key] || (key.startsWith('date') ? 'match_date' : null);
        if (field) {
            const dir = /_desc$/.test(o.Sort) ? -1 : 1;
            rows = [...rows].sort((a, b) => (a[field] < b[field] ? -dir : a[field] > b[field] ? dir : a.id - b.id));
        }
        return rows;
    };
    db.ListMatches = async (o = {}) => slice(listed(o), o.Offset, o.Limit);
    db.CountMatches = async (o = {}) => listed(o).length;
    db.GetAllMatches = async () => matches;
    db.GetMatchByID = async (id) => matches.find((m) => m.id === id) ?? null;
    db.GetAllTournaments = async () => tournaments;
    db.GetTournamentMatches = async (id) => matches.filter((m) => m.tournament_id === id);

    const collections = ids(volume.collections).map((id) => ({
        id,
        name: `Collection ${id}`,
        description: id % 3 ? `Positions to review, set ${id}` : '',
        sortOrder: id,
        createdAt: '2026-01-05T11:15:00Z',
        updatedAt: '2026-03-10T09:00:00Z',
        positionCount: id === 1 ? volume.collectionRows : 5 + (id % 40),
        filterQuery: ''
    }));
    const ownedCache = new Map();
    const owned = (id) => ownedCache.get(id) ?? ownedCache.set(id, ownedOf(id)).get(id);
    const ownedOf = (id) => ids(collections[id - 1]?.positionCount ?? 0, id, 7).map((x) => ((x - 1) % volume.positions) + 1);
    db.GetAllCollections = async () => collections;
    db.GetCollectionByID = async (id) => collections[id - 1] ?? null;
    db.CountCollectionPositions = async (id) => collections[id - 1]?.positionCount ?? 0;
    db.ListCollectionPositionIDs = async (id, offset, limit) => slice(owned(id), offset, limit);
    const ownedIndex = new Map();
    db.IndexOfCollectionPosition = async (id, pos) => {
        if (!ownedIndex.has(id)) ownedIndex.set(id, new Map(owned(id).map((p, i) => [p, i])));
        return ownedIndex.get(id).get(pos) ?? -1;
    };
    db.GetCollectionPositions = async (id) => owned(id).map(position);

    const decks = ids(volume.ankiDecks).map((id) => ({
        id,
        name: `Deck ${id}`,
        description: id % 2 ? 'Checker plays' : '',
        sourceType: 'collection',
        sourceId: id,
        sourceCommand: '',
        requestRetention: 0.9,
        maximumInterval: 365,
        enableFuzz: true,
        cardCount: 20 + (id % 60),
        dueCount: id % 9,
        newCount: id % 4,
        createdAt: '2026-01-05T11:15:00Z',
        updatedAt: '2026-03-10T09:00:00Z'
    }));
    db.GetAllAnkiDecks = async () => decks;
    db.ListLessons = async () => ids(12).map((id) => ({ id, name: `Lesson ${id}`, stepCount: 8 }));
    db.GetLesson = async (id) => ({
        id,
        name: `Lesson ${id}`,
        steps: ids(8).map((s) => ({ id: s, lessonId: id, title: `Step ${s}`, text: `<p>Why the <b>${s}-point</b> matters here.</p>`, collectionId: 0, positionId: s * 11 }))
    });
    db.LessonDoneSteps = async () => [];
}

/** Runs in the page before any script: the long-task log and the timing helpers. */
function installBench() {
    /** @type {{startTime: number, duration: number}[]} */
    const longTasks = [];
    try {
        new PerformanceObserver((list) => longTasks.push(...list.getEntries().map((e) => ({ startTime: e.startTime, duration: e.duration })))).observe({ type: 'longtask', buffered: true });
    } catch {
        /* no long-task API */
    }
    const frame = () => new Promise((r) => requestAnimationFrame(() => r(undefined)));
    const task = () =>
        new Promise((r) => {
            const ch = new MessageChannel();
            ch.port1.onmessage = () => r(performance.now());
            ch.port2.postMessage(0);
        });
    const afterFrame = () => frame().then(task);
    const nodes = () => document.getElementsByTagName('*').length;
    const tasksIn = (/** @type {number} */ t0, /** @type {number} */ t1) => {
        const inside = longTasks.filter((t) => t.startTime + t.duration > t0 && t.startTime < t1);
        return { longTasks: inside.length, longTaskMs: inside.reduce((s, t) => s + t.duration, 0) };
    };

    /**
     * Time an action to the end of the frame after the last DOM mutation (or the last frame
     * later than 50 ms), once nothing moved for `quiet` ms and `until()` holds.
     * @param {() => any} act
     * @param {{quiet?: number, max?: number, until?: () => boolean}} [opts]
     */
    async function run(act, { quiet = 300, max = 10000, until } = {}) {
        await afterFrame();
        let last = performance.now();
        let end = last;
        const touch = () => {
            last = performance.now();
            afterFrame().then((t) => (end = Math.max(end, t)));
        };
        const observer = new MutationObserver(touch);
        observer.observe(document, { subtree: true, childList: true, attributes: true, characterData: true });
        let watching = true;
        let prev = performance.now();
        const watch = () => {
            const now = performance.now();
            if (now - prev > 50) {
                last = now;
                end = Math.max(end, now);
            }
            prev = now;
            if (watching) requestAnimationFrame(watch);
        };
        requestAnimationFrame(watch);
        const t0 = performance.now();
        end = t0;
        await act();
        touch();
        let capped = false;
        for (;;) {
            await new Promise((r) => setTimeout(r, 40));
            const now = performance.now();
            if (now - t0 > max) {
                capped = true;
                break;
            }
            if (now - last >= quiet && (!until || until())) break;
        }
        watching = false;
        observer.disconnect();
        return { ms: end - t0, ...tasksIn(t0, end), nodes: nodes(), capped };
    }

    /**
     * Time each of `n` steps as `run` times an action, with a shorter quiet window.
     * @param {(i: number) => any} step
     * @param {number} n
     * @param {{quiet?: number}} [opts]
     */
    async function steps(step, n, { quiet = 60 } = {}) {
        const times = [];
        const nodesSeen = [];
        const t0 = performance.now();
        for (let i = 0; i < n; i++) {
            const r = await run(() => step(i), { quiet });
            times.push(r.ms);
            nodesSeen.push(r.nodes);
        }
        const t1 = performance.now();
        return { times, n, nodesFirst: nodesSeen[0], nodesLast: nodesSeen[nodesSeen.length - 1], ...tasksIn(t0, t1), totalMs: t1 - t0, status: status() };
    }

    const key = (/** @type {string} */ k, extra = {}) => {
        const target = document.activeElement ?? document.body;
        // Letters are matched on `key`, digits and Space on `code` (src/utils/keys.js).
        const code = k === ' ' ? 'Space' : /^[a-z]$/i.test(k) ? `Key${k.toUpperCase()}` : /^[0-9]$/.test(k) ? `Digit${k}` : k === '?' ? 'Slash' : k;
        extra = { code, ...extra };
        target.dispatchEvent(new KeyboardEvent('keydown', { key: k, bubbles: true, cancelable: true, ...extra }));
        target.dispatchEvent(new KeyboardEvent('keyup', { key: k, bubbles: true, cancelable: true, ...extra }));
    };
    const $ = (/** @type {string} */ s) => /** @type {HTMLElement | null} */ (document.querySelector(s));
    const click = (/** @type {string} */ s) => {
        const el = $(s);
        if (!el) throw new Error(`no ${s}`);
        // The tabs act on mousedown/mouseup (they can be dragged): the whole sequence, as a pointer gives it.
        const r = el.getBoundingClientRect();
        const at = { bubbles: true, cancelable: true, button: 0, clientX: r.x + r.width / 2, clientY: r.y + r.height / 2 };
        el.dispatchEvent(new MouseEvent('mousedown', at));
        el.dispatchEvent(new MouseEvent('mouseup', at));
        el.click();
    };
    const status = () => $('[data-testid="status-bar"]')?.textContent ?? '';
    /** @param {() => boolean} cond */
    const until = async (cond, max = 10000) => {
        const t0 = performance.now();
        while (!cond()) {
            if (performance.now() - t0 > max) throw new Error(`timeout waiting for ${cond}`);
            await frame();
        }
    };
    /** Resolves once no resource was fetched for `ms` (the idle preload of the deferred chunks). */
    const resourcesQuiet = async (ms = 600) => {
        let count = -1;
        let since = performance.now();
        for (;;) {
            const now = performance.getEntriesByType('resource').length;
            if (now !== count) {
                count = now;
                since = performance.now();
            } else if (performance.now() - since > ms) return;
            await new Promise((r) => setTimeout(r, 50));
        }
    };
    // @ts-ignore
    window.__bench = { run, steps, key, click, $, status, until, frame, afterFrame, task, resourcesQuiet, nodes };
}

// ── Scenarios ──────────────────────────────────────────────────────────────────

const N = VOLUME.positions;
/** In the page: wait for a tab to be active. */
const tabActive = (/** @type {string} */ id) => `document.querySelector('[data-testid="tab-${id}"]')?.classList.contains('active')`;

/**
 * Each scenario: `prepare(page)` brings the page from ready to the starting state (not timed),
 * `measure(page)` returns one sample, timed in the page.
 * @type {{name: string, kind?: 'steps', prepare?: (page: import('@playwright/test').Page) => Promise<void>, measure: (page: import('@playwright/test').Page) => Promise<any>}[]}
 */
const SCENARIOS = [
    { name: 'témoin (aucune action)', measure: (page) => page.evaluate(() => window.__bench.run(() => {})) },
    ...['matches', 'tournaments', 'collections', 'search', 'analysis', 'comments', 'anki', 'training', 'stats', 'eval'].map((id) => ({
        name: `ouvrir onglet ${id}`,
        prepare: async (/** @type {import('@playwright/test').Page} */ page) => {
            // From the board with the Matches tab closed, except for the Matches tab itself, opened from Analysis.
            const from = id === 'analysis' ? 'comments' : 'analysis';
            await page.evaluate((f) => window.__bench.click(`[data-testid="tab-${f}"]`), from);
            await page.waitForFunction(tabActive(from));
            await page.evaluate(() => window.__bench.resourcesQuiet(300));
        },
        measure: (/** @type {import('@playwright/test').Page} */ page) =>
            page.evaluate((tab) => window.__bench.run(() => window.__bench.click(`[data-testid="tab-${tab}"]`), { until: () => !!document.querySelector(`[data-testid="tab-${tab}"].active`) }), id)
    })),
    // A tab switch back and forth: the target was visited before, so a panel kept mounted only unhides.
    ...[
        ['analysis', 'matches'],
        ['matches', 'analysis']
    ].map(([from, to]) => ({
        name: `bascule ${from} → ${to} (déjà visité)`,
        prepare: async (/** @type {import('@playwright/test').Page} */ page) => {
            for (const id of [to, from]) {
                await page.evaluate((f) => window.__bench.click(`[data-testid="tab-${f}"]`), id);
                await page.waitForFunction(tabActive(id));
                await page.evaluate(() => window.__bench.resourcesQuiet(300));
            }
        },
        measure: (/** @type {import('@playwright/test').Page} */ page) =>
            page.evaluate((tab) => window.__bench.run(() => window.__bench.click(`[data-testid="tab-${tab}"]`), { until: () => !!document.querySelector(`[data-testid="tab-${tab}"].active`) }), to)
    })),
    {
        name: 'ouvrir leçon (le 1)',
        prepare: async (page) => {
            await page.evaluate(() => window.__bench.key('Escape'));
        },
        measure: (page) =>
            page.evaluate(async () => {
                const b = window.__bench;
                return b.run(
                    async () => {
                        b.key(' ');
                        await b.until(() => !!document.querySelector('input.command-input, [data-testid="command-input"]'));
                        const input = /** @type {HTMLInputElement} */ (document.querySelector('input.command-input, [data-testid="command-input"]'));
                        input.value = 'le 1';
                        input.dispatchEvent(new Event('input', { bubbles: true }));
                        input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', code: 'Enter', bubbles: true }));
                    },
                    { until: () => !!document.querySelector('.lesson-bar') }
                );
            })
    },
    {
        name: 'j/k : 50 pas (plateau + analyse)',
        kind: 'steps',
        prepare: async (page) => {
            await page.evaluate(() => window.__bench.click('[data-testid="tab-analysis"]'));
            await page.waitForFunction(tabActive('analysis'));
            await page.evaluate(() => window.__bench.key('h'));
            await page.waitForFunction(() => /\b1 \/ 20000/.test(window.__bench.status()));
        },
        measure: (page) =>
            page.evaluate(async () => {
                const b = window.__bench;
                return b.steps(
                    (i) => {
                        b.key(i % 10 === 9 ? 'k' : 'j');
                    },
                    50,
                    { quiet: 120 }
                );
            })
    },
    {
        name: 'défiler matchs (1 000 lignes)',
        kind: 'steps',
        prepare: async (page) => {
            await page.waitForFunction(tabActive('matches'));
            await page.waitForFunction(() => document.querySelectorAll('.match-list-pane .panel-table tbody tr').length > 5);
        },
        measure: (page) =>
            page.evaluate(async () => {
                const b = window.__bench;
                const el = /** @type {HTMLElement} */ (document.querySelector('.match-list-pane .panel-table .scroll'));
                const rows = 1000;
                const n = 100;
                const per = (rows * 28) / n;
                return b.steps(
                    () => {
                        el.scrollTop += per;
                        el.dispatchEvent(new Event('scroll'));
                    },
                    n,
                    { quiet: 40 }
                );
            })
    },
    {
        name: 'ouvrir une collection (2 000 positions)',
        prepare: async (page) => {
            await page.evaluate(() => window.__bench.click('[data-testid="tab-collections"]'));
            await page.waitForFunction(tabActive('collections'));
            await page.waitForFunction(() => document.querySelectorAll('.panel-table tbody tr').length > 5);
            await page.evaluate(() => window.__bench.run(() => {}));
        },
        measure: (page) =>
            page.evaluate(() =>
                window.__bench.run(
                    () => {
                        const row = /** @type {HTMLElement} */ (document.querySelector('.panel-table tbody tr'));
                        row.dispatchEvent(new MouseEvent('click', { bubbles: true, detail: 1 }));
                        row.dispatchEvent(new MouseEvent('dblclick', { bubbles: true, detail: 2 }));
                    },
                    { until: () => document.querySelectorAll('.id-cell').length > 5 }
                )
            )
    },
    {
        name: 'défiler positions de collection (2 000 lignes)',
        kind: 'steps',
        prepare: async (page) => {
            await openCollection(page);
        },
        measure: (page) =>
            page.evaluate(async () => {
                const b = window.__bench;
                const tables = document.querySelectorAll('.panel-table .scroll');
                const el = /** @type {HTMLElement} */ (tables[tables.length - 1]);
                const rows = 2000;
                const n = 100;
                const per = (rows * 28) / n;
                return b.steps(
                    async () => {
                        el.scrollTop += per;
                        el.dispatchEvent(new Event('scroll'));
                    },
                    n,
                    { quiet: 40 }
                );
            })
    },
    {
        name: 'trier matchs (colonne joueur 1)',
        prepare: async (page) => {
            await page.waitForFunction(tabActive('matches'));
            await page.waitForFunction(() => document.querySelectorAll('.match-list-pane .panel-table tbody tr').length > 5);
        },
        measure: (page) => page.evaluate(() => window.__bench.run(() => /** @type {HTMLElement} */ (document.querySelectorAll('.match-list-pane th .sort-btn')[1])?.click()))
    },
    {
        name: 'filtrer matchs (« Alice », attente 250 ms comprise)',
        prepare: async (page) => {
            await page.waitForFunction(tabActive('matches'));
            await page.waitForFunction(() => document.querySelectorAll('.match-list-pane .panel-table tbody tr').length > 5);
        },
        measure: (page) =>
            page.evaluate(() =>
                window.__bench.run(() => {
                    const input = /** @type {HTMLInputElement} */ (document.querySelector('input.match-filter'));
                    input.value = 'Alice';
                    input.dispatchEvent(new Event('input', { bubbles: true }));
                })
            )
    },
    {
        name: 'recherche : 5 000 résultats',
        prepare: async (page) => {
            await page.evaluate(() => window.__bench.click('[data-testid="tab-search"]'));
            await page.waitForFunction(tabActive('search'));
            await page.waitForSelector('[data-testid="search-run"]');
            await page.evaluate(() => window.__bench.resourcesQuiet(300));
        },
        measure: (page) => page.evaluate(() => window.__bench.run(() => window.__bench.click('[data-testid="search-run"]'), { until: () => /\/ 5000/.test(window.__bench.status()) }))
    },
    {
        name: 'recherche : page suivante (PageDown)',
        prepare: async (page) => {
            await page.evaluate(() => window.__bench.click('[data-testid="tab-search"]'));
            await page.waitForSelector('[data-testid="search-run"]');
            await page.evaluate(() => window.__bench.click('[data-testid="search-run"]'));
            await page.waitForFunction(() => /\b1 \/ 5000/.test(window.__bench.status()));
            await page.evaluate(() => window.__bench.run(() => {}));
        },
        measure: (page) => page.evaluate(() => window.__bench.run(() => window.__bench.key('PageDown'), { until: () => /\b101 \/ 5000/.test(window.__bench.status()) }))
    },
    {
        name: 'ouvrir configuration',
        measure: (page) => page.evaluate(() => window.__bench.run(() => window.__bench.click('button[aria-label="Settings"]'), { until: () => !!document.querySelector('.modal-box') }))
    },
    {
        name: 'fermer configuration',
        prepare: async (page) => {
            await page.evaluate(() => window.__bench.click('button[aria-label="Settings"]'));
            await page.waitForSelector('.modal-box');
            await page.evaluate(() => window.__bench.run(() => {}));
        },
        measure: (page) => page.evaluate(() => window.__bench.run(() => window.__bench.key('Escape'), { until: () => !document.querySelector('.modal-box') }))
    },
    {
        name: 'ouvrir aide',
        prepare: async (page) => {
            await page.evaluate(() => window.__bench.key('Escape'));
        },
        measure: (page) => page.evaluate(() => window.__bench.run(() => window.__bench.key('?'), { until: () => !!document.querySelector('.modal-box') }))
    },
    {
        name: 'fermer aide',
        prepare: async (page) => {
            await page.evaluate(() => window.__bench.key('Escape'));
            await page.evaluate(() => window.__bench.key('?'));
            await page.waitForSelector('.modal-box');
            await page.evaluate(() => window.__bench.run(() => {}));
        },
        measure: (page) => page.evaluate(() => window.__bench.run(() => window.__bench.key('Escape'), { until: () => !document.querySelector('.modal-box') }))
    },
    {
        name: 'rouvrir aide (seconde ouverture)',
        prepare: async (page) => {
            await page.evaluate(() => window.__bench.key('Escape'));
            await page.evaluate(() => window.__bench.key('?'));
            await page.waitForSelector('.modal-box');
            await page.evaluate(() => window.__bench.run(() => {}));
            await page.evaluate(() => window.__bench.key('Escape'));
            await page.waitForFunction(() => !document.querySelector('.modal-box'));
            await page.evaluate(() => window.__bench.run(() => {}));
        },
        measure: (page) => page.evaluate(() => window.__bench.run(() => window.__bench.key('?'), { until: () => !!document.querySelector('.modal-box') }))
    },
    {
        name: 'changer de langue (en → fr)',
        prepare: (page) => openSetting(page, '#config-language'),
        measure: (page) => page.evaluate(() => window.__bench.run(() => window.changeSelect('#config-language', 'fr')))
    },
    {
        name: 'changer de thème (→ dark)',
        prepare: (page) => openSetting(page, '#config-theme'),
        measure: (page) => page.evaluate(() => window.__bench.run(() => window.changeSelect('#config-theme', 'dark')))
    },
    {
        name: 'redimensionner la fenêtre (1280×800 → 1100×700)',
        prepare: async (page) => {
            await page.evaluate(() => window.__bench.key('Escape'));
        },
        measure: async (page) => {
            // The resize comes from outside: t0 is the resize event, read in the page.
            const pending = page.evaluate(
                () =>
                    new Promise((resolve) => {
                        window.addEventListener(
                            'resize',
                            (e) => {
                                const lag = performance.now() - e.timeStamp;
                                window.__bench.run(() => {}).then((r) => resolve({ ...r, ms: r.ms + lag }));
                            },
                            { once: true }
                        );
                    })
            );
            await page.evaluate(() => window.__bench.frame());
            await page.setViewportSize({ width: 1100, height: 700 });
            return pending;
        }
    }
];

/** @param {import('@playwright/test').Page} page */
async function openCollection(page) {
    await page.evaluate(() => window.__bench.click('[data-testid="tab-collections"]'));
    await page.waitForFunction(tabActive('collections'));
    await page.waitForFunction(() => document.querySelectorAll('.panel-table tbody tr').length > 5);
    // A double click on the first collection opens its positions.
    const first = page.locator('.panel-table tbody tr').first();
    await first.dblclick();
    await page.waitForFunction(() => {
        const tables = document.querySelectorAll('.panel-table');
        return tables.length > 0 && /\b500\b|2000/.test(document.body.textContent ?? '') && document.querySelectorAll('.id-cell').length > 5;
    });
}

/** @param {import('@playwright/test').Page} page @param {string} selector */
async function openSetting(page, selector) {
    await page.evaluate(() => window.__bench.click('button[aria-label="Settings"]'));
    await page.waitForSelector('.modal-box');
    for (const tab of await page.locator('.modal-box [role="tab"]').all()) {
        if (await page.locator(selector).count()) break;
        await tab.click();
    }
    await page.waitForSelector(selector);
    await page.evaluate(() => window.__bench.run(() => {}));
}

// ── Running ────────────────────────────────────────────────────────────────────

async function serve() {
    const server = spawn('npx', ['vite', 'preview', '--outDir', dist, '--port', String(port), '--strictPort'], { stdio: 'pipe' });
    await new Promise((resolve, reject) => {
        server.stdout.on('data', (d) => String(d).includes(String(port)) && resolve(undefined));
        server.on('exit', (code) => reject(new Error(`vite preview exited (${code})`)));
    });
    return server;
}

/** @param {import('@playwright/test').Browser} browser */
async function readyPage(browser) {
    const context = await browser.newContext({ viewport: { width: 1280, height: 800 } });
    const page = await context.newPage();
    page.setDefaultTimeout(15000);
    const cdp = await context.newCDPSession(page);
    if (cpu > 1) await cdp.send('Emulation.setCPUThrottlingRate', { rate: cpu });
    await cdp.send('Performance.enable');
    await page.addInitScript(installBench);
    await installWailsMock(page, baseOverrides());
    await page.addInitScript(installVolumes, { volume: VOLUME, templates: showcaseLibrary, analysis: showcaseAnalysis });
    await page.addInitScript(() => {
        // @ts-ignore
        window.changeSelect = (/** @type {string} */ selector, /** @type {string} */ value) => {
            const select = /** @type {HTMLSelectElement} */ (document.querySelector(selector));
            select.value = value;
            select.dispatchEvent(new Event('change', { bubbles: true }));
        };
    });
    page.on('pageerror', (e) => console.error('page error:', e.message));
    await page.goto(`http://localhost:${port}/`);
    await page.waitForFunction(() => performance.getEntriesByName('bdb:ready').length > 0, null, { timeout: 60000 });
    await page.waitForFunction((n) => new RegExp(`${n} / ${n}`).test(window.__bench.status()), N, { timeout: 60000 });
    await page.evaluate(() => window.__bench.resourcesQuiet());
    await page.evaluate(() => window.__bench.run(() => {}));
    return { context, page, cdp };
}

/** @param {import('@playwright/test').CDPSession} cdp */
async function metrics(cdp) {
    const { metrics: list } = await cdp.send('Performance.getMetrics');
    return Object.fromEntries(list.map((m) => [m.name, m.value]));
}

const median = (/** @type {number[]} */ xs) => {
    const s = xs.filter((x) => Number.isFinite(x)).sort((a, b) => a - b);
    return s.length ? s[Math.floor((s.length - 1) / 2)] : NaN;
};
const pct = (/** @type {number[]} */ xs, /** @type {number} */ p) => {
    const s = [...xs].sort((a, b) => a - b);
    return s[Math.min(s.length - 1, Math.floor(p * s.length))];
};

/**
 * @param {import('@playwright/test').Browser} browser
 * @param {typeof SCENARIOS[number]} scenario
 * @param {boolean} [profile]
 */
async function sample(browser, scenario, profile = false) {
    const { context, page, cdp } = await readyPage(browser);
    try {
        if (scenario.prepare) await scenario.prepare(page);
        if (profile) {
            await cdp.send('Profiler.enable');
            await cdp.send('Profiler.setSamplingInterval', { interval: 100 });
            await cdp.send('Profiler.start');
        }
        const before = await metrics(cdp);
        const result = await scenario.measure(page);
        const after = await metrics(cdp);
        // `--shots dir`: the screen after the scenario, to check that it did what it says.
        if (args.shots && !profile) await page.screenshot({ path: path.join(args.shots, `${scenario.name.replace(/[^\p{L}\d]+/gu, '_')}.png`) });
        const cpuOf = (/** @type {string} */ k) => ((after[k] ?? 0) - (before[k] ?? 0)) * 1000;
        const out = { ...result, scriptMs: cpuOf('ScriptDuration'), layoutMs: cpuOf('LayoutDuration'), styleMs: cpuOf('RecalcStyleDuration'), taskMs: cpuOf('TaskDuration') };
        if (profile) out.profile = (await cdp.send('Profiler.stop')).profile;
        return out;
    } finally {
        await context.close();
    }
}

/** One line of the table from the samples of a scenario. */
function summarise(/** @type {typeof SCENARIOS[number]} */ scenario, /** @type {any[]} */ samples) {
    const m = (/** @type {string} */ k) => median(samples.map((s) => s[k]));
    if (scenario.kind === 'steps') {
        const perRun = samples.map((s) => {
            const q = Math.max(1, Math.floor(s.times.length / 4));
            return { med: median(s.times), p95: pct(s.times, 0.95), flat: median(s.times.slice(-q)) / median(s.times.slice(0, q)) };
        });
        return {
            name: scenario.name,
            ms: median(perRun.map((r) => r.med)),
            p95: median(perRun.map((r) => r.p95)),
            flat: median(perRun.map((r) => r.flat)),
            longTasks: m('longTasks'),
            longTaskMs: m('longTaskMs'),
            nodes: `${m('nodesFirst')}→${m('nodesLast')}`,
            // Per step: the main-thread time Chromium counted over the whole sequence, divided.
            cpuMs: median(samples.map((s) => s.taskMs / s.n)),
            scriptMs: median(samples.map((s) => s.scriptMs / s.n)),
            layoutMs: median(samples.map((s) => s.layoutMs / s.n)),
            styleMs: median(samples.map((s) => s.styleMs / s.n)),
            totalMs: m('totalMs'),
            status: samples[0].status
        };
    }
    return {
        name: scenario.name,
        ms: m('ms'),
        longTasks: m('longTasks'),
        longTaskMs: m('longTaskMs'),
        nodes: m('nodes'),
        cpuMs: m('taskMs'),
        scriptMs: m('scriptMs'),
        layoutMs: m('layoutMs'),
        styleMs: m('styleMs'),
        capped: samples.some((s) => s.capped)
    };
}

// ── Profiles ───────────────────────────────────────────────────────────────────

/** @type {Map<string, SourceMapConsumer | null>} */
const consumers = new Map();
function consumerFor(/** @type {string} */ url) {
    if (consumers.has(url)) return consumers.get(url);
    let consumer = null;
    try {
        const file = path.join(dist, new URL(url).pathname);
        if (existsSync(`${file}.map`)) consumer = new SourceMapConsumer(JSON.parse(readFileSync(`${file}.map`, 'utf8')));
    } catch {
        /* not one of ours */
    }
    consumers.set(url, consumer);
    return consumer;
}

/** @param {{url: string, lineNumber: number, columnNumber: number, functionName: string}} frame */
function where(frame) {
    if (!frame.url) return { file: '', line: 0, name: frame.functionName || '(anonymous)' };
    const c = consumerFor(frame.url);
    const short = (/** @type {string} */ f) => f.replace(/^.*?(node_modules\/|src\/)/, '$1').replace(/^\.\.\//g, '');
    if (c) {
        const o = c.originalPositionFor({ line: frame.lineNumber + 1, column: frame.columnNumber });
        if (o.source) return { file: short(o.source), line: o.line ?? 0, name: frame.functionName || o.name || '(anonymous)' };
    }
    return { file: short(new URL(frame.url).pathname), line: frame.lineNumber + 1, name: frame.functionName || '(anonymous)' };
}

/** Self time by source function, and inclusive time by the innermost src/ file of the stack. */
function aggregate(/** @type {any} */ profile) {
    const byId = new Map(profile.nodes.map((/** @type {any} */ n) => [n.id, n]));
    const parent = new Map();
    for (const n of profile.nodes) for (const c of n.children ?? []) parent.set(c, n.id);
    const self = new Map();
    const inSrc = new Map();
    let busy = 0;
    profile.samples.forEach((/** @type {number} */ id, /** @type {number} */ i) => {
        const dt = (profile.timeDeltas[i] ?? 0) / 1000;
        const node = /** @type {any} */ (byId.get(id));
        const fn = node.callFrame.functionName;
        if (fn === '(idle)') return;
        busy += dt;
        const w = where(node.callFrame);
        const key = fn.startsWith('(') ? fn : `${w.name} ${w.file}:${w.line}`;
        self.set(key, (self.get(key) ?? 0) + dt);
        // The innermost frame in src/ (our code) carries the sample; outside src/ it goes to its own file.
        let cur = id;
        /** @type {string | null} */
        let owner = null;
        while (cur !== undefined) {
            const f = /** @type {any} */ (byId.get(cur)).callFrame;
            const wf = where(f);
            if (wf.file.startsWith('src/')) {
                owner = `${wf.file} (${wf.name}:${wf.line})`;
                break;
            }
            cur = parent.get(cur);
        }
        owner ??= fn.startsWith('(') ? fn : `hors src/ : ${w.file}`;
        inSrc.set(owner, (inSrc.get(owner) ?? 0) + dt);
    });
    const top = (/** @type {Map<string, number>} */ m, k = 12) =>
        [...m.entries()]
            .sort((a, b) => b[1] - a[1])
            .slice(0, k)
            .map(([name, ms]) => ({ name, ms: Math.round(ms * 10) / 10, share: Math.round((1000 * ms) / busy) / 10 }));
    // The native buckets ((program): style, layout, paint and the event loop; GC) hide the
    // script's own ranking: the shares are also given over the JavaScript samples alone.
    const nativeMs = [...self.entries()].filter(([k]) => k.startsWith('(')).reduce((t, [, ms]) => t + ms, 0);
    const js = busy - nativeMs;
    const topJs = (/** @type {Map<string, number>} */ m, k = 10) =>
        [...m.entries()]
            .filter(([name]) => !name.startsWith('('))
            .sort((a, b) => b[1] - a[1])
            .slice(0, k)
            .map(([name, ms]) => ({ name, ms: Math.round(ms * 10) / 10, share: Math.round((1000 * ms) / js) / 10 }));
    return { busyMs: Math.round(busy), jsMs: Math.round(js), self: top(self, 4), selfJs: topJs(self), bySrcJs: topJs(inSrc) };
}

// ── Main ───────────────────────────────────────────────────────────────────────

const chosen = SCENARIOS.filter((s) => !only || only.test(s.name));
const server = await serve();
const browser = await chromium.launch();
const results = [];
/** @type {Record<string, any>} */
const profiles = {};
try {
    for (const scenario of chosen) {
        const samples = [];
        for (let i = 0; i < runs; i++) {
            try {
                samples.push(await sample(browser, scenario));
            } catch (error) {
                console.error(`${scenario.name}: ${/** @type {Error} */ (error).message.split('\n')[0]}`);
            }
        }
        if (samples.length) results.push(summarise(scenario, samples));
    }
    // The costliest by main-thread time: a wall time can be a deliberate wait (the filter's debounce).
    const ranked = [...results].filter((r) => !r.name.startsWith('témoin') && r.p95 === undefined).sort((a, b) => b.cpuMs - a.cpuMs);
    for (const r of ranked.slice(0, profileTop)) {
        const scenario = chosen.find((s) => s.name === r.name);
        try {
            const s = await sample(browser, /** @type {any} */ (scenario), true);
            profiles[r.name] = aggregate(s.profile);
        } catch (error) {
            console.error(`profile ${r.name}: ${/** @type {Error} */ (error).message.split('\n')[0]}`);
        }
    }
} finally {
    await browser.close();
    server.kill();
}

const f = (/** @type {number} */ x) => (Number.isFinite(x) ? x.toFixed(x < 10 ? 1 : 0) : '–');
console.log(`| scénario (médiane de ${runs}, CPU ×${cpu}) | ms | p95 pas | fin/début | CPU fil principal | script | layout | style | long tasks | ms en long tasks | nœuds DOM |`);
console.log('|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|');
for (const r of [...results].sort((a, b) => b.ms - a.ms)) {
    const step = r.p95 !== undefined;
    console.log(
        `| ${r.name}${r.capped ? ' (plafonné)' : ''} | ${f(r.ms)} | ${step ? f(r.p95) : ''} | ${step ? r.flat.toFixed(2) : ''} | ${f(r.cpuMs)} | ${f(r.scriptMs)} | ${f(r.layoutMs)} | ${f(r.styleMs)} | ${r.longTasks} | ${f(r.longTaskMs)} | ${r.nodes} |${step && r.status ? ` <!-- ${r.status.replace(/\s+/g, ' ').slice(0, 60)} -->` : ''}`
    );
}
console.log('\nPas (j/k, défilement) : ms, p95 et fin/début par pas ; CPU, script, layout et style par pas.');
for (const [name, p] of Object.entries(profiles)) {
    console.log(`\n### profil : ${name} (${p.busyMs} ms échantillonnés hors idle, dont ${p.jsMs} ms de JavaScript)`);
    console.log('self, tout compris :');
    for (const e of p.self) console.log(`  ${e.share}%  ${e.ms} ms  ${e.name}`);
    console.log('self, part du JavaScript :');
    for (const e of p.selfJs) console.log(`  ${e.share}%  ${e.ms} ms  ${e.name}`);
    console.log('par fichier de src/ le plus interne de la pile, part du JavaScript :');
    for (const e of p.bySrcJs) console.log(`  ${e.share}%  ${e.ms} ms  ${e.name}`);
}
if (args.json) writeFileSync(args.json, JSON.stringify({ volume: VOLUME, runs, cpu, results, profiles }, null, 2));
