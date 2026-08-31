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
        analysisEngine: 'gammonNet v1.2.0',
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
