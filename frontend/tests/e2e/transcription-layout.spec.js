/**
 * transcription-layout.spec.js — le contrat de disposition d'ADR-0048, en
 * pixels : le défaut était géométrique (une chaîne de hauteur cassée faisait
 * défiler `.tab-content`), qu'un décompte de nœuds ne voit pas.
 *
 * 1. `.tab-content` ne défile pas.
 * 2. Cinq lignes de candidats sont ENTIÈREMENT visibles dans la liste
 *    (`boundingBox`, jamais `count()`).
 * 3. Rien ne s'intercale entre les cases du jet et la première ligne : la seule
 *    qui mesure la règle, sans quoi on tient 1 et 2 en rétrécissant le
 *    triangle. Un écart négatif est admis (en dock bas, la palette est à côté).
 *
 * Deux tailles fixes, dock bas plancher (280 px) et dock latéral par défaut
 * (420 px) : pas de balayage, pour qu'un rouge désigne un point de rupture.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';
import { installTranscriptionEngine } from './helpers/transcriptionDraft.js';

/** L'écart toléré, en px, entre le bas des dés et le haut de la première ligne. */
const NO_INTERPOSITION = 40;

/** Ouvre l'onglet et le brouillon, dans le régime de dock demandé. */
async function openDraft(page, config = {}) {
    await installWailsMock(page, openLibraryMock({ config }));
    await installTranscriptionEngine(page);
    await page.goto('/');
    await page.locator('[data-testid="tab-transcription"]').click();
    await page.locator('#transcriptionPanel tbody tr').first().click();
    await expect(page.locator('#transcriptionPanel .draft-bar')).toBeVisible();
    // Le jet est saisi : c'est le seul état où la liste des candidats existe, et
    // c'est celui où le panneau passe 250 tours sur 250.
    await page.keyboard.press('Digit3');
    await page.keyboard.press('Digit1');
    await expect(page.locator('[data-testid="transcription-candidates"] tbody tr').first()).toBeVisible();
}

/** De combien `.tab-content` peut défiler : 0 est le contrat. */
function overflow(page) {
    return page.locator('.tab-content').evaluate((el) => el.scrollHeight - el.clientHeight);
}

/** Le nombre de lignes de candidats entièrement dans le rectangle visible. */
function fullyVisibleRows(page) {
    return page.locator('[data-testid="transcription-candidates"]').evaluate((box) => {
        const view = box.getBoundingClientRect();
        // L'en-tête est collant : une ligne cachée dessous n'est pas lisible.
        const head = box.querySelector('thead');
        const top = head ? Math.max(view.top, head.getBoundingClientRect().bottom) : view.top;
        return [...box.querySelectorAll('tbody tr')].filter((tr) => {
            const r = tr.getBoundingClientRect();
            return r.height > 0 && r.top >= top - 0.5 && r.bottom <= view.bottom + 0.5;
        }).length;
    });
}

/** L'écart vertical entre le bas des cases du jet et le haut de la 1re ligne. */
async function interposition(page) {
    const dice = await page.locator('[data-testid="transcription-dice"]').boundingBox();
    const row = await page.locator('[data-testid="transcription-candidates"] tbody tr').first().boundingBox();
    return row.y - (dice.y + dice.height);
}

test.describe('dock bas, à la hauteur plancher de l’onglet', () => {
    test.beforeEach(async ({ page }) => {
        await openDraft(page, { GetPanelPosition: 'bottom', GetTabPanelHeights: { '*': 280 } });
    });

    test('le panneau ne défile pas', async ({ page }) => {
        expect(await overflow(page)).toBe(0);
    });

    test('cinq lignes de candidats sont entièrement visibles', async ({ page }) => {
        expect(await fullyVisibleRows(page)).toBeGreaterThanOrEqual(5);
    });

    test('rien ne s’intercale entre les dés et la première ligne', async ({ page }) => {
        expect(await interposition(page)).toBeLessThanOrEqual(NO_INTERPOSITION);
    });

    test('la palette est À CÔTÉ de la liste, pas dessus', async ({ page }) => {
        const palette = await page.locator('[data-testid="transcription-palette"]').boundingBox();
        const list = await page.locator('[data-testid="transcription-candidates"]').boundingBox();
        expect(palette.x + palette.width).toBeLessThanOrEqual(list.x + 1);
    });
});

