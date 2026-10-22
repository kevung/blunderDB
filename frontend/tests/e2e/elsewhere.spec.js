/**
 * elsewhere.spec.js
 *
 * Deux bases ouvertes. Le panneau Autres bases montre ce que l'autre sait de la position
 * (présence, analyse, autre MET, commentaire signé, match joué, voisines) et reste lisible à
 * 1366 px ; Copier vers / Récupérer depuis copient la position et Annuler reprend les copies de la
 * session, de la plus récente à la plus ancienne ; le lot se remplit au clavier (MAJ-B) et se
 * copie vers la base nommée ; x montre sa position à la place du plateau, puis revient. Le groupe
 * Envoyer du menu contextuel du plateau copie vers l'autre base ou ajoute au lot ; « Protéger des
 * copies » est mémorisé pour le fichier et revient à la session suivante.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock, libraryMockAfter, positionC, matchSample, matchGames, matchMovePositions } from './helpers/fixtures.js';
import en from '../../src/i18n/locales/en.json' with { type: 'json' };

const pathB = '/libs/club.db';
const board = {
    board: { points: Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 })), bearoff: [15, 15] },
    cube: { value: 0, owner: -1 },
    dice: [0, 0],
    score: [-1, -1],
    player_on_roll: 0,
    decision_type: 0
};

const elsewhere = {
    zobrist: 77,
    judge: { settings: { errorThresholdMP: 40, blunderThresholdMP: 80 }, metDigest: '' },
    groups: [
        {
            group: 1,
            path: '/libs/mine.db',
            positionId: 31,
            analysis: { analysisEngineVersion: 'GNU 1.07', checkerAnalysis: { moves: [{ move: '13/7 8/7' }] } },
            metName: 'Kazaross',
            otherMet: true,
            comments: [{ id: 5, positionId: 31, text: 'Le coach préfère construire.', author: 'Coach', signer: { name: 'Coach', fingerprint: 'ab:cd', verified: true } }],
            plays: [
                {
                    matchId: 3,
                    player1: 'Alice',
                    player2: 'Bruno',
                    event: 'Club',
                    matchLength: 7,
                    matchDate: '2026-05-01',
                    moveNumber: 12,
                    player: 0,
                    moveType: 'checker',
                    played: '24/18 13/7',
                    errorMP: 120,
                    verdict: 'other-met',
                    dice: [6, 6]
                }
            ],
            matchesTotal: 1,
            playsTruncated: false,
            neighbours: [{ zobrist: 88, distance: 2, position: { ...board, id: 44 } }]
        }
    ]
};

/** The calls made to method, by their arguments. */
const callsOf = (page, method) => page.evaluate((m) => window.__wailsCalls.filter((c) => c.method === m).map((c) => c.args), method);

/**
 * Opens a second base next to the first. `app` and `database` add to the mock's answers; with
 * `land: false` the wait for the new base's first position view is skipped (it holds none).
 */
async function openTwoBases(page, { app = {}, database = {}, land = true } = {}) {
    await page.setViewportSize({ width: 1366, height: 768 });
    await installWailsMock(
        page,
        openLibraryMock({
            app: {
                OpenDatabaseDialog: pathB,
                LibraryGroupHolding: 0,
                OpenLibraryGroup: { id: 2, path: pathB, active: true, readOnly: false },
                ActivateLibraryGroup: null,
                CloseLibraryGroup: null,
                ElsewhereInGroups: elsewhere,
                SetLibraryGroupLocked: null,
                TransferToGroup: { group: 1, positions: 1, matches: 0, tournaments: 0, alreadyHeld: 0, collections: 0, lessons: 0, errors: [], cancelled: false },
                CopyPositionToGroup: { group: 1, positions: 1, matches: 0, tournaments: 0, alreadyHeld: 0, collections: 0, lessons: 0, errors: [], cancelled: false, receipt: 7 },
                UndoPositionCopy: null,
                ...app
            },
            database
        })
    );
    await page.goto('/');
    await expect(page.locator('[data-testid="status-bar"]')).toBeVisible({ timeout: 8000 });
    await page.waitForFunction(() => performance.getEntriesByName('bdb:mounted').length > 0);
    const callsBefore = await page.evaluate(() => window.__wailsCalls.length);
    await page.locator('[data-testid="open-library-group"]').click();
    await expect(page.locator('[data-testid="library-group"]').nth(1)).toHaveClass(/active/);
    if (!land) return;
    // The new library lands on its position (and its Matches tab) once its list is read: the
    // position's view is the last call of that landing.
    await expect.poll(() => page.evaluate((n) => window.__wailsCalls.slice(n).some((c) => c.method === 'LoadPositionView'), callsBefore)).toBe(true);
    // ...and stays there: no position view asked for a while.
    const views = () => page.evaluate(() => window.__wailsCalls.filter((c) => c.method === 'LoadPositionView').length);
    await expect
        .poll(async () => {
            const n = await views();
            await page.waitForTimeout(300);
            return (await views()) === n;
        })
        .toBe(true);
}

