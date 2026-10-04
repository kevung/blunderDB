/**
 * import-watched.spec.js — un fichier arrive dans le dossier surveillé
 *
 * L'import se fait sans fenêtre : un bandeau dans la barre d'état en donne le compte. Celui
 * qui étudie une position n'en est pas éjecté : position courante et onglet restent (un
 * rechargement l'aurait renvoyé à la dernière position, sur l'onglet Match).
 * L'événement `folder-watch:files` du backend est simulé en capturant son abonnement.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock, getWailsCalls } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';

const statusBar = (page) => page.getByTestId('status-bar');

test('un import surveillé laisse la position et l’onglet en place', async ({ page }) => {
    // Les .xg passent par le pipeline du backend, en un seul appel pour toute la liste.
    await installWailsMock(
        page,
        openLibraryMock({
            app: { ImportFiles: { succeeded: 1, skipped: 1, failed: 0, errors: [], hadMatches: true, lastPositionID: 0, cancelled: false } }
        })
    );
    await page.addInitScript(() => {
        /** @type {Record<string, Function>} */
        window.__events = {};
        window.runtime.EventsOnMultiple = (name, cb) => {
            window.__events[name] = cb;
            return () => {};
        };
    });
    await page.goto('/');
    await expect(statusBar(page)).toContainText('3 / 3');

    // L'utilisateur étudie la première position, onglet Analyse.
    await page.keyboard.press('Escape');
    await page.keyboard.press('Home');
    await expect(statusBar(page)).toContainText('1 / 3');
    await page.getByTestId('tab-analysis').click();
    await expect(page.getByTestId('tab-analysis')).toHaveClass(/active/);

    await page.evaluate(() => window.__events['folder-watch:files'](['/tmp/watch/new.xg', '/tmp/watch/dup.xg']));

    await expect(statusBar(page)).toContainText('Watched folder: 1 imported, 1 skipped, 0 failed');
    await expect(statusBar(page)).toContainText('1 / 3');
    await expect(page.getByTestId('tab-analysis')).toHaveClass(/active/);
    // Ni fenêtre de progression, ni rapport.
    await expect(page.getByText('Import Completed')).toHaveCount(0);
    await expect(page.getByText('Importing file')).toHaveCount(0);

    const imported = await getWailsCalls(page, 'ImportFiles');
    expect(imported.map((c) => c.args[0])).toEqual([['/tmp/watch/new.xg', '/tmp/watch/dup.xg']]);
});
