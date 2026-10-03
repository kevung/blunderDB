/**
 * import-folder.spec.js — importer un dossier (CTRL-MAJ-F)
 *
 * Le dialogue de dossier et la liste des fichiers importables sont mockés ; l'import du
 * deuxième fichier attend qu'on le relâche, ce qui laisse voir la progression. Un fichier
 * déjà importé compte comme ignoré, pas comme échec, et le rapport final le dit.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock, getWailsCalls } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';

const FILES = ['/tmp/club/a.xg', '/tmp/club/b.xg', '/tmp/club/c.xg'];
const statusBar = (page) => page.getByTestId('status-bar');

test('un dossier s’importe fichier par fichier, doublon compté à part', async ({ page }) => {
    await installWailsMock(
        page,
        openLibraryMock({
            app: { OpenPositionFolderDialog: '/tmp/club', CollectImportableFiles: FILES }
        })
    );
    await page.goto('/');
    await expect(statusBar(page)).toContainText('3 / 3');

    await page.evaluate(() => {
        let release;
        window.__releaseSecond = new Promise((r) => (release = r));
        window.__release = release;
        window.go.database.Database.ImportXGMatch = async (path) => {
            if (path.endsWith('b.xg')) {
                await window.__releaseSecond;
                throw new Error('duplicate match: already been imported');
            }
            return 7;
        };
    });

    await page.keyboard.press('Control+Shift+F');

    const modal = page.locator('.modal-content, [role="dialog"]').filter({ hasText: 'Importing file' });
    await expect(modal).toContainText('Importing file 2 of 3');
    await expect(modal).toContainText('b.xg');

    await page.evaluate(() => window.__release());

    const done = page.locator('[role="dialog"], .modal-content').filter({ hasText: 'Import Completed' });
    await expect(done).toBeVisible();
    const stat = (label) => done.locator('.stat-item').filter({ hasText: label }).locator('.stat-value');
    await expect(stat('Imported')).toHaveText('2');
    await expect(stat('Duplicates skipped')).toHaveText('1');
    await expect(stat('Failed')).toHaveText('0');

    const imported = await getWailsCalls(page, 'ImportXGMatch');
    expect(imported.map((c) => c.args[0])).toEqual(FILES);
});

test('un dossier sans fichier importable le dit, sans ouvrir la progression', async ({ page }) => {
    await installWailsMock(
        page,
        openLibraryMock({
            app: { OpenPositionFolderDialog: '/tmp/vide', CollectImportableFiles: [] }
        })
    );
    await page.goto('/');
    await expect(statusBar(page)).toContainText('3 / 3');

    await page.keyboard.press('Control+Shift+F');

    await expect(page.getByTestId('status-bar-message')).toHaveText('No importable files found in folder');
    await expect(page.getByText('Importing file')).toHaveCount(0);
    expect(await getWailsCalls(page, 'ImportXGMatch')).toHaveLength(0);
});
