/**
 * transcription-theatre.spec.js — le mode théâtre de la Transcription, dans un vrai
 * navigateur : la vidéo prend toute la fenêtre, le mini-plateau flotte au coin et se
 * déplace, le clavier de saisie reste actif, F11 et Échap en sortent.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';
import { installTranscriptionEngine, sentKinds, resetGestures } from './helpers/transcriptionDraft.js';

test.beforeEach(async ({ page }) => {
    await installWailsMock(page, openLibraryMock({ config: { GetPanelPosition: 'side', GetPanelWidth: 420 } }));
    await installTranscriptionEngine(page, { video: '/videos/match.mp4' });
    await page.goto('/');
    await page.locator('[data-testid="tab-transcription"]').click();
    await page.locator('#transcriptionPanel tbody tr').first().click();
    await expect(page.locator('#transcriptionPanel .draft-bar')).toBeVisible();
});

const theatre = (/** @type {import('@playwright/test').Page} */ page) => page.locator('[data-testid="transcription-theatre"]');

test('F11 ouvre le théâtre : la vidéo couvre la fenêtre, le mini-plateau est au coin', async ({ page }) => {
    await page.locator('#transcriptionPanel').focus();
    await page.keyboard.press('F11');
    await expect(theatre(page)).toBeVisible();
    const view = page.viewportSize() ?? { width: 0, height: 0 };
    const video = await page.locator('[data-testid="video-dock"]').boundingBox();
    // Under the theatre's bar, the video takes the rest of the window.
    expect(video?.width).toBe(view.width);
    expect((video?.y ?? 0) + (video?.height ?? 0)).toBe(view.height);
    expect(video?.y ?? 0).toBeLessThanOrEqual(40);
    const board = await page.locator('[data-testid="theatre-board"]').boundingBox();
    expect((board?.x ?? 0) + (board?.width ?? 0)).toBeGreaterThan(view.width - 40);
    expect((board?.y ?? 0) + (board?.height ?? 0)).toBeGreaterThan(view.height - 100);
    await page.keyboard.press('F11');
    await expect(theatre(page)).toHaveCount(0);
    // La vidéo revient où elle était : à côté du plateau, la place par défaut.
    await expect(page.locator('[data-testid="video-dock"]')).toHaveAttribute('data-placement', 'board');
});

test('la saisie continue dans le théâtre, et Échap en sort', async ({ page }) => {
    await page.locator('[data-testid="video-theatre"]').click();
    await expect(theatre(page)).toBeVisible();
    await resetGestures(page);
    await page.keyboard.press('3');
    await expect.poll(() => sentKinds(page)).toContain('enter_die');
    await page.keyboard.press('Escape');
    await expect(theatre(page)).toHaveCount(0);
});

test('le mini-plateau se déplace à la souris et garde sa place', async ({ page }) => {
    await page.locator('#transcriptionPanel').focus();
    await page.keyboard.press('F11');
    const board = page.locator('[data-testid="theatre-board"]');
    const before = await board.boundingBox();
    if (!before) throw new Error('mini-plateau absent');
    await page.mouse.move(before.x + 40, before.y + 10);
    await page.mouse.down();
    await page.mouse.move(80, 60, { steps: 5 });
    await page.mouse.up();
    const after = await board.boundingBox();
    expect(after?.x).toBeLessThan(before.x - 100);
    expect(after?.y).toBeLessThan(before.y - 100);
    // Le glisser ne prend pas le focus : le panneau garde ses touches.
    await expect(page.locator('#transcriptionPanel')).toBeFocused();
    await page.keyboard.press('Escape');
    await page.keyboard.press('F11');
    const again = await page.locator('[data-testid="theatre-board"]').boundingBox();
    expect(Math.abs((again?.x ?? 0) - (after?.x ?? 0))).toBeLessThan(2);
});

for (const size of ['s', 'm', 'l']) {
    test(`le plateau entier, numéros compris, tient dans la boîte du mini-plateau (taille ${size})`, async ({ page }) => {
        await page.locator('#transcriptionPanel').focus();
        await page.keyboard.press('F11');
        const board = page.locator('[data-testid="theatre-board"]');
        while ((await board.getAttribute('data-size')) !== size) await page.locator('[data-testid="theatre-board-size"]').click();
        const m = await board.evaluate((el) => {
            const box = el.getBoundingClientRect();
            const svg = /** @type {SVGSVGElement} */ (el.querySelector('.theatre-board-svg svg'));
            const holder = /** @type {HTMLElement} */ (el.querySelector('.theatre-board-svg')).getBoundingClientRect();
            const bottom = Math.max(...[...svg.querySelectorAll('text')].map((t) => t.getBoundingClientRect().bottom));
            const rendered = svg.getBoundingClientRect();
            return { boxBottom: box.bottom, holderBottom: holder.bottom, svgBottom: rendered.bottom, textBottom: bottom, clientH: el.clientHeight, scrollH: el.scrollHeight };
        });
        expect(m.textBottom).toBeLessThanOrEqual(m.boxBottom - 1 + 0.5);
        expect(m.svgBottom).toBeLessThanOrEqual(m.boxBottom - 1 + 0.5);
        expect(m.scrollH).toBeLessThanOrEqual(m.clientH);
    });
}

