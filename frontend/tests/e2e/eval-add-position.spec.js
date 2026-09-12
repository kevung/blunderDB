/**
 * eval-add-position.spec.js — « Ajouter à la base » in the Eval panel (#399).
 *
 * The button is a labelled one, at the head of the badge strip, precisely so
 * it is seen; a label costs width, and the strip is one line (ADR-0020 rule 8,
 * ADR-0021). So the claim is measured where it is tightest: at blunderDB's
 * default window width (1024 px, config.go), in French and in German, the
 * longest of the labels.
 *
 * The two runs also cover the two reasons the button is disabled on the
 * default Eval board: no database open (French), and a database open with the
 * default bearoff, whose top side has borne everything off (German).
 */

import { test, expect } from '@playwright/test';
import { dismissHomeScreen, installWailsMock } from './helpers/wailsMock.js';
import { epcResultA, openLibraryMock } from './helpers/fixtures.js';
import fr from '../../src/i18n/locales/fr.json' with { type: 'json' };
import de from '../../src/i18n/locales/de.json' with { type: 'json' };

// A pure bearoff read off the exact table: the strip carries the regime badge,
// the engine link and the Défi toggle behind the button.
const raceEpc = {
    ...epcResultA,
    top: {
        all_in_home: true,
        checker_count: 15,
        epc: { epc: 71.2, pipCount: 66, wastage: 5.2, meanRolls: 11.867, stdDev: 2.4 }
    },
    race: {
        regime: 'exact',
        on_roll: 0,
        source_checkers: 6,
        win_prob: 0.6421,
        money: { cube_state: 'centered', cubeless: 0.284, no_double: 0.301, double_take: 0.42, double_pass: 1.0, verdict: 'double_take' }
    }
};

async function openEval(page, mock) {
    await installWailsMock(page, mock);
    await page.addInitScript((epcFixture) => {
        window.__epcFixture = epcFixture;
    }, raceEpc);
    await page.addInitScript(() => {
        Object.defineProperty(window.go.database.Database, 'ComputeEPCFromPosition', {
            get: () => () => Promise.resolve(window.__epcFixture ?? null),
            configurable: true
        });
    });

    await page.goto('/');
    await expect(page.locator('[data-testid="status-bar"]')).toBeVisible({ timeout: 8000 });
    await dismissHomeScreen(page);
    await page.click('[data-testid="tab-epc"]');
    await expect(page.locator('[data-testid="tab-epc"]')).toHaveClass(/active/);
    await expect(page.locator('.epc-panel .badges-strip .badge')).toBeVisible({ timeout: 4000 });
}

/** Every child of the strip, as the rectangles the browser laid out. */
async function stripLayout(page) {
    return page.locator('.epc-panel .badges-strip').evaluate((strip) => ({
        firstIsButton: strip.firstElementChild?.classList.contains('add-position') ?? false,
        children: [...strip.children].map((el) => {
            const r = el.getBoundingClientRect();
            return { cls: el.className, top: r.top, bottom: r.bottom, left: r.left, right: r.right };
        }),
        strip: (() => {
            const r = strip.getBoundingClientRect();
            return { left: r.left, right: r.right };
        })()
    }));
}

function expectOneLine(layout) {
    expect(layout.firstIsButton).toBe(true);
    expect(layout.children.length).toBeGreaterThanOrEqual(4);
    // One line: every pill's vertical centre within a couple of pixels of the others.
    const centres = layout.children.map((c) => (c.top + c.bottom) / 2);
    expect(Math.max(...centres) - Math.min(...centres)).toBeLessThanOrEqual(3);
    // Left to right in DOM order, and inside the strip.
    for (let i = 1; i < layout.children.length; i++) {
        expect(layout.children[i].left).toBeGreaterThanOrEqual(layout.children[i - 1].right - 1);
    }
    expect(layout.children[0].left).toBeGreaterThanOrEqual(layout.strip.left - 1);
    expect(layout.children[layout.children.length - 1].right).toBeLessThanOrEqual(layout.strip.right + 1);
}

test.describe("at blunderDB's default window width", () => {
    test.use({ viewport: { width: 1024, height: 768 } });

    test('French, no database: the strip is one line and the button says to open a database', async ({ page }) => {
        await openEval(page, { config: { GetLanguage: 'fr' } });

        const button = page.locator('.epc-panel .badges-strip .add-position');
        await expect(button).toHaveText(fr.eval.addPosition);
        await expect(button).toBeDisabled();
        await expect(button).toHaveAttribute('title', fr.eval.addPositionNoDatabase);

        expectOneLine(await stripLayout(page));
    });

    test('German, a database open, the default board: one line, and the refusal as the reason', async ({ page }) => {
        await openEval(page, openLibraryMock({ config: { GetLanguage: 'de' } }));

        const button = page.locator('.epc-panel .badges-strip .add-position');
        await expect(button).toHaveText(de.eval.addPosition);
        await expect(button).toBeDisabled();
        await expect(button).toHaveAttribute('title', de.status.invalidP2BorneOff);

        expectOneLine(await stripLayout(page));
    });
});
