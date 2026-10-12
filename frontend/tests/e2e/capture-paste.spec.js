/**
 * Une image collée est une capture d'écran du plateau : elle est lue sur le plateau brouillon,
 * avec ce qu'elle laisse en doute, et rien n'est enregistré avant le geste habituel.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock, getWailsCalls } from './helpers/wailsMock.js';
import { openLibraryMock, pastedPosition } from './helpers/fixtures.js';

const statusBar = (page) => page.getByTestId('status-bar');
const statusMessage = (page) => page.getByTestId('status-bar-message');

test('coller une image lit la position sur le plateau d’édition sans rien enregistrer', async ({ page }) => {
    const reading = {
        position: pastedPosition,
        software: 'xg',
        fit: 0.97,
        unread: ['score'],
        doubts: [{ field: 'point', point: 7, reason: 'unclear stack' }]
    };
    await installWailsMock(page, openLibraryMock({ database: { ReadCaptureData: reading } }));
    await page.goto('/');
    await expect(statusBar(page)).toContainText('3 / 3');

    await page.evaluate(() => {
        const data = new DataTransfer();
        data.items.add(new File([new Uint8Array([137, 80, 78, 71])], 'shot.png', { type: 'image/png' }));
        window.dispatchEvent(new ClipboardEvent('paste', { clipboardData: data, bubbles: true, cancelable: true }));
    });

    await expect(statusMessage(page)).toContainText('Capture read onto the board');
    await expect(statusMessage(page)).toContainText('1 doubtful');
    await expect(page.getByTestId('board-situation-banner')).toContainText('Search board');

    await page.screenshot({ path: process.env.CAPTURE_SHOT ?? 'test-results/capture-paste.png' });

    expect(await getWailsCalls(page, 'ReadCaptureData')).toHaveLength(1);
    expect(await getWailsCalls(page, 'SaveIndividualPosition')).toHaveLength(0);
    expect(await getWailsCalls(page, 'SavePosition')).toHaveLength(0);
});