test.describe('dock latéral, à sa largeur par défaut', () => {
    test.beforeEach(async ({ page }) => {
        await openDraft(page, { GetPanelPosition: 'side', GetPanelWidth: 420 });
    });

    test('le panneau ne défile pas', async ({ page }) => {
        expect(await overflow(page)).toBe(0);
    });

    test('cinq lignes de candidats sont entièrement visibles', async ({ page }) => {
        expect(await fullyVisibleRows(page)).toBeGreaterThanOrEqual(5);
    });

    test('rien ne s’intercale entre les dés et la première ligne', async ({ page }) => {
        expect(await interposition(page)).toBeLessThanOrEqual(NO_INTERPOSITION);
    });

    test('la palette est SOUS la liste quand la boîte est étroite', async ({ page }) => {
        const palette = await page.locator('[data-testid="transcription-palette"]').boundingBox();
        const list = await page.locator('[data-testid="transcription-candidates"]').boundingBox();
        expect(palette.y).toBeGreaterThanOrEqual(list.y + list.height - 1);
    });

    test('le Transcript est SOUS la palette : la place n’est pas là pour l’apparier', async ({ page }) => {
        const palette = await page.locator('[data-testid="transcription-palette"]').boundingBox();
        const transcript = await page.locator('.transcript-col').boundingBox();
        expect(transcript.y).toBeGreaterThanOrEqual(palette.y + palette.height - 1);
    });
});

/**
 * Le troisième régime : un dock latéral élargi, où le Transcript occupe le
 * blanc à droite du triangle. Point de rupture 500 px (triangle 211 +
 * Transcript 250 = 485), vérifié à 520 : une seconde colonne rognée ne serait
 * plus un Transcript (ADR-0048 décision 6).
 */
test.describe('dock latéral élargi, au-delà du second point de rupture', () => {
    test.beforeEach(async ({ page }) => {
        await openDraft(page, { GetPanelPosition: 'side', GetPanelWidth: 520 });
    });

    test('le panneau ne défile pas', async ({ page }) => {
        expect(await overflow(page)).toBe(0);
    });

    test('cinq lignes de candidats sont entièrement visibles', async ({ page }) => {
        expect(await fullyVisibleRows(page)).toBeGreaterThanOrEqual(5);
    });

    test('le Transcript est À CÔTÉ du triangle, pas dessous', async ({ page }) => {
        const palette = await page.locator('[data-testid="transcription-palette"]').boundingBox();
        const transcript = await page.locator('.transcript-col').boundingBox();
        expect(transcript.x).toBeGreaterThanOrEqual(palette.x + palette.width - 1);
        // Les deux partagent la rangée : ils commencent à la même hauteur.
        expect(Math.abs(transcript.y - palette.y)).toBeLessThanOrEqual(1);
    });

    test('les deux colonnes du Transcript tiennent sans être rognées', async ({ page }) => {
        // La table est ce qui doit tenir : rognée, c'est la colonne du joueur 2
        // qui disparaît, et le Transcript n'a plus qu'un camp.
        const clipped = await page.locator('.transcript-col').evaluate((col) => {
            const table = col.querySelector('table');
            return table ? table.scrollWidth - col.clientWidth : 0;
        });
        expect(clipped).toBeLessThanOrEqual(0);
    });
});

/**
 * Le formulaire d'en-tête ouvert ne chasse pas la liste : il défile dans sa
 * propre boîte, et le panneau ne déborde pas.
 */
test.describe('dock latéral, en-tête du brouillon ouvert', () => {
    test.beforeEach(async ({ page }) => {
        await openDraft(page, { GetPanelPosition: 'side', GetPanelWidth: 420 });
        await page.locator('#transcriptionPanel .draft-bar').getByRole('button', { name: 'Metadata' }).click();
        await expect(page.locator('[data-testid="transcription-metadata"]')).toBeVisible();
    });

    test('le panneau ne défile pas', async ({ page }) => {
        expect(await overflow(page)).toBe(0);
    });

    test('la liste des candidats reste visible, au moins deux lignes entières', async ({ page }) => {
        expect(await fullyVisibleRows(page)).toBeGreaterThanOrEqual(2);
    });
});

/**
 * Vidéo attachée, à la hauteur la plus grande que la poignée accorde : la
 * liste garde son plancher, c'est la vidéo qui cède.
 */
