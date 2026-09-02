/**
 * import-position.spec.js — flux Import d'une position
 *
 * 1. Coller un XGID (Ctrl+V) : le presse-papier et ParsePositionText sont
 *    mockés ; la position est enregistrée (SaveIndividualPosition) et
 *    apparaît dans la bibliothèque, affichée sur l'onglet Analyse.
 * 2. Importer un fichier (Ctrl+I) via le mock de dialogue.
 * 3. Dialogue annulé : rien ne change.
 *
 * Le mock simule la mutation du backend : dès que l'écriture est appelée,
 * LoadAllPositions renvoie la bibliothèque agrandie.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock, overrideDbMethodThen, getWailsCalls } from './helpers/wailsMock.js';
import { openLibraryMock, libraryPositions, pastedPosition, xgidSample, parsedPositionResult } from './helpers/fixtures.js';

const statusBar = (page) => page.getByTestId('status-bar');
const statusMessage = (page) => page.getByTestId('status-bar-message');

test('coller un XGID enregistre la position et l’affiche dans la bibliothèque', async ({ page }) => {
    await installWailsMock(
        page,
        openLibraryMock({
            runtime: { ClipboardGetText: xgidSample },
            database: { ParsePositionText: parsedPositionResult }
        })
    );
    await page.goto('/');
    await expect(statusBar(page)).toContainText('3 / 3');

    const saved = { ...pastedPosition, id: 1004 };
    await overrideDbMethodThen(page, 'SaveIndividualPosition', { id: 1004, existed: false }, { LoadAllPositions: [...libraryPositions, saved] });

    await page.keyboard.press('Control+v');

    await expect(statusMessage(page)).toHaveText('Pasted position and analysis saved successfully');
    await expect(statusBar(page)).toContainText('4 / 4');
    await expect(page.getByTestId('tab-analysis')).toHaveClass(/active/);

    const parsed = await getWailsCalls(page, 'ParsePositionText');
    expect(parsed.map((c) => c.args[0])).toEqual([xgidSample]);
    const writes = await getWailsCalls(page, 'SaveIndividualPosition');
    expect(writes).toHaveLength(1);
    expect(writes[0].args[0].board.points).toEqual(pastedPosition.board.points);
    expect(writes[0].args[0].dice).toEqual([3, 1]);
    // La position affichée est bien la nouvelle (son analyse a été demandée)
    const analysed = (await getWailsCalls(page, 'LoadAnalysis')).map((c) => c.args[0]);
    expect(analysed.at(-1)).toBe(1004);
});

test('importer un fichier de position via le dialogue', async ({ page }) => {
    await installWailsMock(page, openLibraryMock({ app: { OpenPositionFilesDialog: ['/tmp/pos.xgp'] } }));
    await page.goto('/');
    await expect(statusBar(page)).toContainText('3 / 3');

    const imported = { ...pastedPosition, id: 1005 };
    await overrideDbMethodThen(page, 'ImportXGPPosition', 1005, { LoadAllPositions: [...libraryPositions, imported] });

    await page.keyboard.press('Control+i');

    await expect(statusMessage(page)).toHaveText('XGP position imported successfully (ID: 1005)');
    await expect(statusBar(page)).toContainText('4 / 4');
    await expect(page.getByTestId('tab-analysis')).toHaveClass(/active/);

    const calls = await getWailsCalls(page, 'ImportXGPPosition');
    expect(calls.map((c) => c.args[0])).toEqual(['/tmp/pos.xgp']);
    const analysed = (await getWailsCalls(page, 'LoadAnalysis')).map((c) => c.args[0]);
    expect(analysed.at(-1)).toBe(1005);
});

test('dialogue d’import annulé : la bibliothèque ne bouge pas', async ({ page }) => {
    await installWailsMock(page, openLibraryMock({ app: { OpenPositionFilesDialog: [] } }));
    await page.goto('/');
    await expect(statusBar(page)).toContainText('3 / 3');

    await page.keyboard.press('Control+i');

    await expect.poll(async () => (await getWailsCalls(page, 'OpenPositionFilesDialog')).length).toBe(1);
    expect(await getWailsCalls(page, 'ImportXGPPosition')).toHaveLength(0);
    await expect(statusBar(page)).toContainText('3 / 3');
});
