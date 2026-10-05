/**
 * Le panneau garde sa boîte quand on change d'onglet : seule la poignée la change.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { showcaseGalleryMock } from './helpers/showcase.js';

test("changer d'onglet ne change pas la boîte du panneau", async ({ page }) => {
    test.setTimeout(30000);
    await installWailsMock(page, showcaseGalleryMock());
    await page.goto('/');
    const panel = page.locator('.panel-wrapper');
    await expect(panel).toBeVisible();
    const first = await panel.boundingBox();
    for (const id of ['stats', 'transcription', 'comment', 'search', 'analysis']) {
        const tab = page.getByTestId(`tab-${id}`);
        if (!(await tab.count())) continue;
        await tab.click();
        const box = await panel.boundingBox();
        expect(box.height, id).toBe(first.height);
        expect(box.width, id).toBe(first.width);
    }
});