test.describe('with a second base open', () => {
    test.beforeEach(async ({ page }) => {
        await openTwoBases(page, {});
    });

    test('the Elsewhere tab tells what the other library knows of the position, readable at 1366 px', async ({ page }) => {
        await page.keyboard.press('Control+Shift+A');
        const panel = page.locator('[data-testid="elsewhere-panel"]');
        await expect(panel).toBeVisible();
        const group = panel.locator('[data-testid="elsewhere-group"]');
        await expect(group).toHaveCount(1);
        await expect(group).toContainText('mine');
        await expect(group).toContainText('13/7 8/7');
        await expect(group.locator('[data-testid="elsewhere-other-met"]')).toContainText('Kazaross');
        await expect(group).toContainText('Le coach préfère construire.');
        await expect(group).toContainText('Alice');
        const overflow = await panel.evaluate((el) => el.scrollWidth - el.clientWidth);
        expect(overflow).toBeLessThanOrEqual(0);

        // Opening a result brings its group forward.
        await group.locator('[data-testid="elsewhere-open"]').click();
        await expect.poll(() => page.evaluate(() => window.__wailsCalls.some((c) => c.method === 'ActivateLibraryGroup'))).toBe(true);
    });

    test("x shows the other library's board in place of this one, then puts it back", async ({ page }) => {
        const boardEl = page.locator('#backgammon-board');
        // The focus off any panel: a panel that holds it keeps its bare letters.
        await page.evaluate(() => /** @type {HTMLElement | null} */ (document.activeElement)?.blur());
        const before = await boardEl.boundingBox();

        await page.keyboard.press('x');
        const swap = page.locator('[data-testid="elsewhere-swap"]');
        await expect(swap).toBeVisible();
        await expect(swap).toContainText('mine');
        const shown = await swap.boundingBox();
        expect(Math.abs(shown.x - before.x)).toBeLessThan(2);
        expect(Math.abs(shown.width - before.width)).toBeLessThan(2);
        expect(Math.abs(shown.height - before.height)).toBeLessThan(2);
        // The swap changes neither the group nor the view.
        await expect(page.locator('[data-testid="library-group"]').nth(1)).toHaveClass(/active/);

        await page.keyboard.press('x');
        await expect(swap).toHaveCount(0);

        // Held, it returns on release.
        await page.keyboard.down('x');
        await expect(swap).toBeVisible();
        await page.waitForTimeout(400);
        await page.keyboard.up('x');
        await expect(swap).toHaveCount(0);
    });

    test('copy to the other base and fetch from it, then undo both, newest first', async ({ page }) => {
        await page.keyboard.press('Control+Shift+A');
        const group = page.locator('[data-testid="elsewhere-group"]');
        await expect(group.locator('[data-testid="elsewhere-status"]')).toBeVisible();

        await group.locator('[data-testid="elsewhere-copy-to"]').click();
        await expect.poll(async () => (await callsOf(page, 'CopyPositionToGroup')).length).toBe(1);
        const [[src, , dst]] = await callsOf(page, 'CopyPositionToGroup');
        expect([src, dst]).toEqual([2, 1]);

        await group.locator('[data-testid="elsewhere-fetch-from"]').click();
        await expect.poll(async () => (await callsOf(page, 'CopyPositionToGroup')).length).toBe(2);
        expect((await callsOf(page, 'CopyPositionToGroup'))[1]).toEqual([1, 31, 2]);

        const undo = page.locator('[data-testid="transfer-undo"]');
        await expect(undo).toContainText('2');
        await undo.click();
        await expect.poll(async () => (await callsOf(page, 'UndoPositionCopy')).length).toBe(1);
        await expect(undo).toContainText('1');
        await undo.click();
        await expect.poll(async () => (await callsOf(page, 'UndoPositionCopy')).length).toBe(2);
        await expect(undo).toHaveCount(0);
    });

    test('the lot, filled with SHIFT-B, is copied at once into the base it names', async ({ page }) => {
        await page.keyboard.press('Control+Shift+A');
        await page.evaluate(() => /** @type {HTMLElement | null} */ (document.activeElement)?.blur());
        await page.keyboard.press('Shift+B');
        const lot = page.locator('[data-testid="transfer-lot"]');
        await expect(lot.locator('[data-testid="transfer-lot-item"]')).toHaveCount(1);

        await lot.locator('[data-testid="transfer-lot-copy"]').click();
        await expect(lot.locator('[data-testid="transfer-lot-item"]')).toHaveCount(0);
        const calls = await callsOf(page, 'TransferToGroup');
        expect(calls).toHaveLength(1);
        expect(calls[0][0]).toBe(1);
        expect(calls[0][1][0].group).toBe(2);
    });

    test('the Send group of the board copies the position to the other base, or adds it to the lot', async ({ page }) => {
        await page.locator('#backgammon-board').click({ button: 'right' });
        const menu = page.getByRole('menu', { name: en.common.contextMenu }).first();
        await menu.getByRole('menuitem', { name: en.actions.group.send }).hover();
        await page.getByRole('menuitem', { name: /Copy the position to/ }).click();
        await expect.poll(async () => (await callsOf(page, 'CopyPositionToGroup')).length).toBe(1);
        const [[src, , dst]] = await callsOf(page, 'CopyPositionToGroup');
        expect([src, dst]).toEqual([2, 1]);
        await expect(page.locator('[data-testid="transfer-undo"]')).toContainText('1');

        await page.locator('#backgammon-board').click({ button: 'right' });
        await page.getByRole('menu', { name: en.common.contextMenu }).first().getByRole('menuitem', { name: en.actions.group.send }).hover();
        await page.getByRole('menuitem', { name: 'Add to the lot' }).click();
        await page.keyboard.press('Control+Shift+A');
        await expect(page.locator('[data-testid="transfer-lot-item"]')).toHaveCount(1);
    });

    test('« Protect from copies » is remembered for the file, and holds again in the next session', async ({ page }) => {
        const lock = page.locator('[data-testid="library-group-lock"]').first();
        await lock.click();
        await expect(lock).toHaveAttribute('aria-pressed', 'true');
        expect(await callsOf(page, 'SetLibraryGroupLocked')).toContainEqual([1, true]);

        await page.reload();
        await expect(page.locator('[data-testid="status-bar"]')).toBeVisible({ timeout: 8000 });
        // The file opens again in group 1: its protection is asked of the backend once more.
        await expect.poll(async () => await callsOf(page, 'SetLibraryGroupLocked')).toContainEqual([1, true]);
    });
});

