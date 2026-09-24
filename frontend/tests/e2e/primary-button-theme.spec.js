/**
 * primary-button-theme.spec.js — suite de #402
 *
 * Le bouton principal d'une fenêtre (« Fermer » de la configuration, « Aller »
 * d'Aller à la position…) reste l'action qu'on repère, et reste lisible, dans
 * chaque thème. #359 l'avait peint encre sur fond d'encre : juste en clair, mais
 * en sombre c'était le seul aplat clair de la fenêtre. Il suit désormais le
 * schéma sombre, teinté de l'accent.
 *
 * Mesuré sur les couleurs CALCULÉES par le moteur, au repos et au survol :
 * - l'encre tient 4,5:1 sur le fond du bouton (WCAG AA) ;
 * - le bouton se distingue du bouton neutre voisin (fond ou filet) ;
 * - sous un schéma sombre, son fond est sombre.
 * Les thèmes et leur schéma sont lus dans themes.js, pas recopiés.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock, dismissHomeScreen } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';
import { THEMES } from '../../src/utils/themes.js';

/**
 * Luminance relative WCAG d'une couleur calculée : `rgb(…)`, `rgba(…)` ou
 * `color(srgb …)` — la forme que rend un `color-mix`.
 * @param {string} css
 */
function luminance(css) {
    const nums = css.match(/-?\d+(\.\d+)?(e-?\d+)?/g).map(Number);
    const channels = css.startsWith('color(') ? nums.slice(0, 3) : nums.slice(0, 3).map((c) => c / 255);
    const [r, g, b] = channels.map((c) => (c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4));
    return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

const contrast = (a, b) => {
    const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x);
    return (hi + 0.05) / (lo + 0.05);
};

const paint = (locator) =>
    locator.evaluate((el) => {
        const cs = getComputedStyle(el);
        return { bg: cs.backgroundColor, ink: cs.color, border: cs.borderTopColor };
    });

async function openGoTo(page) {
    await page.goto('/');
    await dismissHomeScreen(page);
    await page.locator('button[aria-label="Go To Position"]').click();
    await expect(page.locator('.modal-box')).toBeVisible();
}

async function checkPrimary(page, scheme) {
    const primary = page.locator('.modal-footer button.primary');
    const neutral = page.locator('.modal-footer button:not(.primary)');
    await page.mouse.move(0, 0);

    // Les boutons ont une transition de 0,2 s : on attend qu'elle soit finie.
    await expect.poll(async () => contrast((await paint(primary)).ink, (await paint(primary)).bg)).toBeGreaterThanOrEqual(4.5);
    const rest = await paint(primary);
    const other = await paint(neutral);
    expect(rest.bg !== other.bg || rest.border !== other.border, 'le bouton principal se confond avec le neutre').toBe(true);
    if (scheme === 'dark') expect(luminance(rest.bg), 'fond clair sous un schéma sombre').toBeLessThan(0.2);

    await primary.hover();
    await expect.poll(async () => (await paint(primary)).bg).not.toBe(rest.bg);
    await page.waitForTimeout(250);
    const hover = await paint(primary);
    expect(contrast(hover.ink, hover.bg), 'survol').toBeGreaterThanOrEqual(4.5);
}

for (const [theme, { scheme }] of Object.entries(THEMES)) {
    test(`le bouton principal est lisible et repérable dans le thème ${theme}`, async ({ page }) => {
        await installWailsMock(page, openLibraryMock({ config: { GetTheme: theme } }));
        await openGoTo(page);
        await expect(page.locator('html')).toHaveAttribute('data-scheme', scheme);
        await checkPrimary(page, scheme);
    });
}

test('le bouton principal suit le thème système quand le bureau est sombre', async ({ page }) => {
    await page.emulateMedia({ colorScheme: 'dark' });
    await installWailsMock(page, openLibraryMock({ config: { GetTheme: 'system' } }));
    await openGoTo(page);
    await expect(page.locator('html')).toHaveAttribute('data-scheme', 'dark');
    await checkPrimary(page, 'dark');
});
