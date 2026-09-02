/**
 * eval-after-search-restores-board.spec.js — #201 (D.1)
 *
 * Bibliothèque → onglet Recherche (damier vierge) → onglet Eval → retour à
 * l'analyse : la position étudiée doit être de retour sur le damier. App.svelte
 * enchaîne exitEditMode() sans await puis enterEPCMode() ; le damier vierge
 * de la recherche est un clone sous l'id de la position, et son redessin est
 * asynchrone : la photo prise à l'entrée d'Eval était le damier vierge, que
 * la sortie remettait à l'écran — définitivement quand la position étudiée
 * était la première de la liste (aucun changement d'index ne redessinait).
 *
 * Le plateau décrit la position affichée dans son aria-label (pips des deux
 * joueurs, joueur au trait) : c'est ce que la spec compare.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';

const statusBar = (page) => page.getByTestId('status-bar');
const tab = (page, id) => page.getByTestId(`tab-${id}`);
const board = (page) => page.locator('#backgammon-board');

async function clickTab(page, id) {
    await tab(page, id).click();
    await expect(tab(page, id)).toHaveClass(/active/);
}

test.beforeEach(async ({ page }) => {
    await installWailsMock(page, openLibraryMock());
    await page.goto('/');
    await expect(statusBar(page)).toContainText('3 / 3');
});

test('revenir d’Eval après la recherche remet la position étudiée sur le damier', async ({ page }) => {
    // Se placer sur la première position (depuis l'onglet Analyse : l'onglet
    // Matchs, ouvert au chargement, garde les touches nues pour sa liste).
    await clickTab(page, 'analysis');
    await page.keyboard.press('k');
    await page.keyboard.press('k');
    await expect(statusBar(page)).toContainText('1 / 3');
    await expect(board(page)).toHaveAttribute('aria-label', /pip count [1-9]/);
    const studied = await board(page).getAttribute('aria-label');

    // Recherche : damier vierge.
    await page.keyboard.press('Control+f');
    await expect(tab(page, 'search')).toHaveClass(/active/);
    await expect(board(page)).not.toHaveAttribute('aria-label', studied);

    // Eval : bearoff par défaut.
    await clickTab(page, 'epc');
    await expect(board(page)).not.toHaveAttribute('aria-label', studied);

    // Retour : la position étudiée, pas le damier vierge de la recherche.
    await clickTab(page, 'analysis');
    await expect(board(page)).toHaveAttribute('aria-label', studied);
    await expect(statusBar(page)).toContainText('1 / 3');
});
