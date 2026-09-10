/**
 * transcription-layout.spec.js — le contrat de disposition d'ADR-0048, en pixels.
 *
 * ## Pourquoi en pixels, et pas en présence dans le DOM
 *
 * Le panneau livré montait tout ce qu'il devait monter : une spec qui compte des
 * nœuds n'aurait rien vu. Ce qui était faux était géométrique — 354 px de
 * contrôles avant la première ligne de candidat dans une boîte de 168 px, et une
 * chaîne de hauteur cassée (`.draft` sans `flex: 1; min-height: 0`) qui rendait
 * inertes les deux `overflow: auto` déjà écrits, faisait de `.tab-content` le
 * seul conteneur qui défile, et faisait donc sauter le panneau entier à chaque
 * Action validée, le `scrollIntoView` du Cursor ne trouvant pas de boîte bornée.
 *
 * Le patron est celui d'`eval-panel-no-scroll.spec.js`, y compris son principe :
 * « a property nobody measures after three layout changes is not a property ».
 *
 * ## Les trois assertions, et laquelle compte
 *
 * 1. `.tab-content` ne défile pas — l'anti-régression du défaut lui-même.
 * 2. Cinq lignes de candidats sont ENTIÈREMENT dans le rectangle visible de la
 *    liste (`boundingBox`, jamais `count()`) : une ligne montée mais rognée ne
 *    compte pas, sinon on affirme une présence et non le contrat.
 * 3. Rien ne s'intercale entre les cases du jet et la première ligne de
 *    candidats. C'est la seule des trois qui mesure la RÈGLE ; sans elle on
 *    satisfait les deux autres en rétrécissant le triangle et en le remettant
 *    entre les deux, c'est-à-dire par la faute que l'ADR corrige.
 *
 * L'assertion 3 tolère un écart NÉGATIF : en dock bas la palette est à côté de
 * la liste, donc la première ligne commence plus haut que le bas des dés. Ce qui
 * est interdit est l'intercalation, pas la coexistence.
 *
 * ## Les deux tailles
 *
 * Dock bas à sa hauteur plancher (280 px sur cet onglet) et dock latéral à sa
 * largeur par défaut (420 px). Pas de balayage : un contrat mesuré à toutes les
 * tailles devient une distribution, et personne ne saura dire dans six mois à
 * quel point de rupture le rouge est légitime.
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
        await openDraft(page, { GetPanelPosition: 'bottom', GetPanelHeight: 280 });
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
 * Le troisième régime : un dock latéral élargi, où le Transcript vient occuper
 * le blanc que le triangle laissait à sa droite.
 *
 * Il a son point de rupture à lui — 500 px —, et c'est une mesure : le triangle
 * demande 211 px et un Transcript 250 px, soit 485 px de panneau. La spec le
 * vérifie à 520, le premier cran où l'appariement est censé tenir. Sans elle,
 * régler le seuil sur l'impression laisserait passer un Transcript dont la
 * seconde colonne est rognée — ce qui n'est pas un Transcript rétréci mais autre
 * chose (ADR-0048 décision 6, l'argument qui a sorti le texte `.mat` d'ici).
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
