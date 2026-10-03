/**
 * stats-tabs.spec.js — les six onglets du panneau Stats
 *
 * Chaque onglet s'ouvre et rend son contenu ; les agrégats coûteux (joueurs, erreurs
 * récurrentes, entraînement) ne sont demandés au backend qu'à l'ouverture de leur onglet,
 * jamais derrière un autre.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock, getWailsCalls } from './helpers/wailsMock.js';
import { openLibraryMock, statsResult } from './helpers/fixtures.js';

const TABS = ['Dashboard', 'Progression', 'Errors', 'Training', 'Breakdowns', 'Players'];

test.beforeEach(async ({ page }) => {
    await installWailsMock(page, openLibraryMock({ database: { ComputeStats: statsResult, GetPlayerTable: [], ComputeRecurringErrors: [] } }));
    await page.goto('/');
    await expect(page.getByTestId('status-bar')).toContainText('3 / 3');
    await page.getByTestId('tab-stats').click();
    await expect(page.getByTestId('tab-stats')).toHaveClass(/active/);
});

const statsTab = (page, label) => page.locator('.stats-panel [role="tab"]', { hasText: label });

test('chaque onglet s’ouvre et rend son contenu', async ({ page }) => {
    await expect(statsTab(page, 'Dashboard')).toHaveAttribute('aria-selected', 'true');
    for (const label of TABS) {
        await statsTab(page, label).click();
        await expect(statsTab(page, label)).toHaveAttribute('aria-selected', 'true');
        await expect(page.locator('.stats-panel [role="tabpanel"]')).not.toBeEmpty();
    }
    await expect(page.locator('.stats-panel [role="tab"][aria-selected="true"]')).toHaveCount(1);
});

test('un agrégat coûteux n’est demandé qu’à l’ouverture de son onglet', async ({ page }) => {
    await expect.poll(async () => (await getWailsCalls(page, 'ComputeStats')).length).toBeGreaterThan(0);
    for (const method of ['GetPlayerTable', 'ComputeRecurringErrors', 'ComputeTrainingStats']) {
        expect(await getWailsCalls(page, method), method).toHaveLength(0);
    }

    await statsTab(page, 'Players').click();
    await expect.poll(async () => (await getWailsCalls(page, 'GetPlayerTable')).length).toBe(1);
    await statsTab(page, 'Errors').click();
    await expect.poll(async () => (await getWailsCalls(page, 'ComputeRecurringErrors')).length).toBe(1);
    await statsTab(page, 'Training').click();
    await expect.poll(async () => (await getWailsCalls(page, 'ComputeTrainingStats')).length).toBe(1);
});
