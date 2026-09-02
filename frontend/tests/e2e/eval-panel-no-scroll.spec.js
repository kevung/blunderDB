/**
 * eval-panel-no-scroll.spec.js
 *
 * "Only the candidate list ever scrolls" was decided by ADR-0017, restated by
 * ADR-0018, and left as an owed test by both. ADR-0020 finally owes it for a
 * reason rather than out of conscience: moving the badges to their own strip
 * (rule 8) spends ~18 px of the panel's height budget, and a property nobody
 * measures after three layout changes is not a property.
 *
 * Measured at the DEFAULT panel size, which is the only size at which the claim
 * is true — asserting it at any size would assert the impossible.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { epcResultA } from './helpers/fixtures.js';

// A cube decision as the live evaluation delivers it (no dice on the board, so
// gammonNet answers with a cube rather than a move list).
const cubeEval = {
    cube: {
        analysisDepth: '2-ply',
        analysisEngine: 'gammonNet v1.2.1',
        playerWinChances: 0.5432,
        playerGammonChances: 0.1321,
        playerBackgammonChances: 0.0071,
        opponentWinChances: 0.4568,
        opponentGammonChances: 0.1102,
        opponentBackgammonChances: 0.0049,
        cubefulNoDoubleEquity: 0.055,
        cubefulNoDoubleError: 0,
        cubefulDoubleTakeEquity: -0.256,
        cubefulDoubleTakeError: -0.311,
        cubefulDoublePassEquity: 1.0,
        cubefulDoublePassError: 0,
        bestCubeAction: 'No Double'
    },
    cubeVerdict: 'no_double',
    preRoll: {
        playerWinChance: 0.5432,
        playerGammonChance: 0.1321,
        playerBackgammonChance: 0.0071,
        opponentWinChance: 0.4568,
        opponentGammonChance: 0.1102,
        opponentBackgammonChance: 0.0049,
        cubelessEquity: 0.04
    }
};

// The same position seen as a pure bearoff: the facts table grows its five race
// columns, which is the widest the panel ever gets without a move list.
const raceEpc = {
    ...epcResultA,
    top: {
        all_in_home: true,
        checker_count: 15,
        epc: { epc: 71.2, pipCount: 66, wastage: 5.2, meanRolls: 11.867, stdDev: 2.4 }
    },
    race: {
        regime: 'exact',
        on_roll: 0,
        source_checkers: 6,
        win_prob: 0.6421,
        money: { cube_state: 'centered', cubeless: 0.284, no_double: 0.301, double_take: 0.42, double_pass: 1.0, verdict: 'double_take' }
    }
};

async function openEval(page, { epc, evalResult }) {
    await installWailsMock(page);
    await page.addInitScript(
        ({ epcFixture, evalFixture }) => {
            window.__epcFixture = epcFixture;
            window.__evalFixture = evalFixture;
        },
        { epcFixture: epc, evalFixture: evalResult }
    );
    await page.addInitScript(() => {
        Object.defineProperty(window.go.database.Database, 'ComputeEPCFromPosition', {
            get: () => () => Promise.resolve(window.__epcFixture ?? null),
            configurable: true
        });
        Object.defineProperty(window.go.gui.App, 'EvaluatePositionImmediate', {
            get: () => () => Promise.resolve(window.__evalFixture ?? null),
            configurable: true
        });
    });

    await page.goto('/');
    await expect(page.locator('[data-testid="status-bar"]')).toBeVisible({ timeout: 8000 });
    await page.click('[data-testid="tab-epc"]');
    await expect(page.locator('[data-testid="tab-epc"]')).toHaveClass(/active/);
    await expect(page.locator('.epc-panel .cube-table')).toBeVisible({ timeout: 4000 });
}

async function overflow(page) {
    return page.locator('.epc-panel').evaluate((el) => el.scrollHeight - el.clientHeight);
}

test('a non-race cube position fits the panel at the default size', async ({ page }) => {
    await openEval(page, { epc: null, evalResult: cubeEval });
    expect(await overflow(page)).toBeLessThanOrEqual(0);
});

test('a race cube position fits too — five race columns and all', async ({ page }) => {
    await openEval(page, { epc: raceEpc, evalResult: cubeEval });
    await expect(page.locator('.epc-panel')).toContainText('66.47');
    expect(await overflow(page)).toBeLessThanOrEqual(0);
});

// blunderDB's own default window (config.go: 1024 px), not the suite's 1280:
// the claim ADR-0021 makes is about the width the app actually ships with, and
// at 1280 the old layout fit in French too — the test would have passed against
// the very layout it is written to rule out.
test.describe("at blunderDB's default window width", () => {
    test.use({ viewport: { width: 1024, height: 768 } });

    test('the decision sits beside the facts, not under them (ADR-0021)', async ({ page }) => {
        await openEval(page, { epc: raceEpc, evalResult: cubeEval });

        // The facts are two stacked blocks — two tbodies of one table, sharing
        // one column grid — precisely so this holds at the default window in
        // every language: on a single line of ten columns they needed 963 px
        // (fr) to 1125 px (el) against the 996 px the panel has, so the cube
        // block, last in the flex row, wrapped under them in seven languages
        // out of nine.
        await expect(page.locator('.epc-panel .facts-table tbody')).toHaveCount(2);

        const box = (sel) => page.locator(sel).boundingBox();
        const facts = await box('.epc-panel .facts-table');
        const cube = await box('.epc-panel .decision-cube');
        expect(cube.x).toBeGreaterThan(facts.x + facts.width - 1); // to the right of
        expect(cube.y).toBeLessThan(facts.y + 4); // and on the same line, not below

        // Rule 4: the slack is left over at the end of the row, never inserted
        // between the blocks.
        expect(cube.x - (facts.x + facts.width)).toBeLessThan(40);
    });

    test('and the panel still does not scroll there', async ({ page }) => {
        await openEval(page, { epc: raceEpc, evalResult: cubeEval });
        expect(await overflow(page)).toBeLessThanOrEqual(0);
    });
});

test('the decision has the same three options plus a verdict, race or not', async ({ page }) => {
    await openEval(page, { epc: raceEpc, evalResult: cubeEval });
    await expect(page.locator('.epc-panel .cube-table tbody tr')).toHaveCount(4);

    // The strip sits above the content, never inside it: that is what stops a
    // `margin-left: auto` from manufacturing a void across the middle.
    const stripThenRow = await page.locator('.epc-content').evaluate((el) => {
        const kids = [...el.children].map((c) => c.className);
        return kids.findIndex((c) => c.includes('badges-strip')) < kids.findIndex((c) => c.includes('top-row'));
    });
    expect(stripThenRow).toBe(true);
});