test('le lecteur n’est jamais remonté : même élément avant, pendant et après le théâtre, par-dessus la fenêtre', async ({ page }) => {
    const dock = page.locator('[data-testid="video-dock"]');
    await dock.evaluate((el) => {
        /** @type {any} */ (el).__mark = 'same';
        /** @type {any} */ (el.querySelector('[data-testid="video-pane"]')).__mark = 'same';
    });
    await page.locator('#transcriptionPanel').focus();
    await page.keyboard.press('F11');
    await expect(theatre(page)).toBeVisible();
    const during = await dock.evaluate((el) => {
        const r = el.getBoundingClientRect();
        const hit = document.elementFromPoint(r.x + r.width / 2, r.y + r.height / 2);
        return { mark: /** @type {any} */ (el).__mark, player: /** @type {any} */ (el.querySelector('[data-testid="video-pane"]')).__mark, onTop: el.contains(hit) };
    });
    expect(during).toEqual({ mark: 'same', player: 'same', onTop: true });
    await page.keyboard.press('Escape');
    await expect(theatre(page)).toHaveCount(0);
    const after = await dock.evaluate((el) => ({ mark: /** @type {any} */ (el).__mark, player: /** @type {any} */ (el.querySelector('[data-testid="video-pane"]')).__mark }));
    expect(after).toEqual({ mark: 'same', player: 'same' });
});

test('les boutons du dock sont dans une barre au-dessus de l’image, hors de la vidéo', async ({ page }) => {
    const bar = await page.locator('[data-testid="video-dock"] .video-dock-bar').boundingBox();
    const pane = await page.locator('[data-testid="video-pane"]').boundingBox();
    expect(bar).not.toBeNull();
    expect((bar?.y ?? 0) + (bar?.height ?? 0)).toBeLessThanOrEqual((pane?.y ?? 0) + 0.5);
    for (const id of ['video-theatre', 'video-placement']) {
        const b = await page.locator(`[data-testid="${id}"]`).boundingBox();
        expect((b?.y ?? 0) + (b?.height ?? 0)).toBeLessThanOrEqual((pane?.y ?? 0) + 0.5);
    }
});

test('le plateau du théâtre est visible et non vide', async ({ page }) => {
    await page.locator('#transcriptionPanel').focus();
    await page.keyboard.press('F11');
    const board = page.locator('[data-testid="theatre-board"]');
    await expect(board).toBeVisible();
    const m = await board.evaluate((el) => {
        const svg = el.querySelector('.theatre-board-svg svg');
        const r = svg ? svg.getBoundingClientRect() : null;
        const hit = r ? document.elementFromPoint(r.x + r.width / 2, r.y + r.height / 2) : null;
        return {
            w: r?.width ?? 0,
            h: r?.height ?? 0,
            labels: svg ? svg.querySelectorAll('text').length : 0,
            shapes: svg ? svg.querySelectorAll('path, circle, rect').length : 0,
            onTop: !!hit && el.contains(hit)
        };
    });
    expect(m.w).toBeGreaterThan(200);
    expect(m.h).toBeGreaterThan(150);
    expect(m.labels).toBeGreaterThanOrEqual(24);
    expect(m.shapes).toBeGreaterThan(30);
    expect(m.onTop).toBe(true);
});

test('sous une échelle d’interface de 150 %, le mini-plateau reste dans la fenêtre, à sa place comme glissé', async ({ page }) => {
    // Le théâtre est sous le `zoom` de l'interface : une place retenue en fraction de la fenêtre
    // doit tomber dans la fenêtre, et le glisser suivre la souris.
    await page.evaluate(() => {
        localStorage.setItem('blunderdb.theatre.board', JSON.stringify({ x: 0.75, y: 0.55, size: 'm', hidden: false }));
        document.documentElement.style.setProperty('--ui-scale', '1.5');
    });
    await page.locator('#transcriptionPanel').focus();
    await page.keyboard.press('F11');
    const view = page.viewportSize() ?? { width: 0, height: 0 };
    const board = page.locator('[data-testid="theatre-board"]');
    const inside = async () => {
        const b = await board.boundingBox();
        if (!b) throw new Error('mini-plateau absent');
        expect(b.x).toBeGreaterThanOrEqual(0);
        expect(b.y).toBeGreaterThanOrEqual(0);
        expect(b.x + b.width).toBeLessThanOrEqual(view.width + 1);
        expect(b.y + b.height).toBeLessThanOrEqual(view.height + 1);
        return b;
    };
    const before = await inside();
    expect(before.x).toBeGreaterThan(view.width / 3);
    // Saisi par sa tête, il suit la souris d'autant.
    await page.mouse.move(before.x + 40, before.y + 10);
    await page.mouse.down();
    await page.mouse.move(before.x - 160, before.y - 90, { steps: 5 });
    await page.mouse.up();
    const after = await inside();
    expect(Math.abs(after.x - (before.x - 200))).toBeLessThan(3);
    expect(Math.abs(after.y - (before.y - 100))).toBeLessThan(3);
});
