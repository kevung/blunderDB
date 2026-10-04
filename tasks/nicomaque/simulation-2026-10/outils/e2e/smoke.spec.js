/**
 * smoke.spec.js — le harnais tient : vrai front, vrai backend, une base neuve.
 *
 * Par l'interface seulement : créer un tournoi, le diriger (préréglage suisse + tableau), ouvrir
 * sa Direction, inscrire quatre joueurs, confirmer une proposition, saisir un résultat. Puis, hors
 * interface, le backend doit avoir ce résultat dans son historique.
 */
import { test, expect } from '@playwright/test';
import { installRealBackend, shimCall, shimPageWarnings } from './realBackend.js';
import { Shim } from './shimProcess.js';
import { Meter, VIEWPORTS } from './meter.js';
import path from 'node:path';

const PLAYERS = ['Alix Morvan', 'Bastien Ruelle', 'Céline Vasseur', 'Damien Ortolan'];

test('harnais : un résultat saisi à l’écran arrive dans la vraie base', async ({ page }) => {
    const shim = await new Shim({ name: 'smoke', port: 8941 }).start();
    try {
        await page.setViewportSize(VIEWPORTS.hd);
        await installRealBackend(page, { shimUrl: shim.url, dbPath: shim.dbPath, outDir: shim.outDir });
        const m = new Meter(page, { persona: 'P1', tournament: 'smoke', viewport: VIEWPORTS.hd, file: path.join(shim.dir, 'meter.jsonl'), feedbackMs: 300 });
        await page.goto('/');

        await m.act({ action: 'ouvrir Tournois', mustPass: true }, async (g) => {
            await g.click('[data-testid="tab-tournaments"]');
        });
        await m.act({ action: 'créer le tournoi', mustPass: true }, async (g) => {
            await g.click('#tournamentPanel [data-testid="panel-new"]');
            await g.type('Club du jeudi');
            await g.press('Enter');
        });
        await expect(page.locator('#tournamentPanel tbody tr')).toHaveCount(1);
        await m.act({ action: 'diriger', mustPass: true }, async (g) => {
            // Un clic ne fait que surligner : le tournoi s'ouvre au double-clic.
            await g.dblclick(page.locator('#tournamentPanel tbody tr').first());
            await g.click('[data-testid="tournament-direct"]');
        });
        // « Diriger » crée la Direction et l'ouvre : la vue est chargée à la demande (import
        // dynamique), il faut l'attendre plutôt que cliquer « Ouvrir », qui la refermerait.
        await expect(page.locator('[data-testid="direction-tab-players"]')).toBeVisible({ timeout: 20000 });

        await m.act({ action: 'onglet Joueurs', mustPass: true }, async (g) => {
            await g.click('[data-testid="direction-tab-players"]');
        });
        for (const name of PLAYERS) {
            await m.act({ action: `inscrire ${name}`, mustPass: true }, async (g) => {
                await g.click('[data-testid="direction-player-entry"] input[type="text"] >> nth=0');
                await g.type(name);
                await g.press('Enter');
            });
        }
        const parts = await shimCall(shim.url, 'Participants', await tournamentId(shim));
        expect(parts.length).toBe(4);

        await m.act({ action: 'onglet Direction', mustPass: true }, async (g) => {
            await g.click('[data-testid="direction-tab-direction"]');
        });
        // Le préréglage peut commencer par un tirage (« Tirage : Tableau de 8 places ») : on lance
        // la tête de file jusqu'à ce qu'un match occupe une table.
        const queue = page.locator('.proposals .queue li:not(.empty)');
        const busy = page.locator('[role="gridcell"].busy');
        for (let i = 0; i < 6 && !(await busy.count()); i++) {
            await expect(queue.first()).toBeVisible();
            await m.act({ action: `lancer la tête de file (${(await queue.first().innerText()).split('\n')[0]})`, mustPass: true }, async (g) => {
                await g.click(queue.first().locator('.go'));
            });
            await page.waitForTimeout(300);
        }
        await expect(busy.first()).toBeVisible();
        await m.act({ action: 'saisir le résultat (A gagne)', mustPass: true }, async (g) => {
            await g.click(busy.first());
            await g.click('[data-testid="direction-result-winner-a"]');
        });
        await expect(page.locator('[data-testid="direction-last"]')).toBeVisible();
        const hist = await shimCall(shim.url, 'History', await tournamentId(shim), '', '');
        expect(hist.map((e) => e.kind).join(' '), 'événements du journal').toMatch(/result/i);
        expect(await shimPageWarnings(shim.url)).toEqual([]);
        expect(m.records.every((r) => r.ok)).toBe(true);
    } finally {
        await shim.dispose({ keep: !!process.env.SIM_KEEP });
    }
});

async function tournamentId(shim) {
    const list = await shimCall(shim.url, 'ListDirections');
    return list[0].tournamentId;
}
