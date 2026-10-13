/**
 * Saisie et correction d'une décision au clavier (ADR-0091). En Saisie, la ligne 0 de
 * l'analyse est présélectionnée et dessinée : Entrée l'enregistre, et les dés du jet suivant
 * l'enregistrent aussi avant d'ouvrir la décision suivante. `e` ouvre le champ « dés: coup » :
 * une seule validation corrige jet et coup, des dés illisibles laissent le champ ouvert.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';
import { installTranscriptionEngine, resetGestures, sentGestures } from './helpers/transcriptionDraft.js';

const ENGINE = 'rgba(52, 120, 246, 0.85)';

/** @param {any} g */
const label = (g) => (g.Kind === 'enter_die' ? `die ${g.Die}` : g.Kind === 'select_candidate' ? `select ${g.Candidate}` : g.Kind);

/** @param {import('@playwright/test').Page} page */
async function openDraft(page) {
    await installWailsMock(page, openLibraryMock({ config: { GetPanelPosition: 'bottom' } }));
    await installTranscriptionEngine(page);
    await page.goto('/');
    await page.locator('[data-testid="tab-transcription"]').click();
    await page.locator('#transcriptionPanel tbody tr').first().click();
    await expect(page.locator('#transcriptionPanel .draft-bar')).toBeVisible();
    await expect(page.locator('[data-testid="transcription-mode"]')).toHaveAttribute('data-mode', 'entry');
}

test('Saisie : la ligne 0 est proposée et dessinée ; Entrée ou le jet suivant l’enregistre', async ({ page }) => {
    await openDraft(page);
    const candidates = page.locator('[data-testid="transcription-candidates"] tr[data-move]');

    await page.keyboard.press('5');
    await page.keyboard.press('2');
    await expect(candidates.first()).toBeVisible();
    await expect(page.locator('[data-testid="transcription-candidates"] tr.selected')).toHaveAttribute('data-move', '8/5 6/5');
    await expect(page.locator(`#backgammon-board svg [stroke="${ENGINE}"]`)).not.toHaveCount(0);

    // Le jet suivant valide la ligne présélectionnée, sans Entrée, puis ouvre la décision suivante.
    await resetGestures(page);
    await page.keyboard.press('3');
    await page.keyboard.press('1');
    await expect.poll(async () => (await sentGestures(page)).map(label).slice(0, 3)).toEqual(['validate', 'die 3', 'die 1']);
    await expect(page.locator('[data-testid="transcription-candidates"] tr.selected')).toHaveAttribute('data-move', '8/5 6/5');

    await resetGestures(page);
    await page.keyboard.press('Enter');
    await expect.poll(async () => (await sentGestures(page)).map(label)).toEqual(['validate']);
});

test('e : le champ porte « dés: coup » ; une validation corrige les deux', async ({ page }) => {
    await openDraft(page);
    await page.locator('#transcriptionPanel .cell[data-index="4"]').click();
    await expect(page.locator('[data-testid="transcription-mode"]')).toHaveAttribute('data-mode', 'review');

    await page.keyboard.press('e');
    const field = page.locator('#transcriptionPanel input.move-field');
    await expect(field).toHaveValue('31: 8/5 24/23');

    // Des dés illisibles : signalés, le champ reste ouvert, rien n'est envoyé.
    await resetGestures(page);
    await field.fill('72: 24/18 13/11');
    await field.press('Enter');
    await expect(field).toBeVisible();
    expect(await sentGestures(page)).toEqual([]);

    await field.fill('62: 24/18 13/11');
    await field.press('Enter');
    await expect(field).toHaveCount(0);
    await expect.poll(async () => (await sentGestures(page)).map(label)).toEqual(['die 6', 'die 2', 'enter_play', 'validate']);
    // La correction validée mène à la décision suivante (ADR-0091).
    await expect(page.locator('#transcriptionPanel .cell.cursor')).toHaveAttribute('data-index', '5');
});
