import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, it } from 'vitest';
import { createAutomata, framesAt, THRESHOLDS } from '../vision/automata.js';
import { createDetector, replay } from '../vision/descriptors.js';
import { simulate } from './fixtures/vision/board.js';
import { SCENARIOS } from './fixtures/vision/scenarios.js';

// The tuned values, written out: the boundary journals are built on these literals, not on
// THRESHOLDS, so a threshold that moves breaks a behaviour test and not only the pin below.
const T = Object.freeze({
    stateRate: 2,
    denseRate: 10,
    denseMaxDt: 0.25,
    maxGap: 2,
    windowAfter: 2,
    noiseWindow: 20,
    noiseMin: 3,
    hiFactor: 3,
    loFactor: 1.5,
    hiFloor: 0.003,
    loFloor: 0.002,
    handsIn: 0.03,
    handsOut: 0.02,
    throwFrames: 1,
    restFrames: 5,
    handsInFrames: 2,
    handsOutFrames: 3,
    throwGap: 1,
    maxRoll: 4,
    razHi: 0.012,
    razLo: 0.004
});

// The automata are tuned on journals (ADR-0109 rule 8). Three kinds of guard hold the tuning:
// the scripted scenes, played through the live pipeline, must still give the journals on disk;
// each journal must replay to its events, scored by F1 at ±0.5 s against the annotated truth;
// and every threshold has a boundary journal, written in literal values, whose behaviour
// changes when that threshold moves by more than a part in a million.
//
// These journals are synthetic: the same generator sets the thresholds and checks them, for
// want of a real video in the repository. The F1 here says the automata do what was meant on
// scenes built for them, not that they hold on a camera; the thresholds are to be set again on
// real journals (V1.7).

const DIR = join(__dirname, 'fixtures', 'vision');
const NAMES = Object.keys(SCENARIOS);
/** @param {string} name */
const load = (name) => JSON.parse(readFileSync(join(DIR, `${name}.journal.json`), 'utf8'));
/** @param {import('../vision/contract.js').VisionEvent} e */
const plain = (e) => ({ type: e.type, mediaTime: e.mediaTime, state: e.state, payload: { ...e.payload } });

/**
 * One-to-one matching of detected events to truth, same type, within `tol` seconds; greedy on
 * the closest pairs first.
 * @param {{ type: string, mediaTime: number }[]} detected
 * @param {{ type: string, mediaTime: number }[]} truth
 */
function score(detected, truth, tol = 0.5) {
    /** @type {[number, number, number, number][]} */
    const pairs = [];
    detected.forEach((d, i) =>
        truth.forEach((g, j) => d.type === g.type && Math.abs(d.mediaTime - g.mediaTime) <= tol + 1e-9 && pairs.push([Math.abs(d.mediaTime - g.mediaTime), i, j, d.mediaTime - g.mediaTime]))
    );
    pairs.sort((a, b) => a[0] - b[0]);
    const usedD = new Set();
    const usedG = new Set();
    const offsets = [];
    for (const [, i, j, off] of pairs) {
        if (usedD.has(i) || usedG.has(j)) continue;
        usedD.add(i);
        usedG.add(j);
        offsets.push(off);
    }
    const tp = offsets.length;
    const f1 = detected.length + truth.length === 0 ? 1 : (2 * tp) / (detected.length + truth.length);
    offsets.sort((a, b) => a - b);
    const median = offsets.length ? offsets[offsets.length >> 1] : 0;
    return { f1, median: Math.round(median * 1e3) / 1e3 };
}

// Playing a scene renders and describes each analysed frame: seconds under a loaded suite.
const SLOW = { timeout: 60_000 };

