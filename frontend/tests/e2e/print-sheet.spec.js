/**
 * Les feuilles imprimées (listes de positions, feuille de match) sont mises en page par leur
 * propre script, dans le navigateur : c'est ici seulement qu'un ArkID surimprimé par les pips ou
 * l'analyse se voit. Les documents viennent des renderers Go (TestPrintFixtures, dans
 * pkg/blunderdb/printsheet) et sont assez denses pour descendre au corps plancher.
 *
 * Tenu : l'ArkID garde toute sa largeur et rien ne le chevauche ; aucune feuille ne déborde ;
 * une entrée longue continue sur la page où elle commence au lieu de la laisser presque vide ;
 * au verso, aucun recto ne reste vide quand une réponse se prolonge ; le code de réimport d'une
 * tranche est sur la page de sa première position.
 */

import { test, expect } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';

const fixture = (name) => readFileSync(fileURLToPath(new URL(`./fixtures/printsheet/${name}`, import.meta.url)), 'utf8');

/** Mesures prises après le script de mise en page. */
function measure() {
    const hits = (a, b) => a.left < b.right - 0.5 && b.left < a.right - 0.5 && a.top < b.bottom - 0.5 && b.top < a.bottom - 0.5;
    const overlaps = [];
    for (const nt of document.querySelectorAll('.nt')) {
        if (nt.scrollWidth > nt.clientWidth) overlaps.push(`débord ${nt.textContent}`);
        const range = document.createRange();
        range.selectNodeContents(nt);
        const text = range.getBoundingClientRect();
        for (let sib = nt.nextElementSibling; sib; sib = sib.nextElementSibling) {
            for (const box of sib.getClientRects()) if (hits(text, box)) overlaps.push(`chevauche ${nt.textContent}`);
        }
    }
    const sheets = [...document.querySelectorAll('.sheet')].map((s) => {
        const c = getComputedStyle(s);
        const avail = s.clientHeight - parseFloat(c.paddingTop) - parseFloat(c.paddingBottom);
        const dl = s.querySelector('.dl');
        return {
            fs: Number(s.dataset.fs),
            positions: s.querySelectorAll('.de:not(.cont) .nt').length,
            lines: dl ? dl.children.length : 0,
            fill: s.firstElementChild.scrollHeight / avail,
            verso: !!s.querySelector('[data-verso]')
        };
    });
    return { overlaps, overflow: document.querySelectorAll('[data-overflow]').length, sheets };
}

async function render(page, name) {
    test.setTimeout(60000);
    await page.setContent(fixture(name), { waitUntil: 'load' });
    return page.evaluate(measure);
}

for (const name of ['list-analysis.html', 'list-plain.html', 'list-long.html', 'list-verso-long.html', 'list-codes.html', 'match.html']) {
    test(`${name} : l'ArkID n'est ni comprimé ni chevauché, rien ne déborde`, async ({ page }) => {
        const r = await render(page, name);
        expect(r.overlaps.slice(0, 5)).toEqual([]);
        expect(r.overflow).toBe(0);
    });
}

test('une entrée longue commence sur la page en cours au lieu de la laisser presque vide', async ({ page }) => {
    const r = await render(page, 'list-long.html');
    const thin = r.sheets.slice(0, -1).filter((s) => s.fill < 0.6);
    expect(thin).toEqual([]);
});

test('au verso, une réponse qui se prolonge ne laisse aucun recto vide', async ({ page }) => {
    const r = await render(page, 'list-verso-long.html');
    expect(r.sheets.length % 2).toBe(0);
    const empty = r.sheets.filter((s) => !s.verso && s.lines === 0);
    expect(empty).toEqual([]);
});

test("le code de réimport d'une tranche est sur la page de sa première position", async ({ page }) => {
    await render(page, 'list-codes.html');
    const placed = await page.evaluate(() =>
        [...document.querySelectorAll('.qrb[data-from]')].map((q) => {
            const sheet = q.closest('.sheet');
            const first = [...sheet.querySelectorAll('.de:not(.cont) .no')].some((n) => n.textContent === q.dataset.from);
            return { from: q.dataset.from, first };
        })
    );
    expect(placed.map((p) => p.from)).toEqual(['1', '146', '291']);
    expect(placed.filter((p) => !p.first)).toEqual([]);
});
