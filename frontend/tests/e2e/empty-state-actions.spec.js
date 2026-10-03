/**
 * empty-state-actions.spec.js — un état vide propose un geste
 *
 * Sans base ouverte, après « Continuer sans ouvrir de base », le panneau Stats
 * (filtre vide) et la liste des matchs offrent « Import… » et le retour à
 * l'accueil ; avec une base ouverte et sans match, seul l'import reste.
 */

import { test, expect } from '@playwright/test';
import { dismissHomeScreen, installWailsMock, getWailsCalls } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';

test("sans base : la liste des matchs propose l'import et le retour à l'accueil", async ({ page }) => {
    await installWailsMock(page);
    await page.goto('/');
    await dismissHomeScreen(page);
    await page.getByTestId('tab-matches').click();

    const empty = page.getByTestId('empty-state');
    // Import needs a library: without one the way forward is to open one.
    await expect(empty.getByRole('button', { name: 'Import… (Ctrl+I)' })).toHaveCount(0);
    await empty.getByRole('button', { name: 'Open a database…' }).click();
    await expect.poll(async () => (await getWailsCalls(page, 'OpenDatabaseDialog')).length).toBe(1);
    await empty.getByRole('button', { name: 'Back to the welcome screen' }).click();
    await expect(page.getByTestId('home-dismiss')).toBeVisible();
});

test("base ouverte sans match : seul l'import est proposé", async ({ page }) => {
    await installWailsMock(page, openLibraryMock());
    await page.goto('/');
    await expect(page.getByTestId('status-bar')).toContainText('3 / 3');
    await page.getByTestId('tab-matches').click();

    const empty = page.getByTestId('empty-state');
    await expect(empty).toContainText('No matches imported yet');
    await expect(empty.getByRole('button', { name: 'Import… (Ctrl+I)' })).toBeVisible();
    await expect(empty.getByRole('button', { name: 'Back to the welcome screen' })).toHaveCount(0);
});

test("liste vidée par le filtre texte : on propose d'effacer le filtre, pas d'importer", async ({ page }) => {
    await installWailsMock(page, openLibraryMock());
    await page.goto('/');
    await expect(page.getByTestId('status-bar')).toContainText('3 / 3');
    await page.getByTestId('tab-matches').click();

    await page.locator('.match-list-toolbar input').fill('zzz');
    const empty = page.getByTestId('empty-state');
    await expect(empty.getByRole('button', { name: 'Import… (Ctrl+I)' })).toHaveCount(0);
    await empty.getByRole('button', { name: 'Clear the filter' }).click();
    await expect(page.locator('.match-list-toolbar input')).toHaveValue('');
});
