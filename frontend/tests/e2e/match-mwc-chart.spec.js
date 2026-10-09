/**
 * match-mwc-chart.spec.js
 *
 * Fiche d'un match analysé : le graphique de la perte MWC par décision et son cumul, la
 * colonne MWC de la transcription, et le saut d'un clic sur une barre vers la ligne de la
 * transcription (la revue s'ouvre sur ce coup), dans les deux thèmes.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock, matchSample, matchGames, matchMovePositions } from './helpers/fixtures.js';

const moves = matchMovePositions.map((mp, i) => ({ ...mp, move_id: 100 + i, player_on_roll: i % 2 }));
const losses = moves.map((mp, i) => ({
    move_id: mp.move_id,
    game_number: mp.game_number,
    move_number: mp.move_number,
    player: i % 2,
    decision_type: 'checker',
    mwc_loss: i === 2 ? null : [0.012, 0, null, 0.034, 0.005, 0.021][i]
}));

for (const theme of ['light', 'dark']) {
    test(`le graphique MWC saute à la décision cliquée (${theme})`, async ({ page }) => {
        await installWailsMock(
            page,
            openLibraryMock({
                database: { GetAllMatches: [matchSample], GetMatchMovePositions: moves, GetGamesByMatch: matchGames, GetMatchDecisionLosses: losses },
                config: { GetTheme: theme }
            })
        );
        await page.goto('/');
        const panel = page.getByRole('region', { name: 'Match navigator' });
        await panel.getByRole('row', { name: /Alice/ }).getByRole('cell', { name: /Alice/ }).click();

        const chart = panel.getByTestId('match-losses');
        await expect(chart).toBeVisible();
        await expect(panel.getByTestId('loss-total-0')).toHaveText('1.70 %');
        await expect(panel.getByTestId('loss-total-1')).toHaveText('5.50 %');
        await expect(panel.getByTestId('move-loss').first()).toHaveText('3.40 %');
        await page.screenshot({ path: process.env.MWC_SHOT_DIR ? `${process.env.MWC_SHOT_DIR}/mwc-${theme}.png` : undefined });

        // Une barre du graphique : le pointeur survole la quatrième décision de six.
        const plot = chart.getByTestId('loss-plot-per');
        const box = await plot.boundingBox();
        await plot.hover({ position: { x: box.width * (3.5 / 6), y: box.height / 2 } });
        await expect(chart.getByRole('tooltip').first()).toContainText('4');
        await plot.click({ position: { x: box.width * (3.5 / 6), y: box.height / 2 } });
        await expect(page.getByTestId('match-info-bar')).toBeVisible();
        await expect(page.getByTestId('status-bar')).toContainText('move 4/6');
    });
}
