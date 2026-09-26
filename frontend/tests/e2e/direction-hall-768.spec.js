/**
 * La grille des tables reste à portée à 768 px de haut : à 1024×768, dock ouvert à 250 px, la
 * page Direction n'a que ~370 px utiles, et une file de propositions assez longue pousse la
 * grille des tables entièrement sous l'écran.
 *
 * Un cran ne se compte pas par un clic : Playwright fait défiler tout seul jusqu'à sa cible, y
 * compris un conteneur `overflow: hidden` qu'un utilisateur ne peut pas faire défiler. Une
 * cible « à 0 cran » est une cible dont la boîte est ENTIÈREMENT dans la zone visible de
 * `.direction-view .body` et de la fenêtre, lue par getBoundingClientRect avant le clic, sans
 * que rien n'ait défilé (`scrollTop` de chaque ancêtre relevé avant et après le geste).
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';
import { countGestures } from './helpers/gestureCount.js';
import { installDirectionEngine, S2_HALL } from './helpers/directionEngine.js';

const VIEWPORT = { width: 1024, height: 768 };
// Le dock bas à sa hauteur par défaut. Sans ces deux valeurs, le mock rend null et le dock
// prend une hauteur automatique qui écrase la page : ce n'est pas ce qu'un directeur voit.
const DOCK = { GetPanelHeight: 250, GetPanelPosition: 'bottom' };

async function openHall(page) {
    await page.setViewportSize(VIEWPORT);
    await installWailsMock(page, openLibraryMock({ config: DOCK }));
    await installDirectionEngine(page, S2_HALL);
    await page.goto('/');
    await page.locator('[data-testid="tab-tournaments"]').click();
    await expect(page.locator('#tournamentPanel')).toBeVisible();
    await page.locator('#tournamentPanel tbody tr').first().click();
    await page.locator('#tournamentPanel .direction-btn').click();
    await expect(page.locator('.direction-view')).toBeVisible();
    await page.locator('[data-testid="direction-tab-direction"]').click();
    await expect(page.locator('.grid .cell.busy')).toHaveCount(14);
    await expect(page.locator('.proposals .queue li').first()).toBeVisible();
}

/** Le `scrollTop` de tout ce qui peut défiler entre la page et la racine. */
async function scrollState(page) {
    return page.evaluate(() => {
        const out = [window.scrollY];
        for (const el of document.querySelectorAll('*')) if (el.scrollTop) out.push(`${el.className}:${el.scrollTop}`);
        return out.join('|');
    });
}

/**
 * La cible est-elle visible sans défiler ? Sa boîte entière dans `.direction-view .body` ET
 * dans la fenêtre. Rend la boîte pour que l'échec dise où elle était.
 */
async function reachable(locator) {
    return locator.evaluate((el) => {
        const r = el.getBoundingClientRect();
        const body = document.querySelector('.direction-view .body').getBoundingClientRect();
        const top = Math.max(body.top, 0);
        const bottom = Math.min(body.bottom, window.innerHeight);
        return { ok: r.height > 0 && r.top >= top - 0.5 && r.bottom <= bottom + 0.5, y: Math.round(r.top), bottom: Math.round(r.bottom), bodyTop: Math.round(top), bodyBottom: Math.round(bottom) };
    });
}

async function expectReachable(locator, what) {
    const box = await reachable(locator);
    expect(box.ok, `${what} : y ${box.y} → ${box.bottom}, zone visible ${box.bodyTop} → ${box.bodyBottom}`).toBe(true);
}

test.describe('#440 — 24 propositions, 14 tables, 1024×768, dock ouvert', () => {
    test('un résultat sur n’importe quelle table : 2 clics, 0 cran', async ({ page }) => {
        await openHall(page);
        for (const table of [1, 7, 12, 14]) {
            const scroll = await scrollState(page);
            const cell = page.locator(`[data-testid="direction-table-${table}"]`);
            await expectReachable(cell, `table ${table}`);
            const busyBefore = await page.locator('.grid .cell.busy').count();

            const counted = await countGestures(page, async (g) => {
                await g.click(cell);
                const winner = page.locator('[data-testid="direction-result-winner-a"]');
                await expect(winner).toBeVisible();
                await expectReachable(winner, `vainqueur de la table ${table}`);
                await g.click(winner);
                await expect(page.locator('.grid .cell.busy')).toHaveCount(busyBefore - 1);
            });
            expect(counted.total, `résultat table ${table} : ${counted.total} gestes pour 2`).toBeLessThanOrEqual(2);
            expect(await scrollState(page), `résultat table ${table} : la page a défilé`).toBe(scroll);
        }
    });

    test('« Tout lancer » avec 24 propositions : 2 clics, 0 cran', async ({ page }) => {
        await openHall(page);
        const scroll = await scrollState(page);
        const all = page.locator('[data-testid="direction-proposals-all"]');
        await expectReachable(all, '« Tout lancer »');

        const counted = await countGestures(page, async (g) => {
            await g.click(all);
            const confirm = page.locator('[data-testid="direction-proposals-confirm-all"]');
            await expect(confirm).toBeVisible();
            await expectReachable(confirm, '« Confirmer »');
            await g.click(confirm);
            await expect(page.locator('.proposals .queue li.empty, .proposals .queue li')).not.toHaveCount(24);
        });
        expect(counted.total, `tout lancer : ${counted.total} gestes pour 2`).toBeLessThanOrEqual(2);
        expect(await scrollState(page), 'tout lancer : la page a défilé').toBe(scroll);
    });

    // Après un résultat, la dernière décision s'intercale entre la grille et la file : c'est
    // elle qui pousse « Tout lancer » vers le bas, et le budget doit tenir quand même.
    test('« Tout lancer » reste à portée sous la dernière décision', async ({ page }) => {
        await openHall(page);
        await page.locator('[data-testid="direction-table-14"]').click();
        await page.locator('[data-testid="direction-result-winner-a"]').click();
        await expect(page.locator('[data-testid="direction-last"]')).toBeVisible();
        const scroll = await scrollState(page);
        const all = page.locator('[data-testid="direction-proposals-all"]');
        await expectReachable(all, '« Tout lancer »');
        await all.click();
        await expectReachable(page.locator('[data-testid="direction-proposals-confirm-all"]'), '« Confirmer »');
        expect(await scrollState(page), 'tout lancer : la page a défilé').toBe(scroll);
    });

    // J / K descendent dans une file plus longue que l'écran : la ligne choisie vient à l'écran,
    // et ENTRÉE confirme celle qu'on voit.
    test('J amène la proposition choisie à l’écran', async ({ page }) => {
        await openHall(page);
        for (let i = 0; i < 10; i++) await page.keyboard.press('j');
        const row = page.locator('.proposals .queue li').nth(10);
        await expect(row).toHaveClass(/selected/);
        await expectReachable(row, 'onzième proposition choisie au clavier');
    });

    test('la feuille d’appariements et la file restent à portée', async ({ page }) => {
        await openHall(page);
        await expectReachable(page.locator('[data-testid="direction-sheet-print"]'), '« Imprimer la feuille »');
        await expectReachable(page.locator('.proposals .queue li').first(), 'première proposition');
    });
});
