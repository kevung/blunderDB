/**
 * Le zoom de l'interface ne doit changer ni le pas des lignes de la liste fenêtrée ni leur
 * présence : le pas se mesure en pixels CSS de la table, ceux qui servent à l'écrire.
 * La poignée du volet suit le pointeur à tout zoom, et sa largeur se retrouve au retour.
 */
import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock, matchSample } from './helpers/fixtures.js';

const matches = Array.from({ length: 300 }, (_, i) => ({ ...matchSample, id: i + 1, player1_name: `Alice ${i}` }));

test.beforeEach(async ({ page }) => {
    await installWailsMock(page, openLibraryMock({ database: { GetAllMatches: matches }, config: { GetPanelWidth: 420 } }));
    await page.goto('/');
    await expect(page.getByTestId('status-bar')).toContainText('3 / 3');
});

test('les lignes survivent au zoom et au dézoom', async ({ page }) => {
    const panel = page.getByRole('region', { name: 'Match navigator' });
    await expect(panel).toBeVisible();
    const row = () => panel.locator('tbody tr:not(.spacer)').first();
    await expect(row()).toBeVisible();
    const pitch = () => row().evaluate((el) => el.offsetHeight);
    const before = await pitch();

    for (let i = 0; i < 5; i++) {
        await page.keyboard.press('Control+=');
        // Ce qui relance la mesure : un défilement renouvelle la fenêtre de lignes.
        await panel.locator('.scroll').evaluate((el) => (el.scrollTop += 90));
        await page.waitForTimeout(100);
        expect(await pitch()).toBeLessThan(before * 1.2);
    }
    for (let i = 0; i < 5; i++) await page.keyboard.press('Control+-');
    await panel.locator('.scroll').evaluate((el) => (el.scrollTop = 0));
    await expect(row()).toBeVisible();
    expect(Math.abs((await pitch()) - before)).toBeLessThanOrEqual(1);
    await expect(panel.locator('tbody tr:not(.spacer)').nth(3)).toBeInViewport();
});

test('la poignée règle la largeur de la fiche, qui se retrouve au rechargement', async ({ page }) => {
    const panel = page.getByRole('region', { name: 'Match navigator' });
    await panel.locator('tbody tr:not(.spacer)').first().getByRole('cell', { name: /Alice/ }).click();
    const handle = page.getByTestId('match-sheet-handle');
    await expect(handle).toBeVisible();
    await page.keyboard.press('Control+=');
    await page.keyboard.press('Control+=');
    const sheet = panel.locator('.sheet-slot');
    const list = panel.locator('.match-list-pane');
    const start = await sheet.boundingBox();
    const box = await handle.boundingBox();

    await page.mouse.move(box.x, box.y + 40);
    await page.mouse.down();
    await page.mouse.move(box.x - 120, box.y + 40, { steps: 4 });
    await page.mouse.up();

    const grown = await sheet.boundingBox();
    expect(Math.abs(grown.width - start.width - 120)).toBeLessThanOrEqual(2);
    // La liste rend ce que la fiche prend ; la poignée ne pousse rien hors du panneau.
    const listBox = await list.boundingBox();
    expect(Math.abs(listBox.x + listBox.width - (grown.x - 1))).toBeLessThanOrEqual(2);

    const share = await page.evaluate(() => localStorage.getItem('blunderdb.matchSheetShare'));
    expect(Number(share)).toBeGreaterThan(0.55);

    await page.goto('/');
    await panel.locator('tbody tr:not(.spacer)').first().getByRole('cell', { name: /Alice/ }).click();
    await expect(sheet).toBeVisible();
    const content = await panel.locator('.match-panel-content').boundingBox();
    expect(Math.abs((await sheet.boundingBox()).width / content.width - Number(share))).toBeLessThan(0.01);
});
