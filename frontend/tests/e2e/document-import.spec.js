/**
 * document-import.spec.js — l'assistant d'import de document, de la commande
 * `import_document` à la collection créée : défauts confirmés une fois, entrée
 * à vérifier avec sa note, repli et import.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock, getWailsCalls } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';

const summary = (confirmed) => ({
    title: 'Livre',
    entries: 2,
    sure: 1,
    toVerify: 1,
    uncertain: 0,
    complete: 0,
    defaults: { description: 'score 0-0', context: 'ctx', confirmed, direction: { description: 'to the right', confirmed } },
    unverified: confirmed ? [2] : [1, 2],
    unverifiedComplete: confirmed ? [2] : [1, 2],
    fallback: confirmed ? [2] : [],
    fallbackComplete: confirmed ? [2] : []
});

test('un document se lit, se confirme une fois et s’importe sous le repli', async ({ page }) => {
    await installWailsMock(
        page,
        openLibraryMock({
            app: { OpenDocumentDialog: '/tmp/livre.pdf' },
            database: {
                AnalyzeDocument: '{"r":1}',
                SummarizeDocumentReleve: summary(false),
                ReviewDocumentReleve: [{ rank: 2, reference: 'p. 3, Problem 4', state: 'to-verify', note: 'context not read, to check against the diagram: cube (drawn)' }],
                ConfirmDocumentReleve: '{"r":2}',
                ImportDocumentReleve: { collection: 'Livre', added: 2, repeated: 0, entries: 2, positions: 2, newPositions: 2, complete: 0, fallback: 1 }
            }
        })
    );
    await page.goto('/');
    await expect(page.getByTestId('status-bar')).toContainText('3 / 3');

    await page.keyboard.press('Space');
    await page.locator('.command-input').fill('import_document');
    await page.keyboard.press('Enter');

    await expect(page.getByTestId('document-import-review')).toContainText('cube (drawn)');
    await expect(page.getByTestId('document-import-run')).toBeDisabled();
    await expect(page.getByTestId('document-import-fallback')).toBeDisabled();

    // Le backend factice rend désormais le Relevé confirmé.
    await page.evaluate((s) => {
        window.go.database.Database.SummarizeDocumentReleve = async () => s;
    }, summary(true));
    await page.getByTestId('document-import-confirm').click();
    await expect(page.getByTestId('document-import-confirm')).toHaveCount(0);

    await page.getByTestId('document-import-fallback').check();
    await expect(page.getByTestId('document-import-run')).toBeEnabled();
    await page.getByTestId('document-import-run').click();

    await expect(page.getByTestId('document-import-done')).toContainText('Livre');
    const calls = await getWailsCalls(page, 'ImportDocumentReleve');
    expect(calls[0].args).toEqual(['{"r":2}', '', false, true, false]);
});