describe('automata journals', SLOW, () => {
    it.each(NAMES)('%s: the scene still plays to the journal on disk', (name) => {
        const sc = SCENARIOS[name];
        const { records, calibration } = simulate(sc.script, sc.duration);
        const fixture = load(name);
        expect(calibration).toEqual(fixture.calibration);
        expect(records).toEqual(fixture.records);
    });

    it.each(NAMES)('%s: replays to its expected events', (name) => {
        const fixture = load(name);
        expect(replay(fixture.records).map(plain)).toEqual(fixture.expected);
    });

    // F1 and median offset (detected − truth) at ±0.5 s. The offsets are late, never early: at
    // 1× a hand that enters between two coarse samples is seen at the next one.
    it.each([
        ['two-turns', 1, 0.2],
        ['opening', 1, 0.4],
        ['new-game', 1, 0.1],
        ['cut-then-reset', 1, 0],
        ['still', 1, 0]
    ])('%s: F1 %s, median offset %s s', (name, f1, median) => {
        const fixture = load(name);
        expect(score(replay(fixture.records), fixture.truth)).toEqual({ f1, median });
    });

    it('stays coarse on a still table: two images per second of media', () => {
        const { records } = load('still');
        expect(records.length).toBe(40);
    });

    // A decoder that cannot keep up: the durations are halved (floor 2). The dice keep their
    // F1; a hand that pauses between two samples 0.2 s apart can go unseen, and with it a pickup.
    it.each([
        ['two-turns', 0.83, 1],
        ['opening', 1, 1],
        ['new-game', 1, 1]
    ])('%s at 5 img/s in the windows: F1 %s, dice F1 %s', (name, f1, diceF1) => {
        const sc = SCENARIOS[name];
        const { events } = simulate(sc.script, sc.duration, { deliver: 5 });
        const dice = (/** @type {{ type: string }} */ e) => e.type === 'dice.thrown' || e.type === 'dice.rest';
        expect(Math.round(score(events, sc.truth).f1 * 100) / 100).toBe(f1);
        expect(score(events.filter(dice), sc.truth.filter(dice)).f1).toBe(diceF1);
    });

    // Sensor noise from none to ±2 grey levels, on top of the ±3 of the journals.
    it.each([0, 1, 2])('two-turns with sensor noise ±%s: F1 1', (noise) => {
        const sc = SCENARIOS['two-turns'];
        expect(score(simulate(sc.script, sc.duration, { noise }).events, sc.truth).f1).toBe(1);
    });

    it('a cut drops the start template: no game event after it', () => {
        const types = load('cut-then-reset').expected.map((/** @type {any} */ e) => e.type);
        expect(types).not.toContain('game.start');
        expect(types).not.toContain('game.end');
    });
});

// Boundary journals, built by hand. Both halves sit on a constant floor F, so the noise spread
// is nil and the levels are the floors of T; a frame is "x above the floor".
const F = 0.25;
/** Relative step across a boundary: any move of a threshold beyond it breaks a test. */
const D = 1e-6;

function journal(warm = T.noiseWindow) {
    /** @type {import('../vision/descriptors.js').DescriptorRecord[]} */
    const records = [{ t: 0, u: 0, w: null, r: { left: [0.5, null, null], right: [0.5, null, null] } }];
    let t = 0;
    const api = {
        records,
        /**
         * @param {{ x?: number, xRight?: number, bg?: number, dt?: number, n?: number, u?: 0 | 1 }} [o]
         *   `x` excess on the left half, `xRight` on the right (default: the floor), `bg` excess of
         *   both halves over the floor in distance to the background
         */
        frame({ x = 0, xRight = 0, bg = 0, dt = 0.1, n = 1, u = 0 } = {}) {
            for (let i = 0; i < n; i++) {
                t = Math.round((t + dt) * 1e9) / 1e9;
                records.push({ t, u, w: u ? 0.9 : 0, r: { left: [0.5, F + x, F + bg], right: [0.5, F + xRight, F + bg] } });
            }
            return api;
        },
        get t() {
            return t;
        }
    };
    return warm > 0 ? api.frame({ n: warm }) : api;
}

/**
 * @param {ReturnType<typeof journal>} j
 * @param {Partial<typeof THRESHOLDS>} [th]
 */
const run = (j, th) => {
    const a = createAutomata(th);
    return j.records.flatMap((r) => a.push(r)).map((e) => [e.type, e.state]);
};
const types = (/** @type {ReturnType<typeof journal>} */ j) => run(j).map(([type]) => type);

/** A dice throw on the left half of excess `x`, then calm frames at excess `calm`. */
const throwThen = (/** @type {number} */ x, calm = 0, n = T.restFrames) => journal().frame({ x }).frame({ x: calm, n });
/** A hand in for a second at excess `x`, then frames at excess `out`. */
const handThen = (/** @type {number} */ x, out = 0, n = T.handsOutFrames) => journal().frame({ x, n: 10 }).frame({ x: out, n });

