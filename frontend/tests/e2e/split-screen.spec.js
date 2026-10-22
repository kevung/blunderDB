/**
 * split-screen.spec.js
 *
 * L'écran partagé (ADR-0101 §8) : seule la zone du plateau se coupe en deux volets, chacun attaché
 * à une bibliothèque sur un côté fixe. Le volet actif est vivant et cerné de la couleur de son
 * groupe ; l'autre est figé, atténué ; un clic ou CTRL-ALT-←/→ l'active, et rien ne change de côté.
 * Barre d'outils, panneaux et barre d'état restent en un exemplaire et disent toujours la base
 * active. Aucune donnée d'une base n'apparaît sous le nom de l'autre. Glisser une position d'un
 * volet sur l'autre la copie, avec « Annuler ». Sous 1600 px le panneau passe sous les plateaux ;
 * sous 1280 px l'écran partagé se refuse.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';

const pathB = '/libs/club.db';

async function openTwoGroups(page, width, height) {
    await page.setViewportSize({ width, height });
    await installWailsMock(
        page,
        openLibraryMock({
            app: {
                OpenDatabaseDialog: pathB,
                LibraryGroupHolding: 0,
                OpenLibraryGroup: { id: 2, path: pathB, active: true, readOnly: false },
                ActivateLibraryGroup: null,
                CloseLibraryGroup: null,
                ViewerCatalogOf: { matches: [], collections: [] },
                CopyPositionToGroup: { group: 1, positions: 1, matches: 0, alreadyHeld: 0, collections: 0, lessons: 0, errors: [], cancelled: false, created: 99 },
                UndoPositionCopy: null
            }
        })
    );
    await page.goto('/');
    await expect(page.locator('[data-testid="status-bar"]')).toBeVisible({ timeout: 8000 });
    await page.waitForFunction(() => performance.getEntriesByName('bdb:mounted').length > 0);
    await page.locator('[data-testid="open-library-group"]').click();
    await expect(page.locator('[data-testid="library-group"]').nth(1)).toHaveClass(/active/);
    await page.evaluate(() => {
        const points = (n) => Array.from({ length: 26 }, (_, i) => ({ checkers: i === 6 || i === 13 ? n : 0, color: i === 6 ? 0 : i === 13 ? 1 : -1 }));
        const position = (id, n) => ({
            id,
            board: { points: points(n), bearoff: [15 - n, 15 - n] },
            cube: { value: 0, owner: -1 },
            dice: [3, 1],
            score: [-1, -1],
            player_on_roll: 0,
            decision_type: 0
        });
        const app = window.go.gui.App;
        app.ViewerSteps = (group, src) =>
            Promise.resolve(
                src.kind === 'match'
                    ? [
                          { positionId: 11, gameNumber: 1, moveNumber: 1, player: 0, played: '8/5 6/5' },
                          { positionId: 12, gameNumber: 1, moveNumber: 2, player: 1, played: '24/18 13/7' }
                      ]
                    : group === 1
                      ? [{ positionId: 11 }, { positionId: 12 }, { positionId: 13 }]
                      : [{ positionId: 21 }]
            );
        app.ViewerShow = (group, id) =>
            Promise.resolve({
                position: position(id, group === 1 ? 5 : 3),
                analysis: {
                    analysisEngineVersion: 'GNU 1.07',
                    checkerAnalysis: {
                        moves: [
                            { move: '8/5 6/5', equity: 0.123 },
                            { move: '13/10 6/5', equity: 0.05, equityError: 0.073 }
                        ]
                    }
                },
                comment: id === 12 ? 'Le coach préfère construire.' : ''
            });
    });
}

/** The calls of method made since the mark, as their first argument. */
const callsOf = (page, method) => page.evaluate((m) => window.__wailsCalls.filter((c) => c.method === m).map((c) => c.args), method);

// Each library's own data: the club (group 2) holds one position, the e2e library (group 1) three.
const STEPS = { 1: '1 / 3', 2: '1 / 1' };
const NAMES = { 1: 'e2e-library', 2: 'club' };

