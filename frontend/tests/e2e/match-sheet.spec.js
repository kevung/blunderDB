/**
 * match-sheet.spec.js
 *
 * La fiche d'un match se lit de haut en bas : une ligne d'en-tête (joueurs,
 * score final, longueur, date, actions), la synthèse par joueur, les
 * graphiques, la transcription, puis des sections repliées (détails du bilan,
 * infos, statistiques) dont l'état ouvert est mémorisé. En bas et sur le côté,
 * dans les deux thèmes. SHEET_SHOT_DIR=… enregistre une capture de chaque cas.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock, matchSample, matchGames } from './helpers/fixtures.js';
import { reviewMoves, reviewLosses, reviewSummary } from './helpers/matchReviewFixture.js';

const shot = (/** @type {string} */ name) => (process.env.SHEET_SHOT_DIR ? `${process.env.SHEET_SHOT_DIR}/${name}.png` : undefined);

for (const [layout, theme] of [
    ['bottom', 'light'],
    ['bottom', 'dark'],
    ['side', 'light']
]) {
    test(`la fiche d'un match : synthèse, graphiques, transcription, détails repliés (${layout}, ${theme})`, async ({ page }) => {
        await page.setViewportSize(layout === 'side' ? { width: 1700, height: 1000 } : { width: 1400, height: 1200 });
        await installWailsMock(
            page,
            openLibraryMock({
                database: {
                    GetAllMatches: [{ ...matchSample, tournament_name: 'Open de Lyon', round: '3', location: 'Lyon' }],
                    GetMatchMovePositions: reviewMoves,
                    GetGamesByMatch: matchGames,
                    GetMatchDecisionLosses: reviewLosses,
                    GetMatchReview: reviewSummary
                },
                config: { GetTheme: theme, GetTabPanelHeights: { '*': 750 }, GetPanelPosition: layout, GetPanelWidth: 700 }
            })
        );
        await page.goto('/');
        const panel = page.getByRole('region', { name: 'Match navigator' });
        await panel.getByRole('row', { name: /Alice/ }).getByRole('cell', { name: /Alice/ }).click();

        const header = panel.getByTestId('match-detail-header');
        await expect(header.getByTestId('header-score')).toHaveText('1–0');
        await expect(panel.getByTestId('review-mwc-0')).toHaveText('1.70 %');
        await expect(panel.getByTestId('match-losses')).toBeVisible();
        const details = panel.getByTestId('match-section-review');
        await expect(details).not.toHaveAttribute('open');
        await expect(panel.getByTestId('review-mwc7-0')).toBeHidden();
        await page.screenshot({ path: shot(`fiche-${layout}-${theme}`) });

        if (layout !== 'bottom' || theme !== 'light') return;
        await details.locator('summary').click();
        await expect(panel.getByTestId('review-mwc7-0')).toContainText('7 pts');
        await panel.getByTestId('match-section-review').scrollIntoViewIfNeeded();
        await page.screenshot({ path: shot('fiche-details-ouverts') });

        // Le menu ⋯ porte les actions secondaires.
        await header.getByTestId('match-more').click();
        await expect(page.getByRole('menuitem', { name: 'Export .mat' })).toBeVisible();
        await expect(page.getByRole('menuitem', { name: 'Delete the match' })).toBeVisible();
        await page.keyboard.press('Escape');

        // L'état ouvert survit à la fermeture de la fiche : un second clic la ferme, un troisième la rouvre.
        const row = panel.getByRole('row', { name: /Alice/ }).first().getByRole('cell', { name: /Alice/ });
        await row.click();
        await expect(header).toBeHidden();
        await row.click();
        await expect(panel.getByTestId('match-section-review')).toHaveAttribute('open');
    });
}
