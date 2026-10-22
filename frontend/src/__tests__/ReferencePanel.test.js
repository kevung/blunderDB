/**
 * The reference panel: mounted on its own (no Wails binding), each tab shows its tables,
 * the tab chosen survives a remount (TabbedPanel unmounts a left panel), and the Neil grid
 * shows the shortcut's estimate against the Kazaross-XG2 MET.
 */
import { describe, test, expect, afterEach, beforeEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import { get } from 'svelte/store';
import { must } from './helpers/must.js';
import ReferencePanel from '../components/ReferencePanel.svelte';
import { referenceSectionStore, REFERENCE_SECTIONS } from '../stores/referenceStore.js';
import { tableData as metTable } from '../stores/metTable';

/** @param {Element} container @param {string} id @returns {HTMLElement | null} */
const byTestId = (container, id) => container.querySelector(`[data-testid="${id}"]`);
/** @param {Element} container @param {string} id @returns {HTMLElement} */
const get1 = (container, id) => must(byTestId(container, id));

describe('ReferencePanel', () => {
    beforeEach(() => referenceSectionStore.set('met'));
    afterEach(() => cleanup());

    test('mounts on the MET: 15 × 15 values under a header of 1..15', () => {
        const { container } = render(ReferencePanel);
        expect(byTestId(container, 'reference-section-met')).not.toBeNull();
        const table = must(container.querySelector('.reference-body table'));
        const headers = [...table.querySelectorAll('thead th')].map((th) => th.textContent);
        expect(headers.slice(1)).toEqual(Array.from({ length: 15 }, (_, i) => String(i + 1)));
        expect(table.querySelectorAll('tbody td').length).toBe(225);
        expect(must(table.querySelector('tbody td')).textContent).toBe('50.0');
    });

    test('one tab button per section, the active one selected', () => {
        const { container } = render(ReferencePanel);
        const tabs = /** @type {HTMLElement[]} */ ([...container.querySelectorAll('[role="tab"]')]);
        expect(tabs.map((b) => b.dataset.testid)).toEqual(REFERENCE_SECTIONS.map((s) => `reference-tab-${s}`));
        expect(tabs.filter((b) => b.getAttribute('aria-selected') === 'true').map((b) => b.dataset.testid)).toEqual(['reference-tab-met']);
    });

    test('clicking a tab shows its section and records it in the store', async () => {
        const { container } = render(ReferencePanel);
        await fireEvent.click(get1(container, 'reference-tab-takePoints'));
        expect(get(referenceSectionStore)).toBe('takePoints');
        expect(byTestId(container, 'reference-section-takePoints')).not.toBeNull();
        // Cube 2 and cube 4, each live and last side by side.
        const tables = container.querySelectorAll('.reference-body table');
        expect(tables.length).toBe(4);
        expect(must(tables[0].querySelector('thead th:nth-child(2)')).textContent).toBe('2');
        expect(must(tables[2].querySelector('thead th:nth-child(2)')).textContent).toBe('3');

        await fireEvent.click(get1(container, 'reference-tab-gammonValues'));
        expect(container.querySelectorAll('.reference-body table').length).toBe(3);
    });

    test('the tab chosen before mounting is the one shown (a command opens the panel on it)', async () => {
        referenceSectionStore.set('aggregated');
        const { container } = render(ReferencePanel);
        await tick();
        expect(byTestId(container, 'reference-section-aggregated')).not.toBeNull();
        expect(get1(container, 'reference-tab-aggregated').getAttribute('aria-selected')).toBe('true');
        expect(get1(container, 'reference-section-aggregated').textContent).toContain('Per cell: take point ×2 and ×4');
    });

    test('an aggregated cell holds tp2, tp4 and the three gammon values; a missing value is a dash', () => {
        referenceSectionStore.set('aggregated');
        const { container } = render(ReferencePanel);
        const cells = container.querySelectorAll('.reference-body tbody td');
        expect(cells.length).toBe(64);
        // At 2-away vs 2-away, tp4 does not exist (it starts at 3-away) and neither do gv2, gv4.
        const lines = [...get1(container, 'aggregated-2-2').children].map((d) => d.textContent);
        expect(lines).toHaveLength(3);
        expect(lines[1]).toBe('— / —');
        expect(lines[2]).toMatch(/^\d\.\d\d · — · —$/);
        const full = [...get1(container, 'aggregated-5-5').children].map((d) => d.textContent);
        expect(full.every((l) => !l.includes('—'))).toBe(true);
    });

    test('Neil: the anchor numbers, the formula in the grid and its gap to the MET', () => {
        referenceSectionStore.set('neil');
        const { container } = render(ReferencePanel);
        expect(get1(container, 'neil-number-19').textContent).toBe('3½');
        expect(get1(container, 'neil-number-8').textContent).toBe('6');
        // Leader 3-away, trailer 5-away: 50 + 2 × 8 = 66, against the MET's 64.8.
        expect(get1(container, 'neil-3-5').firstElementChild?.textContent).toBe('66.0');
        expect(get1(container, 'neil-gap-3-5').textContent).toBe(`+${(66 - metTable[2][4]).toFixed(1)}`);
        // Leader 2-away: the shortcut does not hold, the MET is shown and no gap.
        const fallback = get1(container, 'neil-2-7');
        expect(fallback.firstElementChild?.textContent).toBe(metTable[1][6].toFixed(1));
        expect(byTestId(container, 'neil-gap-2-7')).toBeNull();
        // A leader that does not lead has no cell.
        expect(byTestId(container, 'neil-7-7')).toBeNull();
        expect(byTestId(container, 'neil-9-4')).toBeNull();
    });
});