describe('every threshold holds on a boundary journal', () => {
    it('pins the tuned values', () => {
        expect(THRESHOLDS).toEqual(T);
    });

    it('hiFloor: dice motion starts at E_hi', () => {
        expect(types(throwThen(T.hiFloor * (1 + D)))).toEqual(['dice.thrown', 'dice.rest']);
        expect(types(throwThen(T.hiFloor * (1 - D)))).toEqual([]);
    });

    it('loFloor: dice rest under E_lo', () => {
        const x = 2 * T.hiFloor;
        expect(types(throwThen(x, T.loFloor * (1 - D)))).toEqual(['dice.thrown', 'dice.rest']);
        expect(types(throwThen(x, T.loFloor * (1 + D)))).toEqual([]);
    });

    describe('hiFactor and loFactor scale the noise spread', () => {
        // On the scene journals the sensor noise is stationary, and a census distance averaged
        // over thousands of bits barely varies: 3σ stays under hiFloor and the floors decide.
        // Here the noise spreads: half the floor samples at F − a, half at F + a (median F,
        // MAD a, σ = 1.4826 a), and the factors decide.
        const a = 0.002;
        const sigma = 1.4826 * a;
        function noisy() {
            const j = journal();
            for (let i = 0; i < T.noiseWindow; i++) j.frame({ x: i % 2 ? a : -a, xRight: i % 2 ? a : -a });
            return j;
        }
        it('the levels follow the spread above the floors', () => {
            const au = createAutomata();
            for (const r of noisy().frame().records) au.push(r);
            expect(au.levels.hi).toBeCloseTo(T.hiFactor * sigma, 12);
            expect(au.levels.lo).toBeCloseTo(T.loFactor * sigma, 12);
            expect(T.hiFactor * sigma).toBeGreaterThan(T.hiFloor);
            expect(T.loFactor * sigma).toBeGreaterThan(T.loFloor);
        });
        it('hiFactor', () => {
            expect(
                types(
                    noisy()
                        .frame({ x: T.hiFactor * sigma * (1 + D) })
                        .frame({ n: T.restFrames })
                )
            ).toEqual(['dice.thrown', 'dice.rest']);
            expect(
                types(
                    noisy()
                        .frame({ x: T.hiFactor * sigma * (1 - D) })
                        .frame({ n: T.restFrames })
                )
            ).toEqual([]);
        });
        it('loFactor', () => {
            const x = 2 * T.hiFactor * sigma;
            expect(
                types(
                    noisy()
                        .frame({ x })
                        .frame({ x: T.loFactor * sigma * (1 - D), n: T.restFrames })
                )
            ).toEqual(['dice.thrown', 'dice.rest']);
            expect(
                types(
                    noisy()
                        .frame({ x })
                        .frame({ x: T.loFactor * sigma * (1 + D), n: T.restFrames })
                )
            ).toEqual([]);
        });
    });

    it('handsIn: a hand enters at its level, under it is dice', () => {
        expect(types(journal().frame({ x: T.handsIn * (1 + D), n: T.handsInFrames }))).toEqual(['hands.in']);
        expect(types(journal().frame({ x: T.handsIn * (1 - D), n: T.handsInFrames }))).toEqual([]);
    });

    it('handsOut: a hand leaves under its level', () => {
        expect(types(handThen(T.handsIn * 2, T.handsOut * (1 - D)))).toEqual(['hands.in', 'hands.out']);
        expect(types(handThen(T.handsIn * 2, T.handsOut * (1 + D)))).toEqual(['hands.in']);
    });

    it('handsInFrames and handsOutFrames', () => {
        const x = 2 * T.handsIn;
        expect(types(journal().frame({ x, n: T.handsInFrames }))).toEqual(['hands.in']);
        expect(types(journal().frame({ x, n: T.handsInFrames - 1 }))).toEqual([]);
        expect(types(handThen(x, 0, T.handsOutFrames))).toEqual(['hands.in', 'hands.out']);
        expect(types(handThen(x, 0, T.handsOutFrames - 1))).toEqual(['hands.in']);
    });

    it('restFrames: a rest is confirmed after its frames, not before', () => {
        const x = 2 * T.hiFloor;
        expect(types(throwThen(x, 0, T.restFrames))).toEqual(['dice.thrown', 'dice.rest']);
        expect(types(throwThen(x, 0, T.restFrames - 1))).toEqual([]);
    });

    it('throwFrames: a throw needs its moving frames', () => {
        const x = 2 * T.hiFloor;
        const moving = (/** @type {number} */ n) => journal().frame({ x, n }).frame({ n: T.restFrames });
        expect(types(moving(T.throwFrames))).toEqual(['dice.thrown', 'dice.rest']);
        expect(types(moving(T.throwFrames + 1))).toEqual(['dice.thrown', 'dice.rest']);
        expect(run(moving(T.throwFrames), { throwFrames: T.throwFrames + 1 })).toEqual([]);
    });

    it('maxRoll: dice that never settle were not thrown', () => {
        const x = 2 * T.hiFloor;
        const rolling = (/** @type {number} */ seconds) =>
            journal()
                .frame({ x, n: Math.round(seconds * 10) + 1 })
                .frame({ n: T.restFrames });
        expect(types(rolling(T.maxRoll - 0.1))).toEqual(['dice.thrown', 'dice.rest']);
        expect(types(rolling(T.maxRoll + 0.1))).toEqual([]);
    });

    it('denseMaxDt: durations count on dense frames only', () => {
        const x = 2 * T.handsIn;
        expect(types(journal().frame({ x, dt: T.denseMaxDt, n: T.handsInFrames }))).toEqual(['hands.in']);
        expect(types(journal().frame({ x, dt: T.denseMaxDt * (1 + D), n: T.handsInFrames + 2 }))).toEqual([]);
    });

    it('framesAt: durations in frames at 10 img/s, halved at 5 img/s with a floor of 2', () => {
        expect([1, 2, 3, 5].map((n) => framesAt(n, 0.1))).toEqual([1, 2, 3, 5]);
        expect([1, 2, 3, 5].map((n) => framesAt(n, 0.2))).toEqual([1, 2, 2, 3]);
    });

    it('maxGap: a longer gap restarts the automata', () => {
        const x = 2 * T.hiFloor;
        expect(types(journal().frame({ x }).frame({ dt: T.maxGap }).frame({ n: T.restFrames }))).toEqual(['dice.thrown', 'dice.rest']);
        expect(
            types(
                journal()
                    .frame({ x })
                    .frame({ dt: T.maxGap * (1 + D) })
                    .frame({ n: T.restFrames })
            )
        ).toEqual([]);
    });

    it('noiseMin: no motion is read before the floor is known', () => {
        const x = 2 * T.handsIn;
        /** @param {number} n */
        const fresh = (n) => journal(n).frame({ x, n: T.handsInFrames });
        expect(types(fresh(T.noiseMin))).toEqual(['hands.in']);
        expect(types(fresh(T.noiseMin - 1))).toEqual([]);
    });

    it('noiseWindow: a new floor is taken after half the window', () => {
        const step = T.hiFloor / 2;
        const floorAfter = (/** @type {number} */ n) => {
            const a = createAutomata();
            for (const r of journal().frame({ x: step, xRight: step, n }).records) a.push(r);
            return a.levels.floor.left;
        };
        const half = T.noiseWindow / 2;
        expect(floorAfter(half + 1)).toBeCloseTo(F + step, 12);
        expect(floorAfter(half)).toBeCloseTo(F + step / 2, 12);
    });

    it('throwGap: dice moving within it of a hand leaving were thrown by that hand', () => {
        const x = 2 * T.hiFloor;
        const after = (/** @type {number} */ gap) => {
            const j = handThen(2 * T.handsIn);
            const out = j.t - (T.handsOutFrames - 1) * 0.1;
            j.frame({ dt: Math.round((out + gap - j.t) * 1e6) / 1e6 })
                .frame({ x })
                .frame({ n: T.restFrames });
            const a = createAutomata();
            const thrown = j.records.flatMap((r) => a.push(r)).find((e) => e.type === 'dice.thrown');
            return thrown && Math.round((thrown.mediaTime - out) * 1e6) / 1e6;
        };
        expect(after(T.throwGap - 0.1)).toBe(0);
        expect(after(T.throwGap + 0.1)).toBeGreaterThan(0);
    });

    describe('razHi and razLo: the board leaves the start template, then comes back', () => {
        /** @param {number} far @param {number} back */
        const game = (far, back) => journal().frame({ bg: far, n: 3 }).frame({ bg: back, n: T.restFrames });
        it('razHi arms a game', () => {
            expect(types(game(T.razHi * (1 + D), 0))).toEqual(['game.start']);
            expect(types(game(T.razHi * (1 - D), 0))).toEqual([]);
        });
        it('razLo says the board is back', () => {
            expect(types(game(2 * T.razHi, T.razLo * (1 - D)))).toEqual(['game.start']);
            expect(types(game(2 * T.razHi, T.razLo * (1 + D)))).toEqual([]);
        });
    });
});

