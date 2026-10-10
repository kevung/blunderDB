/**
 * stats-open-lists.spec.js — ce qu'ouvre un compte du panneau Stats
 *
 * Un compte de positions quitte le mode match avant d'ouvrir sa liste : la navigation et
 * l'analyse suivent alors cette liste, plus le match. Une ligne du Corpus ouvre les matchs
 * qu'elle résume dans le panneau Matchs, que le bouton « × » rend à la liste complète.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock, getWailsCalls } from './helpers/wailsMock.js';
import { openLibraryMock, libraryMockAfter, libraryPositions, matchSample, matchGames, matchMovePositions } from './helpers/fixtures.js';

const statusBar = (page) => page.getByTestId('status-bar');
const infoBar = (page) => page.getByTestId('match-info-bar');

const totals = { NumDecisions: 42, NumCheckerDecisions: 35, NumCubeDecisions: 7 };
const stats = { PRGlobal: 3.14, PRChecker: 2.5, PRCube: 0.64, Totals: totals, ByDecisionType: [], ByDice: [] };
const matchTwo = { ...matchSample, id: 8, player1_name: 'Carol', file_path: '/tmp/carol-bob.xg' };

test('un compte de Stats ouvert pendant un match quitte le match pour sa liste', async ({ page }) => {
    const library = [...libraryPositions, ...matchMovePositions.map((mp) => mp.position)];
    const listed = [1, 2];
    await installWailsMock(
        page,
        openLibraryMock({
            database: {
                ...libraryMockAfter(library),
                GetAllMatches: [matchSample],
                GetMatchMovePositions: matchMovePositions,
                GetGamesByMatch: matchGames,
                ComputeStats: stats,
                GetAllPlayerNames: [{ Name: 'Alice', Count: 30 }],
                GetPositionIDsByStatsSelection: listed,
                LoadPositionIDsByFilters: listed
            }
        })
    );
    await page.goto('/');
    await expect(statusBar(page)).toContainText('9 / 9');

    const panel = page.getByRole('region', { name: 'Match navigator' });
    await panel.getByRole('row', { name: /Alice/ }).click();
    await panel.getByRole('button', { name: /Review/ }).click();
    await expect(infoBar(page)).toBeVisible();

    await page.getByTestId('tab-stats').click();
    await page
        .getByRole('button', { name: /Open 42 positions/ })
        .first()
        .click();

    await expect(infoBar(page)).toBeHidden();
    await expect(statusBar(page)).toContainText('/ 2');
    await expect(statusBar(page)).not.toContainText('move ');
});

test('une ligne du Corpus ouvre ses matchs, le bouton × rend la liste', async ({ page }) => {
    await installWailsMock(
        page,
        openLibraryMock({
            database: {
                GetAllMatches: [matchSample, matchTwo],
                ComputeStats: stats,
                GetAllPlayerNames: [{ Name: 'Alice' }, { Name: 'Bob' }],
                HeadToHead: {
                    player_a: 'Alice',
                    player_b: 'Bob',
                    wins_a: 1,
                    wins_b: 0,
                    pr_a: 2,
                    pr_b: 3,
                    matches: [{ id: 7, date: '2026-01-15', match_length: 7, outcome: 'a', pr_a: 2, pr_b: 3 }]
                }
            }
        })
    );
    await page.goto('/');
    await page.getByTestId('tab-stats').click();
    await page.locator('.stats-panel [role="tab"]', { hasText: 'Corpus' }).click();
    await page.locator('#corpus-a').fill('Alice');
    await page.locator('#corpus-b').fill('Bob');
    await page.getByRole('button', { name: 'Compute' }).first().click();
    await page.getByTestId('corpus-h2h-row').click();

    const chip = page.getByTestId('match-ids-filter');
    await expect(chip).toBeVisible();
    const lastListing = async () => (await getWailsCalls(page, 'ListMatches')).at(-1).args[0];
    await expect.poll(async () => (await lastListing()).IDs).toEqual([7]);

    await chip.click();
    await expect(chip).toBeHidden();
    await expect.poll(async () => (await lastListing()).IDs).toBeUndefined();
});
