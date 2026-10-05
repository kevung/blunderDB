/**
 * eval-without-database.spec.js — the Eval panel with no database open.
 *
 * The scratch board needs no library: it is pasted, evaluated, rolled out in
 * memory, copied as text and as an image, saved to an image file, and left for
 * the search board and back. Only adding it to a library asks for one. The
 * walk ends by checking that nothing went to the database backend but pure
 * computations.
 */

import { test, expect } from '@playwright/test';
import { dismissHomeScreen, installWailsMock, getWailsCalls } from './helpers/wailsMock.js';
import en from '../../src/i18n/locales/en.json' with { type: 'json' };

const MOVES = [
    { move: '8/5 6/5', equity: 0.1, equityError: 0 },
    { move: '24/21 13/11', equity: 0.05, equityError: -0.05 },
    { move: '13/10 13/11', equity: 0.0, equityError: -0.1 }
];

const FAST = { truncation: 7, min_games: 108, max_games: 216, jsd_limit: 3, ply: 0, candidates: 5, seed: 1, workers: 0 };

/** The opening position, 31 to play: what ParsePositionText hands back for the pasted XGID. */
function openingPosition() {
    const points = Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 }));
    for (const [point, checkers] of [
        [24, 2],
        [13, 5],
        [8, 3],
        [6, 5]
    ]) {
        points[point] = { checkers, color: 0 };
        points[25 - point] = { checkers, color: 1 };
    }
    return {
        board: { points, bearoff: [0, 0] },
        cube: { owner: -1, value: 0 },
        dice: [3, 1],
        score: [-1, -1],
        player_on_roll: 0,
        decision_type: 0,
        has_jacoby: 0,
        max_cube: 0,
        has_beaver: 0
    };
}

// Database bindings that touch no database: a parser, the race tables, a constant.
const PURE_DATABASE_CALLS = ['ParsePositionText', 'ComputeEPCFromPosition', 'GetDatabaseVersion'];

const XGID = 'XGID=-b----E-C---eE---c-e----B-:0:0:1:31:0:0:0:0:10';

const row = (/** @type {import('@playwright/test').Page} */ page, /** @type {string} */ move) => page.locator(`.eval-panel tr[data-move="${move}"]`);
const status = (/** @type {import('@playwright/test').Page} */ page) => page.locator('[data-testid="status-bar-message"]');

test('the Eval panel works end to end without a database', async ({ page }) => {
    await installWailsMock(page, {
        app: {
            EvaluatePositionImmediate: { moves: MOVES, preRoll: null },
            StartRollout: 1,
            RolloutPresets: [
                { name: 'fast', settings: FAST },
                { name: 'standard', settings: FAST }
            ],
            CopyImageToClipboard: '',
            SaveBoardImageDialog: '/home/user/board.svg'
        },
        database: { ParsePositionText: { position: openingPosition() } },
        runtime: { ClipboardGetText: XGID, ClipboardSetText: true }
    });
    await page.goto('/');
    await expect(page.locator('[data-testid="status-bar"]')).toBeVisible({ timeout: 8000 });
    await dismissHomeScreen(page);
    await page.click('[data-testid="tab-eval"]');
    await expect(page.locator('[data-testid="tab-eval"]')).toHaveClass(/active/);

    // Paste an XGID onto the scratch board: the engine evaluates it.
    await page.keyboard.press('Control+v');
    await expect(status(page)).toHaveText(en.status.positionPastedClipboard);
    await expect(row(page, '8/5 6/5')).toBeVisible({ timeout: 4000 });

    // Pick two plays and roll them out from the menu: in memory, nothing stored.
    await row(page, '8/5 6/5').click();
    await row(page, '13/10 13/11').click({ modifiers: ['Control'] });
    await row(page, '13/10 13/11').click({ button: 'right' });
    await page
        .getByText(/Rollout \(/)
        .first()
        .click();
    await expect.poll(async () => (await getWailsCalls(page, 'StartRollout')).length).toBe(1);
    const [rollout] = await getWailsCalls(page, 'StartRollout');
    expect(rollout.args[0]).toMatchObject({ positionId: 0, store: false, moves: ['8/5 6/5', '13/10 13/11'] });

    // Ctrl-C: the XGID of the board, as text.
    await page.keyboard.press('Control+c');
    await expect(status(page)).toHaveText(en.status.positionCopied);
    const texts = await getWailsCalls(page, 'ClipboardSetText');
    expect(texts.at(-1)?.args[0]).toContain('XGID=');

    // C-X: the board image; C-X C-X: the board with the evaluation.
    await page.keyboard.press('Control+x');
    await expect(status(page)).toHaveText(en.status.boardImageCopied);
    await page.keyboard.press('Control+x');
    await page.keyboard.press('Control+x');
    await expect(status(page)).toHaveText(en.status.boardAnalysisCopied);

    // The panel's menu saves the board as an image file.
    await row(page, '8/5 6/5').click({ button: 'right' });
    await page.getByText(en.board.menu.saveImageSVG).click();
    await expect(status(page)).toHaveText(en.status.boardImageSaved.replace('{path}', '/home/user/board.svg'));
    expect((await getWailsCalls(page, 'SaveBoardSVG')).length).toBe(1);

    // Adding the board to a library still asks for one.
    await page.keyboard.press('Control+s');
    await expect(status(page)).toHaveText(en.commands.noDatabaseOpened);

    // EVAL → EDIT (the search board) → EVAL: the scratch board comes back.
    await page.click('[data-testid="tab-search"]');
    await expect(page.locator('[data-testid="tab-search"]')).toHaveClass(/active/);
    await expect(page.locator('.search-panel')).toBeVisible();
    // The query board is a scratch board too: a paste lands on it rather than asking for a library.
    await page.keyboard.press('Control+v');
    await expect.poll(async () => (await getWailsCalls(page, 'ParsePositionText')).length).toBe(2);
    await expect(status(page)).not.toHaveText(en.commands.noDatabaseOpened);
    await page.click('[data-testid="tab-eval"]');
    await expect(row(page, '8/5 6/5')).toBeVisible({ timeout: 4000 });

    // Nothing reached the database backend but pure computations, which read no file.
    const dbCalls = await page.evaluate(() => window.__wailsCalls.filter((c) => c.ns === 'database').map((c) => c.method));
    expect(dbCalls.filter((m) => !PURE_DATABASE_CALLS.includes(m))).toEqual([]);
});

test('a training session runs without a database, and is told, not recorded', async ({ page }) => {
    await installWailsMock(page);
    await page.goto('/');
    await expect(page.locator('[data-testid="status-bar"]')).toBeVisible({ timeout: 8000 });
    await dismissHomeScreen(page);
    await page.keyboard.press('Control+j');
    await expect(page.locator('[data-testid="tab-training"]')).toHaveClass(/active/);
    await expect(page.locator('[data-testid="training-journal-no-database"]')).toHaveText(en.training.journalNoDatabase);

    await page.click('[data-testid="training-exercise-scores"]');
    await page.click('[data-testid="training-start"]');
    await page.click('[data-testid="training-reveal"]');
    await page.click('[data-testid="training-finish"]');
    await expect(status(page)).toContainText(en.training.notRecorded.split(':')[0]);
    expect(await getWailsCalls(page, 'SaveTrainingSession')).toEqual([]);

    const dbCalls = await page.evaluate(() => window.__wailsCalls.filter((c) => c.ns === 'database').map((c) => c.method));
    expect(dbCalls.filter((m) => !PURE_DATABASE_CALLS.includes(m))).toEqual([]);
});