for (const [name, config, fits] of [
    ['dock latéral étroit', { GetPanelPosition: 'side', GetPanelWidth: 420 }, true],
    ['dock latéral élargi', { GetPanelPosition: 'side', GetPanelWidth: 520 }, true],
    // À 280 px, barre + vidéo (5 rem) + liste (7 rem) dépassent la hauteur : le
    // panneau défile alors par construction, seuls les planchers sont dus.
    ['dock bas, plancher', { GetPanelPosition: 'bottom', GetTabPanelHeights: { '*': 280 } }, false]
]) {
    test.describe(`${name}, vidéo attachée`, () => {
        test.beforeEach(async ({ page }) => {
            await page.addInitScript(() => {
                localStorage.setItem('blunderdb.transcription.videoHeight', '900');
                // En panneau : sur le plateau la fente se replie et ne prend aucune place.
                localStorage.setItem('blunderdb.video.placement', 'panel');
            });
            await installWailsMock(page, openLibraryMock({ config }));
            await installTranscriptionEngine(page, { video: '/videos/match.mp4' });
            await page.goto('/');
            await page.locator('[data-testid="tab-transcription"]').click();
            await page.locator('#transcriptionPanel tbody tr').first().click();
            await expect(page.locator('#transcriptionPanel .video-slot')).toBeAttached();
            await page.keyboard.press('Digit3');
            await page.keyboard.press('Digit1');
            // Attachée, pas visible : une liste écrasée à zéro doit échouer sur
            // les assertions ci-dessous, qui la mesurent, non sur l'attente.
            await expect(page.locator('[data-testid="transcription-candidates"] tbody tr').first()).toBeAttached();
        });

        test('le panneau ne défile pas', async ({ page }) => {
            test.skip(!fits, 'les planchers cumulés dépassent la hauteur');
            expect(await overflow(page)).toBe(0);
        });

        test('la liste garde au moins deux lignes entières', async ({ page }) => {
            expect(await fullyVisibleRows(page)).toBeGreaterThanOrEqual(2);
        });

        test('la vidéo garde au moins cinq rem', async ({ page }) => {
            const box = await page.locator('#transcriptionPanel .video-slot').boundingBox();
            const rem = await page.evaluate(() => parseFloat(getComputedStyle(document.documentElement).fontSize));
            expect(box.height).toBeGreaterThanOrEqual(5 * rem - 1);
        });

        test('la liste garde son plancher de sept rem', async ({ page }) => {
            const box = await page.locator('[data-testid="transcription-candidates"]').boundingBox();
            const rem = await page.evaluate(() => parseFloat(getComputedStyle(document.documentElement).fontSize));
            expect(box.height).toBeGreaterThanOrEqual(7 * rem - 1);
        });
    });
}

/**
 * Les deux régimes que le plancher a retouchés : la palette empilée sous
 * 500 px, la grille à deux colonnes de 500 à 900 px. Largeurs de part et
 * d'autre des points de rupture, vidéo attachée, hauteur la plus rude.
 */
for (const [name, width, stacked] of [
    ['étroit (sous 500 px)', 380, true],
    ['intermédiaire (500 à 900 px)', 700, false]
]) {
    test.describe(`dock latéral ${name}, vidéo attachée`, () => {
        test.beforeEach(async ({ page }) => {
            await page.addInitScript(() => {
                localStorage.setItem('blunderdb.transcription.videoHeight', '900');
                // En panneau : sur le plateau la fente se replie et ne prend aucune place.
                localStorage.setItem('blunderdb.video.placement', 'panel');
            });
            await installWailsMock(page, openLibraryMock({ config: { GetPanelPosition: 'side', GetPanelWidth: width } }));
            await installTranscriptionEngine(page, { video: '/videos/match.mp4' });
            await page.goto('/');
            await page.locator('[data-testid="tab-transcription"]').click();
            await page.locator('#transcriptionPanel tbody tr').first().click();
            await expect(page.locator('#transcriptionPanel .video-slot')).toBeAttached();
            await page.keyboard.press('Digit3');
            await page.keyboard.press('Digit1');
            await expect(page.locator('[data-testid="transcription-candidates"] tbody tr').first()).toBeAttached();
        });

        test('le panneau ne défile pas', async ({ page }) => {
            expect(await overflow(page)).toBe(0);
        });

        test('la vidéo garde au moins cinq rem', async ({ page }) => {
            const box = await page.locator('#transcriptionPanel .video-slot').boundingBox();
            const rem = await page.evaluate(() => parseFloat(getComputedStyle(document.documentElement).fontSize));
            expect(box.height).toBeGreaterThanOrEqual(5 * rem - 1);
        });

        test('la liste garde son plancher de sept rem', async ({ page }) => {
            const box = await page.locator('[data-testid="transcription-candidates"]').boundingBox();
            const rem = await page.evaluate(() => parseFloat(getComputedStyle(document.documentElement).fontSize));
            expect(box.height).toBeGreaterThanOrEqual(7 * rem - 1);
        });

        test(stacked ? 'la palette est empilée sous la liste' : 'la palette est sous la liste, le Transcript à sa droite', async ({ page }) => {
            const palette = await page.locator('[data-testid="transcription-palette"]').boundingBox();
            const list = await page.locator('[data-testid="transcription-candidates"]').boundingBox();
            const transcript = await page.locator('.transcript-col').boundingBox();
            expect(palette.y).toBeGreaterThanOrEqual(list.y + list.height - 1);
            if (stacked) {
                expect(transcript.y).toBeGreaterThanOrEqual(palette.y + palette.height - 1);
            } else {
                expect(transcript.x).toBeGreaterThanOrEqual(palette.x + palette.width - 1);
                expect(Math.abs(transcript.y - palette.y)).toBeLessThanOrEqual(1);
            }
        });
    });
}