/**
 * Everything on screen speaks for the library it names: the live pane, the panels' and the
 * status bar's marks and the window title name the active one; the frozen pane names the other
 * and shows that library's own walk, read under its own group number.
 */
async function expectCoherent(page, active, frozen, liveSide) {
    const live = page.locator('[data-testid="split-live"]');
    const viewer = page.locator('[data-testid="split-viewer"]');
    await expect(live).toHaveAttribute('data-side', liveSide);
    await expect(viewer).toHaveAttribute('data-side', liveSide === 'left' ? 'right' : 'left');
    await expect(live).toHaveAttribute('data-group', String(active));
    await expect(live.locator('[data-testid="split-name"]')).toHaveText(NAMES[active]);
    await expect(viewer).toHaveAttribute('data-group', String(frozen));
    await expect(viewer.locator('[data-testid="split-name"]')).toHaveText(NAMES[frozen]);
    await expect(viewer.locator('[data-testid="split-count"]')).toHaveText(STEPS[frozen]);
    await expect(page.locator('[data-testid="status-library"]')).toHaveText(NAMES[active]);
    await expect(page.locator('[data-testid="panel-library"]')).toHaveText(NAMES[active]);
    // The frozen pane read its walk and its position under its own group, never the other's.
    const steps = await callsOf(page, 'ViewerSteps');
    expect(steps.at(-1)[0]).toBe(frozen);
    await expect.poll(async () => (await callsOf(page, 'ViewerShow')).at(-1)?.[0]).toBe(frozen);
    const titles = await callsOf(page, 'WindowSetTitle');
    if (titles.length) expect(String(titles.at(-1)[0])).toContain(NAMES[active]);
}

test('each library keeps its side; alternating the panes keeps every mark on the active library, at 1366 px', async ({ page }) => {
    await openTwoGroups(page, 1366, 768);
    await page.evaluate(() => /** @type {HTMLElement | null} */ (document.activeElement)?.blur());
    await page.keyboard.press('Control+Shift+X');
    await expect(page.locator('[data-testid="split-viewer"]')).toBeVisible();
    await expectCoherent(page, 2, 1, 'left');

    // Only the board area is cut: one toolbar, one status bar, the panel under the boards.
    await expect(page.locator('[data-testid="status-bar"]')).toHaveCount(1);
    await expect(page.locator('.panel-wrapper')).not.toHaveClass(/side/);
    const live = await page.locator('[data-testid="split-live"]').boundingBox();
    const frozen = await page.locator('[data-testid="split-viewer"]').boundingBox();
    expect(Math.abs(live.width - frozen.width)).toBeLessThanOrEqual(4);
    expect(live.x + live.width).toBeLessThanOrEqual(frozen.x + 1);
    await expect(page.locator('#backgammon-board')).toBeVisible();
    expect(await page.locator('main.main-container').evaluate((el) => el.scrollWidth - el.clientWidth)).toBeLessThanOrEqual(0);
    await page.screenshot({ path: test.info().outputPath('split-1366.png') });

    // A click on the frozen pane activates its library; nothing changes side.
    await page.locator('[data-testid="split-viewer"] .board').click();
    await expect.poll(() => callsOf(page, 'ActivateLibraryGroup')).toEqual([[1]]);
    await expectCoherent(page, 1, 2, 'right');

    // The keyboard: Ctrl+Alt+← brings the left pane live, Ctrl+Alt+→ the right one.
    await page.evaluate(() => /** @type {HTMLElement | null} */ (document.activeElement)?.blur());
    await page.keyboard.press('Control+Alt+ArrowLeft');
    await expect.poll(() => callsOf(page, 'ActivateLibraryGroup')).toEqual([[1], [2]]);
    await expectCoherent(page, 2, 1, 'left');
    // The live pane's key does nothing.
    await page.keyboard.press('Control+Alt+ArrowLeft');
    await page.keyboard.press('Control+Alt+ArrowRight');
    await expect.poll(() => callsOf(page, 'ActivateLibraryGroup')).toEqual([[1], [2], [1]]);
    await expectCoherent(page, 1, 2, 'right');
    expect(await page.evaluate(() => JSON.parse(localStorage.getItem('blunderdb.split')))).toEqual({ on: true, left: '/libs/club.db', right: expect.stringContaining('e2e') });
});

