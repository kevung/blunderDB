/**
 * Les flèches du plateau en transcription sont celles de la décision transcrite. Une décision
 * qu'aucun coup légal ne joue (« 7/4 6/5 », le 7 est vide) se dessine depuis la case écrite, le
 * pas fautif en rouge pointillé ; le meilleur coup du moteur ne prend jamais sa place, et une
 * ligne de l'analyse ne se dessine, dans sa propre couleur, que survolée.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';
import { installTranscriptionEngine, resetGestures, sentGestures } from './helpers/transcriptionDraft.js';

const ILLEGAL = 'rgba(214, 0, 0, 0.95)';
const ENGINE = 'rgba(52, 120, 246, 0.85)';
const SHOTS = process.env.BLUNDERDB_E2E_SHOTS || '';

/** @param {import('@playwright/test').Page} page @param {string} color */
const strokes = (page, color) => page.locator(`#backgammon-board svg [stroke="${color}"]`);

test('une décision illégale garde ses flèches ; le moteur ne se dessine que désigné', async ({ page }) => {
    await installWailsMock(page, openLibraryMock({ config: { GetPanelPosition: 'bottom' } }));
    await installTranscriptionEngine(page, { notations: { 4: '7/4 6/5' } });
    await page.goto('/');
    await page.locator('[data-testid="tab-transcription"]').click();
    await page.locator('#transcriptionPanel tbody tr').first().click();
    await expect(page.locator('#transcriptionPanel .draft-bar')).toBeVisible();

    await page.locator('#transcriptionPanel .cell[data-index="4"]').click();
    await expect(page.locator('#transcriptionPanel .cell.cursor')).toHaveAttribute('data-index', '4');
    await expect(page.locator('[data-testid="transcription-candidates"] tr[data-move]').first()).toBeVisible();

    // Le pas 7/4 part d'une case vide : pointillé rouge, case de départ cerclée (flèche + tête + cercle).
    await expect(strokes(page, ILLEGAL)).toHaveCount(3);
    await expect(page.locator(`#backgammon-board svg [stroke="${ILLEGAL}"][stroke-dasharray]`)).toHaveCount(1);
    await expect(strokes(page, ENGINE)).toHaveCount(0);
    // Aucune ligne de l'analyse n'est choisie à la place du coup écrit.
    await expect(page.locator('[data-testid="transcription-candidates"] tr.selected')).toHaveCount(0);
    if (SHOTS) await page.locator('#backgammon-board').screenshot({ path: `${SHOTS}/apres-ecrit.png` });

    const best = page.locator('[data-testid="transcription-candidates"] tr[data-move="8/5 6/5"]');
    await best.hover();
    await expect(strokes(page, ENGINE)).not.toHaveCount(0);
    await expect(strokes(page, ILLEGAL)).toHaveCount(0);
    if (SHOTS) await page.locator('#backgammon-board').screenshot({ path: `${SHOTS}/apres-survol.png` });

    await page.locator('#transcriptionPanel .draft-bar').hover();
    await expect(strokes(page, ILLEGAL)).toHaveCount(3);
    await expect(strokes(page, ENGINE)).toHaveCount(0);

    // Choosing the first line of the analysis writes it, like any other line.
    await resetGestures(page);
    await page.keyboard.press('k');
    await page.keyboard.press('Enter');
    await expect.poll(async () => (await sentGestures(page)).map((g) => (g.Kind === 'select_candidate' ? `select ${g.Candidate}` : g.Kind))).toEqual(['select 0', 'validate']);
});
