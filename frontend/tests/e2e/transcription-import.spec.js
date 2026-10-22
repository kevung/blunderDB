/**
 * Importer un brouillon depuis l'écran de transcription : le fichier choisi est lu, son texte part
 * à ImportTranscription, le brouillon s'ouvre ; un refus s'affiche et n'ouvre rien.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock, getWailsCalls } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';
import { installTranscriptionEngine } from './helpers/transcriptionDraft.js';

/** @param {import('@playwright/test').Page} page @param {string} content */
async function setup(page, content) {
    await installWailsMock(page, openLibraryMock());
    await installTranscriptionEngine(page);
    await page.addInitScript((content) => {
        const db = window.go.database.Database;
        const app = window.go.gui.App;
        const open = db.OpenTranscription;
        app.PickTranscriptionFile = () => Promise.resolve('/tmp/match.mat');
        app.ReadFileContent = () => Promise.resolve({ content, error: '' });
        db.ImportTranscription = (text) => (text.startsWith('{') || /match/.test(text) ? open(1) : Promise.reject('not a transcription'));
    }, content);
    await page.goto('/');
    await page.locator('[data-testid="tab-transcription"]').click();
}

test('un fichier valide crée le brouillon et l’ouvre', async ({ page }) => {
    await setup(page, '7 point match');
    await page.locator('#transcriptionImport').click();
    await expect(page.locator('#transcriptionPanel .draft-bar')).toBeVisible();
    const calls = await getWailsCalls(page, 'ImportTranscription');
    expect(calls[0].args[0]).toBe('7 point match');
});

test('un refus affiche le message sans ouvrir de brouillon', async ({ page }) => {
    await setup(page, 'garbage');
    await page.locator('#transcriptionImport').click();
    await expect(page.locator('#transcriptionPanel .error')).toContainText('not a transcription');
    await expect(page.locator('#transcriptionPanel .draft-bar')).toHaveCount(0);
});