describe('cadence', () => {
    it('compares states at stateRate and stays dense windowAfter s after a change', () => {
        const a = createAutomata();
        const j = journal();
        j.records.forEach((r) => a.push(r));
        expect(a.rate).toBe(T.stateRate);
        const t0 = j.t + 0.5;
        // A hand between two samples: it opens the window and leaves nothing running.
        a.push({ t: t0, u: 0, w: 0, r: { left: [0.5, F + 2 * T.handsIn, F], right: [0.5, F, F] } });
        expect(a.rate).toBe(T.denseRate);
        a.push({ t: t0 + T.windowAfter - 0.1, u: 0, w: 0, r: { left: [0.5, F, F], right: [0.5, F, F] } });
        expect(a.rate).toBe(T.denseRate);
        a.push({ t: t0 + T.windowAfter, u: 0, w: 0, r: { left: [0.5, F, F], right: [0.5, F, F] } });
        expect(a.rate).toBe(T.stateRate);
    });

    it('the detector reports the rate the automata want', () => {
        const d = createDetector();
        for (const r of handThen(2 * T.handsIn).records) d.push(r);
        expect(d.rate).toBe(T.denseRate);
    });

    it('marks an episode first seen by a coarse record as to check', () => {
        // A hand seen at 2 img/s, then densely.
        const j = journal()
            .frame({ x: 2 * T.handsIn, dt: 0.5 })
            .frame({ x: 2 * T.handsIn, n: T.handsInFrames });
        expect(run(j)).toEqual([['hands.in', 'check']]);
        expect(run(journal().frame({ x: 2 * T.handsIn, n: T.handsInFrames }))).toEqual([['hands.in', 'sure']]);
    });
});

