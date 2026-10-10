/**
 * match-sheet.spec.js
 *
 * La fiche d'un match : une ligne d'en-tête (joueurs, score final, longueur,
 * date, actions), la synthèse par joueur, puis des onglets (transcription,
 * graphes, à revoir, détails, infos, statistiques) qui occupent la hauteur
 * restante ; l'onglet choisi est mémorisé, et l'info-bulle d'un graphe reste
 * entière dans son cadre. En bas et sur le côté, dans les deux thèmes.
 * SHEET_SHOT_DIR=… enregistre une capture de chaque cas.
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
    test(`la fiche d'un match : synthèse, puis onglets transcription, graphes, détails (${layout}, ${theme})`, async ({ page }) => {
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
        await expect(panel.getByRole('tab', { name: 'Transcript' })).toHaveAttribute('aria-selected', 'true');
        await expect(panel.getByTestId('match-losses')).toBeHidden();
        await page.screenshot({ path: shot(`fiche-${layout}-${theme}`) });

        // Graphes : l'info-bulle d'une décision au bord droit reste dans le cadre du graphe.
        await panel.getByRole('tab', { name: 'Charts' }).click();
        const plot = panel.getByTestId('loss-plot-per');
        await expect(plot).toBeVisible();
        const frame = await plot.boundingBox();
        await page.mouse.move(frame.x + frame.width - 2, frame.y + 3);
        const tip = plot.getByRole('tooltip');
        await expect(tip).toBeVisible();
        const box = await tip.boundingBox();
        expect(box.x).toBeGreaterThanOrEqual(frame.x - 0.5);
        expect(box.x + box.width).toBeLessThanOrEqual(frame.x + frame.width + 0.5);
        expect(box.y).toBeGreaterThanOrEqual(frame.y - 0.5);
        expect(box.y + box.height).toBeLessThanOrEqual(frame.y + frame.height + 0.5);
        await page.screenshot({ path: shot(`fiche-graphes-${layout}-${theme}`) });

        if (layout !== 'bottom' || theme !== 'light') return;
        await panel.getByRole('tab', { name: 'Details' }).click();
        await expect(panel.getByRole('tab', { name: 'Details' })).toHaveClass(/active/);
        await expect(panel.getByRole('tab', { name: 'Charts' })).not.toHaveClass(/active/);
        await expect(panel.getByTestId('review-mwc7-0')).toContainText('7 pts');
        await page.screenshot({ path: shot('fiche-details') });

        // Le menu ⋯ porte les actions secondaires.
        await header.getByTestId('match-more').click();
        await expect(page.getByRole('menuitem', { name: 'Export .mat' })).toBeVisible();
        await expect(page.getByRole('menuitem', { name: 'Delete the match' })).toBeVisible();
        await page.keyboard.press('Escape');

        // L'onglet choisi survit à la fermeture de la fiche : un second clic la ferme, un troisième la rouvre.
        const row = panel.getByRole('row', { name: /Alice/ }).first().getByRole('cell', { name: /Alice/ });
        await row.click();
        await expect(header).toBeHidden();
        await row.click();
        await expect(panel.getByRole('tab', { name: 'Details' })).toHaveAttribute('aria-selected', 'true');
        await expect(panel.getByTestId('review-mwc7-0')).toBeVisible();
    });
}
