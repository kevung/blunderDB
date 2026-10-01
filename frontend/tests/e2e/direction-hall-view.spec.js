/**
 * direction-hall-view.spec.js
 *
 * La Salle d'une Rencontre (ADR-0056 §5) : un onglet à gauche des épreuves, une grille où
 * chaque case porte son épreuve, les propositions groupées, et les gestes de la grille d'une
 * épreuve — glisser-déposer, chiffres, cibles de 44 px — adressés à l'épreuve de la case.
 * L'échange avec une épreuve sœur passe par le service, qui écrit dans les deux journaux.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';
import { installDirectionEngine, S3_SATURDAY_20H } from './helpers/directionEngine.js';

const HALL = '[data-testid="direction-pane-hall"]';
const cell = (n) => `${HALL} [data-testid="direction-table-${n}"]`;

/** La fusion est faite en Go ; ici une Salle factice : le speed (1) sur ses tables 1 et 2, le principal (2) sur les autres tables qu'il occupe (3 à 7). */
async function installHall(page) {
    await page.addInitScript(() => {
        const db = window.go.database.Database;
        window.__hallMoves = [];
        const move = db.MoveMatchToTable;
        db.MoveMatchToTable = (id, matchId, table) => {
            window.__hallMoves.push([id, matchId, table]);
            return id === 1 ? move(id, matchId, table) : db.GetDirection(1);
        };
        db.RencontreTableGrid = async (rid) => {
            const own = await db.TableGrid(1);
            const cells = own.map((c) => {
                if (c.matchId) return { ...c, tournamentId: 1, event: 'Speed', eventIndex: 1 };
                if (c.elsewhere)
                    return {
                        table: c.table,
                        free: false,
                        matchId: `p${c.table}`,
                        a: 'x',
                        b: 'y',
                        aName: `P${c.table}a`,
                        bName: `P${c.table}b`,
                        length: 11,
                        tournamentId: 2,
                        event: 'Principal',
                        eventIndex: 0
                    };
                return { ...c, eventIndex: -1 };
            });
            const view = await db.GetDirection(1);
            return {
                rencontreId: rid,
                name: 'Festival de Lyon',
                events: [
                    { tournamentId: 2, name: 'Principal', index: 0, proposals: [], names: {} },
                    { tournamentId: 1, name: 'Speed', index: 1, proposals: view.proposals, names: Object.fromEntries(view.players.map((p) => [p.id, p.name])) }
                ],
                cells
            };
        };
    });
}

async function openRoom(page) {
    await installWailsMock(page, openLibraryMock());
    await installDirectionEngine(page, { ...S3_SATURDAY_20H, running: 2 });
    await installHall(page);
    await page.goto('/');
    await page.locator('[data-testid="tab-tournaments"]').click();
    await expect(page.locator('#tournamentPanel')).toBeVisible();
    await page.locator('#tournamentPanel tbody tr').first().click();
    await page.locator('#tournamentPanel .direction-btn').click();
    await expect(page.locator('.direction-view')).toBeVisible();
    await page.locator('[data-testid="epreuve-tab-hall"]').click();
    await expect(page.locator(cell(1))).toBeVisible();
}

async function drag(page, from, to) {
    const a = await page.locator(cell(from)).boundingBox();
    const b = await page.locator(cell(to)).boundingBox();
    await page.mouse.move(a.x + a.width / 2, a.y + a.height / 2);
    await page.mouse.down();
    await page.mouse.move(a.x + a.width / 2 + 20, a.y + a.height / 2 + 20, { steps: 3 });
    await page.mouse.move(b.x + b.width / 2, b.y + b.height / 2, { steps: 5 });
    await page.mouse.up();
}

test('la Salle : un onglet avant les épreuves, chaque case marquée de son épreuve', async ({ page }) => {
    await openRoom(page);
    const tabs = page.locator('[data-testid="epreuve-tabs"] button');
    await expect(tabs.first()).toHaveAttribute('data-testid', 'epreuve-tab-hall');
    await expect(page.locator(cell(1))).toContainText('Speed');
    await expect(page.locator(cell(3))).toContainText('Principal');
    await expect(page.locator(`${HALL} [data-testid="hall-proposals-1"]`)).toBeVisible();
    // Les volets de l'épreuve restent montés, mais cachés.
    await expect(page.locator('[data-testid="direction-pane-direction"]')).toBeHidden();
    // Une épreuve choisie quitte la Salle.
    await page.locator('[data-testid="epreuve-tab-1"]').click();
    await expect(page.locator(HALL)).toBeHidden();
    await expect(page.locator('[data-testid="direction-pane-direction"]')).toBeVisible();
});

test('glisser un match du speed sur une table du principal échange entre épreuves, après confirmation', async ({ page }) => {
    await openRoom(page);
    await drag(page, 1, 3);
    const dialog = page.locator('[aria-modal="true"]');
    await expect(dialog).toContainText('Principal');
    await expect(dialog).toContainText('Speed');
    await dialog.getByRole('button', { name: 'Swap' }).click();
    await expect.poll(() => page.evaluate(() => window.__hallMoves)).toEqual([[1, 'm1', 3]]);
});

test('glisser un match du principal vise le principal', async ({ page }) => {
    await openRoom(page);
    await drag(page, 3, 12);
    await expect.poll(() => page.evaluate(() => window.__hallMoves)).toEqual([[2, 'p3', 12]]);
});

test('un chiffre mène à la table dans la Salle seulement, et ouvre sa fiche', async ({ page }) => {
    await openRoom(page);
    await page.locator(cell(1)).focus();
    await page.keyboard.press('Digit3');
    await expect(page.locator(`${HALL} [data-testid="direction-result-card"]`)).toHaveCount(1);
    await expect(page.locator('[data-testid="direction-result-card"]')).toHaveCount(1);
    await expect(page.locator(cell(3))).toBeFocused();
});

test('les cases de la Salle gardent la cible de 44 px', async ({ page }) => {
    await openRoom(page);
    const box = await page.locator(cell(1)).boundingBox();
    expect(box.height).toBeGreaterThanOrEqual(44);
    const go = await page.locator(`${HALL} [data-testid="hall-proposals-1"] button.go`).first().boundingBox();
    expect(go.height).toBeGreaterThanOrEqual(44);
});
