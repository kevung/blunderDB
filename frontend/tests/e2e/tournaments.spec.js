/**
 * tournaments.spec.js — le panneau Tournois (CTRL-Y)
 *
 * Créer un tournoi l'enregistre et l'ajoute à la liste ; un double-clic l'ouvre sur ses
 * matchs ; la flèche de retour ramène à la liste.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock, getWailsCalls } from './helpers/wailsMock.js';
import { openLibraryMock, matchSample } from './helpers/fixtures.js';

const open = { id: 3, name: 'Open de Lyon', date: '2026-05-01', location: 'Lyon', matchCount: 1 };
const created = { id: 4, name: 'Coupe d’automne', date: '', location: 'Paris', matchCount: 0 };
const panel = (page) => page.locator('.tournament-panel');

test.beforeEach(async ({ page }) => {
    await installWailsMock(
        page,
        openLibraryMock({
            database: { GetAllTournaments: [open], GetTournamentMatches: [matchSample], GetAllMatches: [matchSample] }
        })
    );
    await page.goto('/');
    await expect(page.getByTestId('status-bar')).toContainText('3 / 3');
    await page.keyboard.press('Control+y');
    await expect(panel(page)).toBeVisible();
    await expect(panel(page).getByText('Open de Lyon')).toBeVisible();
});

test('créer un tournoi l’enregistre et l’affiche dans la liste', async ({ page }) => {
    await page.evaluate(
        (list) => {
            window.go.database.Database.CreateTournament = async () => {
                window.go.database.Database.GetAllTournaments = async () => list;
                return 4;
            };
        },
        [open, created]
    );

    await panel(page).getByTestId('panel-new').click();
    await panel(page).locator('.add-input.name').fill('Coupe d’automne');
    await panel(page).locator('.add-input.loc').fill('Paris');
    await panel(page).locator('.add-input.loc').press('Enter');

    await expect(panel(page).getByText('Coupe d’automne')).toBeVisible();
    await expect(page.getByTestId('status-bar-message')).toHaveText('Tournament "Coupe d’automne" created');
    const calls = await getWailsCalls(page, 'CreateTournament');
    expect(calls.map((c) => c.args)).toEqual([['Coupe d’automne', '', 'Paris']]);
    await expect(panel(page).locator('.add-input.name')).toHaveCount(0);
});

test('un tournoi s’ouvre sur ses matchs, la flèche ramène à la liste', async ({ page }) => {
    await panel(page).getByText('Open de Lyon').dblclick();

    await expect(panel(page).getByRole('cell', { name: 'Alice' })).toBeVisible();
    expect((await getWailsCalls(page, 'GetTournamentMatches')).map((c) => c.args[0])).toEqual([3]);

    await panel(page).getByTitle('Back to tournaments').click();
    await expect(panel(page).getByText('Open de Lyon')).toBeVisible();
    await expect(panel(page).getByRole('cell', { name: 'Alice' })).toHaveCount(0);
});