describe('dice payload', () => {
    it('names the half, and flags the opening roll when both halves moved', () => {
        const x = 2 * T.hiFloor;
        const a = createAutomata();
        const ev = journal()
            .frame({ x: 0, xRight: x })
            .frame({ n: T.restFrames })
            .records.flatMap((r) => a.push(r));
        expect(ev.map((e) => e.payload)).toEqual([{ side: 'right' }, { side: 'right' }]);
        const b = createAutomata();
        const op = journal()
            .frame({ x, xRight: x })
            .frame({ n: T.restFrames })
            .records.flatMap((r) => b.push(r));
        expect(op.map((e) => e.payload)).toEqual([{}, { opening: true }]);
    });

    it('needs a hand between two throws: the dice had to be picked up', () => {
        const x = 2 * T.hiFloor;
        const twice = journal().frame({ x }).frame({ n: T.restFrames }).frame({ x }).frame({ n: T.restFrames });
        expect(types(twice)).toEqual(['dice.thrown', 'dice.rest']);
        const picked = journal()
            .frame({ x })
            .frame({ n: T.restFrames })
            .frame({ x: 2 * T.handsIn, n: 5 })
            .frame({ n: 30 })
            .frame({ x })
            .frame({ n: T.restFrames });
        expect(types(picked)).toEqual(['dice.thrown', 'dice.rest', 'hands.in', 'hands.out', 'dice.pickup', 'dice.thrown', 'dice.rest']);
    });
});

describe('the worker follows the cadence', SLOW, () => {
    it('posts the dense rate when a window opens and the state rate when it closes', async () => {
        const { createWorkerCore } = await import('../vision/workerCore.js');
        const { createMemoryJournal } = await import('../vision/journal.js');
        const { BOARD_CALIBRATION, render } = await import('./fixtures/vision/board.js');
        /** @type {any[]} */
        const posted = [];
        const core = createWorkerCore({ post: (m) => posted.push(m), openJournal: async (o) => createMemoryJournal(o.session) });
        await core.handle({ kind: 'calibrate', calibration: BOARD_CALIBRATION });
        await core.handle({ kind: 'start', session: 's' });
        const script = SCENARIOS['two-turns'].script;
        // Twelve seconds of the scene, then the table left still for three.
        for (let i = 0; i <= 30; i++) {
            const t = i / 2;
            const data = render(script(Math.min(t, 12)), i * 15);
            await core.handle({ kind: 'frame', mediaTime: t, width: 320, height: 180, buffer: data.buffer });
        }
        const rates = posted.filter((m) => m.kind === 'rate').map((m) => m.fps);
        // Coarse until the hand of 3.0 s is seen at 3.5 s; fed at 2 img/s whatever it asks, the
        // worker still closes the window once the table is still.
        expect(rates.slice(0, 2)).toEqual([T.stateRate, T.denseRate]);
        expect(rates.at(-1)).toBe(T.stateRate);
    });
});
