/**
 * split-screen.spec.js
 *
 * L'écran partagé : deux volets de même largeur, le groupe actif dans l'un, une visionneuse en
 * lecture de l'autre bibliothèque dans l'autre (plateau, analyse, commentaire, navigation, match
 * rejoué coup par coup). Cliquer la visionneuse active son groupe : les volets échangent leurs
 * rôles, chaque bibliothèque garde son côté. Sous la largeur minimale, l'écran partagé se refuse et
 * propose l'échange sur place. Lisible à 1366 px et à 2560 px.
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
                ViewerCatalogOf: { matches: [{ id: 3, label: 'Alice – Bruno (7)', count: 4 }], collections: [{ id: 8, label: 'Primes', count: 2 }] }
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

test('the split shows a viewer of the other library beside the live one, at 1366 px', async ({ page }) => {
    await openTwoGroups(page, 1366, 768);
    await page.evaluate(() => /** @type {HTMLElement | null} */ (document.activeElement)?.blur());
    await page.keyboard.press('Control+Shift+X');
    const viewer = page.locator('[data-testid="split-viewer"]');
    await expect(viewer).toBeVisible();
    await expect(viewer).toContainText('e2e-library');
    await expect(viewer.locator('[data-testid="split-count"]')).toHaveText('1 / 3');
    await expect(viewer.locator('[data-testid="split-info"]')).toContainText('8/5 6/5');

    // Two panes of the same width, side by side.
    const v = await viewer.boundingBox();
    const live = await page.locator('main.main-container').boundingBox();
    expect(Math.abs(v.width - live.width)).toBeLessThanOrEqual(2);
    expect(live.x + live.width).toBeLessThanOrEqual(v.x + 1);
    await expect(page.locator('#backgammon-board')).toBeVisible();
    await page.screenshot({ path: test.info().outputPath('split-1366-open.png') });
    expect(await page.locator('main.main-container').evaluate((el) => el.scrollWidth - el.clientWidth)).toBeLessThanOrEqual(0);
    expect(await viewer.evaluate((el) => el.scrollWidth - el.clientWidth)).toBeLessThanOrEqual(0);

    await viewer.getByRole('button', { name: 'Next' }).click();
    await expect(viewer.locator('[data-testid="split-count"]')).toHaveText('2 / 3');
    await expect(viewer.locator('[data-testid="split-info"]')).toContainText('Le coach préfère construire.');

    // A match, replayed move by move.
    await viewer.locator('[data-testid="split-source"]').selectOption('match:3');
    await expect(viewer.locator('[data-testid="split-played"]')).toContainText('8/5 6/5');
    await viewer.getByRole('button', { name: 'Next' }).click();
    await expect(viewer.locator('[data-testid="split-played"]')).toContainText('24/18 13/7');
    await page.screenshot({ path: test.info().outputPath('split-1366.png') });

    // Navigating the viewer activated nothing: the live group is still the club's.
    expect(await page.evaluate(() => window.__wailsCalls.some((c) => c.method === 'ActivateLibraryGroup'))).toBe(false);

    // Clicking the viewer activates its library; the club's moves to the viewer, on the other side.
    await viewer.locator('.board').click();
    await expect.poll(() => page.evaluate(() => window.__wailsCalls.filter((c) => c.method === 'ActivateLibraryGroup').map((c) => c.args[0]))).toEqual([1]);
    await expect(viewer).toContainText('club');
    await expect(viewer).toHaveClass(/left/);
    const after = await page.locator('main.main-container').boundingBox();
    expect(after.x).toBeGreaterThan(600);
});

test('the split stays readable at 2560 px', async ({ page }) => {
    await openTwoGroups(page, 2560, 1440);
    await page.locator('[data-testid="toggle-split"]').click();
    const viewer = page.locator('[data-testid="split-viewer"]');
    await expect(viewer.locator('[data-testid="split-count"]')).toHaveText('1 / 3');
    await expect(viewer.locator('svg').first()).toBeVisible();
    const boardBox = await viewer.locator('.board svg').first().boundingBox();
    expect(boardBox.width).toBeGreaterThan(800);
    // The live board refits to its half: none of it under the viewer.
    const main = await page.locator('main.main-container').boundingBox();
    await expect
        .poll(async () => (await page.locator('#backgammon-board svg').first().boundingBox()).x + (await page.locator('#backgammon-board svg').first().boundingBox()).width)
        .toBeLessThanOrEqual(main.x + main.width + 1);
    await page.screenshot({ path: test.info().outputPath('split-2560.png') });
});

test('a narrow window refuses the split and offers the in-place swap', async ({ page }) => {
    await openTwoGroups(page, 1100, 768);
    await page.locator('[data-testid="toggle-split"]').click();
    await expect(page.locator('[data-testid="split-viewer"]')).toHaveCount(0);
    await expect(page.locator('[data-testid="status-bar"]')).toContainText('1280');
});