test('a position dragged from one pane onto the other is copied there, and can be taken back', async ({ page }) => {
    await openTwoGroups(page, 1366, 768);
    await page.locator('[data-testid="toggle-split"]').click();
    await expect(page.locator('[data-testid="split-viewer"] [data-testid="split-count"]')).toHaveText('1 / 3');
    const from = await page.locator('[data-testid="split-viewer"] .board').boundingBox();
    const to = await page.locator('[data-testid="split-live"]').boundingBox();
    await page.mouse.move(from.x + from.width / 2, from.y + from.height / 2);
    await page.mouse.down();
    await page.mouse.move(from.x + from.width / 2 - 40, from.y + from.height / 2, { steps: 4 });
    await page.mouse.move(to.x + to.width / 2, to.y + to.height / 2, { steps: 8 });
    await page.mouse.up();
    await expect.poll(() => callsOf(page, 'CopyPositionToGroup')).toEqual([[1, 11, 2]]);
    // The drop activated nothing.
    expect(await callsOf(page, 'ActivateLibraryGroup')).toEqual([]);
    await page.locator('[data-testid="transfer-undo"]').click();
    await expect.poll(() => callsOf(page, 'UndoPositionCopy')).toEqual([[2, 99]]);
    await expect(page.locator('[data-testid="transfer-undo"]')).toHaveCount(0);
});

test('at 2560 px the panel stays beside the two boards, both readable', async ({ page }) => {
    await openTwoGroups(page, 2560, 1440);
    await page.locator('[data-testid="toggle-split"]').click();
    const viewer = page.locator('[data-testid="split-viewer"]');
    await expect(viewer.locator('[data-testid="split-count"]')).toHaveText('1 / 3');
    await expect(page.locator('.panel-wrapper')).toHaveClass(/side/);
    await expect(viewer.locator('.board svg').first()).toBeVisible();
    expect((await viewer.locator('.board svg').first().boundingBox()).width).toBeGreaterThan(500);
    // The live board refits to its pane: none of it under the frozen one.
    const pane = await page.locator('[data-testid="split-live"]').boundingBox();
    await expect
        .poll(async () => {
            const b = await page.locator('#backgammon-board svg').first().boundingBox();
            return b.x + b.width;
        })
        .toBeLessThanOrEqual(pane.x + pane.width + 1);
    await page.screenshot({ path: test.info().outputPath('split-2560.png') });
});

for (const width of [1600, 2560]) {
    test(`at ${width} px the panel beside the boards keeps its width`, async ({ page }) => {
        await openTwoGroups(page, width, 1000);
        await page.locator('[data-testid="toggle-split"]').click();
        await expect(page.locator('[data-testid="split-viewer"] [data-testid="split-count"]')).toHaveText('1 / 3');
        const panel = page.locator('.panel-wrapper');
        await expect(panel).toHaveClass(/side/);
        await expect.poll(async () => Math.round((await panel.boundingBox()).width)).toBeGreaterThanOrEqual(415);
        // The two boards share what is left, beside the panel and not under it.
        const pane = await page.locator('[data-testid="split-live"]').boundingBox();
        expect(pane.x + pane.width).toBeLessThanOrEqual((await panel.boundingBox()).x + 1);
    });
}

test('a narrow window refuses the split and says why', async ({ page }) => {
    await openTwoGroups(page, 1100, 768);
    await page.locator('[data-testid="toggle-split"]').click();
    await expect(page.locator('[data-testid="split-viewer"]')).toHaveCount(0);
    await expect(page.locator('[data-testid="status-bar"]')).toContainText('1280');
});