test.describe('the states of the Elsewhere panel', () => {
    test('alone, it invites to open a base', async ({ page }) => {
        await page.setViewportSize({ width: 1366, height: 768 });
        await installWailsMock(page, openLibraryMock());
        await page.goto('/');
        await expect(page.locator('[data-testid="status-bar"]')).toBeVisible({ timeout: 8000 });
        await page.keyboard.press('Control+Shift+A');
        await expect(page.locator('[data-testid="elsewhere-alone"]')).toBeVisible();
        await expect(page.locator('[data-testid="elsewhere-open-base"]')).toBeVisible();
        await expect(page.locator('[data-testid="elsewhere-group"]')).toHaveCount(0);
    });

    test('an other analysis there is kept by a copy, and the panel says so', async ({ page }) => {
        await openTwoBases(page, {
            database: { LoadPositionView: { analysis: { positionId: positionC.id, checkerAnalysis: { moves: [{ move: '24/18 13/7' }] } }, comment: '' } }
        });
        await page.keyboard.press('Control+Shift+A');
        const group = page.locator('[data-testid="elsewhere-group"]');
        await expect(group.locator('[data-testid="elsewhere-status"]')).toContainText(en.elsewhere.status.otherAnalysis);
        await expect(group.locator('[data-testid="elsewhere-kept"]')).toBeVisible();
    });

    test('the match selected in its navigator stays the selection on the Elsewhere tab', async ({ page }) => {
        await openTwoBases(page, {
            app: { ElsewhereObjectInGroups: [{ group: 1, path: '/libs/mine.db', id: 12, name: '', count: 2 }] },
            database: { GetAllMatches: [matchSample], GetGamesByMatch: matchGames, GetMatchMovePositions: matchMovePositions }
        });
        await page.getByRole('region', { name: 'Match navigator' }).getByRole('row', { name: /Alice/ }).click();
        await page.keyboard.press('Control+Shift+A');
        const object = page.locator('[data-testid="elsewhere-object"]');
        await expect(object).toBeVisible();
        await expect(object).toContainText('Alice');
        await expect(object.locator('[data-testid="elsewhere-object-status"]')).toContainText(en.elsewhere.status.present);
        expect((await callsOf(page, 'ElsewhereObjectInGroups')).at(-1)).toEqual(['match', 7]);
    });

    test('an empty base has no position to compare', async ({ page }) => {
        await openTwoBases(page, { database: libraryMockAfter([]), land: false });
        await page.keyboard.press('Control+Shift+A');
        await expect(page.locator('[data-testid="elsewhere-no-position"]')).toHaveText(en.elsewhere.noPosition);
        await expect(page.locator('[data-testid="elsewhere-group"]')).toHaveCount(0);
    });
});
