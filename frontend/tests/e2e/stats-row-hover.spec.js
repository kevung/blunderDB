/**
 * stats-row-hover.spec.js — la ligne survolée d'un tableau du panneau Stats
 *
 * Les tableaux des onglets sont écrits à la main : la règle vit dans StatsPanel. La ligne
 * survolée prend la teinte --color-row-hover, les autres restent transparentes.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock, statsResult } from './helpers/fixtures.js';

test('la ligne survolée d’un tableau de Breakdowns est teintée', async ({ page }) => {
    await installWailsMock(
        page,
        openLibraryMock({
            database: {
                ComputeStats: {
                    ...statsResult,
                    PerPhase: [
                        { Phase: 'opening', Decisions: 10, PR: 3 },
                        { Phase: 'middle', Decisions: 20, PR: 4 }
                    ]
                },
                GetPlayerTable: [],
                ComputeRecurringErrors: [],
                GetAllPlayerNames: [{ Name: 'Alice', Count: 30 }]
            }
        })
    );
    await page.goto('/');
    await expect(page.getByTestId('status-bar')).toContainText('3 / 3');
    await page.getByTestId('tab-stats').click();
    await page.locator('.stats-panel [role="tab"]', { hasText: 'Breakdowns' }).click();

    const rows = page.locator('.stats-panel tbody tr');
    await expect(rows.first()).toBeVisible();
    const bg = (i) => rows.nth(i).evaluate((el) => getComputedStyle(el).backgroundColor);
    const before = await bg(0);
    await rows.first().hover();
    await expect.poll(() => bg(0)).not.toBe(before);
    expect(await bg(1)).toBe(before);
    await page.screenshot({ path: 'test-results/stats-row-hover.png' });
});
